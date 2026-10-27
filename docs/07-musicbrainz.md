# 07 — Интеграция с MusicBrainz

## 1. Правила API

Документация: <https://musicbrainz.org/doc/MusicBrainz_API>, <https://musicbrainz.org/doc/MusicBrainz_API/Rate_Limiting>.

- База: `https://musicbrainz.org/ws/2/`, ответ в JSON — параметр `fmt=json`.
- **Rate limit: в среднем 1 запрос/сек с IP.** При превышении — `503`; все запросы могут отклоняться.
- **Обязательный `User-Agent`**: `AppName/Version ( contact-url-or-email )`, иначе возможна блокировка.
- Аутентификация для чтения не нужна.
- Browse-запросы: `limit` ≤ 100, пагинация через `offset`, в ответе есть общее количество (`release-group-count`).

## 2. Используемые эндпоинты

| Задача | Запрос |
|--------|--------|
| Поиск исполнителя | `GET /artist?query=<lucene>&limit=10&fmt=json` |
| Lookup исполнителя | `GET /artist/{mbid}?fmt=json` |
| Релиз-группы исполнителя | `GET /release-group?artist={mbid}&inc=artist-credits&limit=100&offset={n}&fmt=json` |
| Обложка | `https://coverartarchive.org/release-group/{mbid}/front-250` (редирект; используется как URL, backend не скачивает) |

**Поиск.** Пользовательский ввод экранируется (спецсимволы Lucene `+ - && || ! ( ) { } [ ] ^ " ~ * ? : \ /`) и отправляется как `artist:"<input>"` с запасным вариантом без кавычек, если результатов нет.

**Browse релиз-групп** возвращает поля: `id`, `title`, `primary-type`, `secondary-types`, `first-release-date`, `disambiguation`. Фильтр по `type=` на стороне API **не используем** — загружаем всё, фильтруем у себя (ADR-2).

> Browse по `artist=` возвращает все релиз-группы, где исполнитель присутствует в artist credit — включая совместные и featuring. С `inc=artist-credits` в ответ попадает сам credit, по которому вычисляется роль (§5).

## 3. Клиент (`adapter/musicbrainz`)

```
Client
├── http.Client{Timeout: 15s}
├── limiter *rate.Limiter   // 1 rps, burst 1, общий для всего процесса
├── userAgent string
└── retry: 503/429/5xx/network → до 4 попыток, backoff 2s·2^n + jitter, уважать Retry-After
```

- Каждый вызов: `limiter.Wait(ctx)` → запрос → декодирование в приватные DTO → маппинг в `domain`.
- `404` → `domain.ErrNotFound`; исчерпаны повторы → `domain.ErrUpstreamUnavailable`.
- MBID-редиректы (слитые исполнители): lookup вернёт другой `id` — обновляем `artists.mbid` через merge-процедуру (этап 2 roadmap).
- Тесты: `httptest.Server` с фикстурами реальных JSON-ответов в `testdata/`.

Приоритеты: поиск из UI интерактивен и должен отвечать быстро, а синк может ждать. Лимитер реализуется с двумя очередями (интерактивная и фоновая): при наличии интерактивного запроса токен отдаётся ему. Простой вариант для MVP — один лимитер, синк ставит паузу ≥ 1 с между исполнителями.

## 4. Алгоритм синхронизации исполнителя

```
SyncArtist(mbid, baseline bool):
  remote := catalog.ListReleaseGroups(mbid)           // все страницы
  known  := releases.KnownMBIDs(mbid)
  fresh  := remote \ known

  tx:
    upsert release_groups(remote)                     // обновляем title/даты/типы у известных
    for rg in fresh:
        role := domain.RoleOf(rg.ArtistCredit, mbid)   // §5
        news := !isStale(rg, now) && !(baseline && rg.first_release_on == nil)
        link artist_release_groups(role, is_baseline = baseline && !news)
        if !news:                       continue      // FR-3.4
        FanOutNotifications(sub, rg, role)            // фильтры типов и featuring — см. 08-notifications.md
    artists.last_synced_at = now
```

```
isStale(rg, now):
    rg.first_release_on != nil && rg.first_release_on < now - settings.new_release_days
```

- `baseline = artists.last_synced_at IS NULL` — первая синхронизация исполнителя в системе.
- Если исполнитель уже был в системе, а пользователь подписался только что, baseline для него определяется по `subscriptions.baseline_at`: релиз-группы с `artist_release_groups.discovered_at < baseline_at` ему не уведомляются (это естественно — уведомления создаются только в момент обнаружения).
- Релиз без даты (`first_release_date` пуст) не считается stale — часто это свежедобавленный анонс. В baseline он всё же история: у старых исполнителей недатированных релиз-групп много.
- В baseline релизы с датой внутри окна `settings.new_release_days` уведомляются как обычные находки — лента сразу показывает свежие релизы нового исполнителя.
- Анонс = `first_release_on > today`. Фильтруется настройкой `notify_announcements` при fan-out.

### Ошибки и устойчивость
- Ошибка по одному исполнителю не прерывает прогон: задача `sync_artist` уходит в повтор, остальные продолжаются.
- Задачи прогона имеют `dedup_key = sync_artist:<run_id>:<mbid>` — повторный запуск планировщика не плодит дубли.
- Частично загруженный список (ошибка на странице N) — транзакция не начинается, исполнитель повторяется целиком.

## 5. Определение роли в artist credit

Artist credit в ответе — массив `[{artist: {id, name}, name, joinphrase}]`; `joinphrase` — текст **после** элемента.

```
RoleOf(credit, mbid):
  i := индекс элемента с artist.id == mbid
  if i == 0:                       return main
  phrase := lower(credit[i-1].joinphrase)          // фраза перед исполнителем
  if phrase ~ /\b(feat\.?|ft\.?|featuring|with|duet with)\b/: return featured
  if какая-либо предыдущая фраза ~ featured-паттерн: return featured   // "A feat. B & C" → C тоже featured
  return collaboration                                   // " & ", " and ", " x ", ", ", " / ", " vs. "
```

- Функция чистая, живёт в `domain/credit.go`, покрыта табличными тестами с реальными примерами credit из MusicBrainz.
- Если исполнителя нет в credit (бывает при связях через alias/merge) — роль `featured` (консервативно, не уведомлять без флага).
- Паттерны вынесены в конфигурируемый список — MusicBrainz-редакторы используют разные написания.

## 6. Оценка нагрузки

| Исполнителей | Запросов (в среднем 1,3 стр.) | Время при 1 rps |
|--------------|-------------------------------|-----------------|
| 500 | ~650 | ~11 мин |
| 2 000 | ~2 600 | ~45 мин |

Оптимизация на будущее: синхронизировать «активных» исполнителей ежедневно, а давно не выпускавших релизы (или распавшихся, `life-span.ended`) — раз в неделю.
