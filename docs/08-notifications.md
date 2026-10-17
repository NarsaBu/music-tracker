# 08 — Уведомления

## 1. Fan-out

Вызывается внутри транзакции синхронизации для каждой новой релиз-группы исполнителя.

```
FanOutNotifications(sub, rg, role):
  if rg.discovered_at < sub.baseline_at:            return
  filter := domain.EffectiveFilter(settings, sub)    // override подписки поверх глобальных
  if !filter.Matches(rg, role):                      return
  if rg.IsAnnouncement(now) && !settings.notify_announcements: return
  for ch in enabledChannels(settings):               // email (если задан email_to), rss
      notifications.Insert(rg, sub.artist, role, ch) // ON CONFLICT (release_group_mbid, channel) DO NOTHING
```

`ReleaseFilter.Matches(rg, role)` (domain):
- `role == featured` → только если `include_featured`;
- `primary_type ∈ primary_types` (релиз-группа без primary type проходит только если разрешён `other`);
- `secondary_types ∩ excluded_secondary_types = ∅`.

Совместный релиз двух отслеживаемых исполнителей даёт одно уведомление на канал (уникальный индекс); `found_via` — тот, кто обработан первым, с приоритетом роли `main`.

## 2. Каналы

Каждый канал — реализация порта:

```go
type Notifier interface {
    Channel() domain.Channel
    SendDigest(ctx context.Context, items []domain.NotificationItem) error
}
```

Добавление Telegram/Webhook = новый адаптер + значение enum + регистрация в composition root. Use cases не меняются (OCP).

### 2.1 Email-дайджест

**Расписание.** Синхронизация и дайджест независимы: синк ежедневный, дайджест — по настройке пользователя.

| `digest_frequency` | Отправка |
|--------------------|----------|
| `daily` | каждый день в `digest_time` (`timezone`) |
| `weekly` | в `digest_weekday` в `digest_time` (`timezone`) |

- `domain.DigestSchedule.Next(after time.Time) time.Time` — чистая функция, покрыта тестами (переход на летнее время, смена часового пояса).
- `next_digest_at` пересчитывается при изменении настроек и после каждой отправки.
- Планировщик раз в минуту проверяет `now() >= next_digest_at` и ставит задачу `send_digest` с `dedup_key = send_digest:<next_digest_at>`.
- Если синк ещё идёт в момент отправки дайджеста, то, что не успело попасть, уйдёт в следующий — допустимо. Время синка по умолчанию (06:00 UTC) раньше времени дайджеста.

**Отправка.**
```sql
SELECT … FROM notifications
WHERE channel = 'email' AND status = 'pending' AND next_attempt_at <= now()
ORDER BY created_at
FOR UPDATE SKIP LOCKED;
```
- Пусто → письмо не отправляется, `next_digest_at` сдвигается (FR-5.4).
- Иначе одно письмо, в той же транзакции все строки → `sent`, `settings.last_digest_at = now()`.
- Ошибка SMTP → `attempts++`, `next_attempt_at = now() + 1m·2^attempts`, после 8 попыток → `failed` (видно в UI).
- Шаблоны `html/template` + `text/template` (multipart/alternative). Разделы: «Вышло», «Анонсы», «Гостевое участие» (featured); внутри — группировка по исполнителю, сортировка по дате.
- Тема: `Новые релизы: Radiohead, Björk и ещё 3` (weekly: `за неделю`).
- Окно «at-least-once»: падение процесса между отправкой SMTP и коммитом даст повторное письмо — допустимо.

### 2.2 RSS / Atom

- Отдельной доставки нет: уведомление `channel='rss'` создаётся сразу со статусом `sent`; лента обновляется сразу после синка.
- `GET /feeds/{token}.atom|.rss` → `BuildFeed(token)` → последние 100 записей.
- Элемент фида:
  - `id`/`guid`: `urn:musicbrainz:release-group:<mbid>` (стабилен — ридер не покажет дубль);
  - `title`: `Radiohead — New Album (Album, 2026-11-07)`; для featured — префикс `[feat.]`;
  - `link`: страница релиз-группы на musicbrainz.org;
  - `updated`/`pubDate`: `notifications.created_at`;
  - `content`: HTML с обложкой Cover Art Archive, типами и датой.
- Кэширование: `ETag` = хэш `max(created_at)` + count; `Cache-Control: max-age=900`.
- Токен — 32 случайных байта в base64url; перевыпуск инвалидирует старую ссылку.

## 3. Наблюдаемость

Метрики Prometheus:
- `notifications_created_total{channel,role}`
- `notifications_sent_total{channel}` / `notifications_failed_total{channel}`
- `notifications_pending{channel}` (gauge)
- `digest_sent_total`, `digest_items` (histogram)
- `musicbrainz_requests_total{endpoint,status}`, `musicbrainz_request_duration_seconds`
- `sync_run_duration_seconds`, `sync_new_release_groups_total`
