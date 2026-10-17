# 09 — Roadmap

## Этап 0. Каркас ✅
- [x] Монорепо: `backend/`, `frontend/`, `docker-compose.yml`, `Makefile`
- [x] Backend: `cmd/tracker` (serve, migrate), config, slog, chi, `/healthz`, `/readyz`, golang-migrate (встроенные миграции), sqlc
- [x] Frontend: Vite + React + TS, роутер, `HttpClient`
- [x] Дизайн-токены, шрифты, текстуры, ночная тема; Storybook
- [x] UI-кит: Panel, PushButton, ToggleSwitch, RotarySelector, Led, LcdDisplay, EngravedInput
- [x] Layout «стойки», эталонная страница «Настройки» (локальное состояние)
- [x] CI: lint + test для обеих частей

## Этап 1. MusicBrainz и подписки ✅
- [x] `adapter/musicbrainz`: клиент, rate limiter, retry, фикстуры
- [x] Use cases: `SearchArtists`, `Subscribe`, `Unsubscribe`, `ListSubscriptions`
- [x] Миграции: `settings`, `artists`, `subscriptions`
- [x] UI: поиск исполнителя (тюнер), список подписок (картотека)
- Отложено: `GET /catalog/artists/{mbid}`, `GET/PATCH /subscriptions/{mbid}` (этапы 2–3); задача `sync_artist(baseline)` при подписке — этап 2

## Этап 2. Синхронизация ✅
- [x] Таблицы `release_groups`, `artist_release_groups`, `jobs`, `sync_runs`
- [x] Очередь задач на `SKIP LOCKED`, воркеры
- [x] `domain.RoleOf` — классификация main/collaboration/featured
- [x] `SyncArtist` с baseline, stale-фильтром и ролями
- [x] Планировщик + ручной запуск
- [x] UI: каталог исполнителя, кнопка «Проверить сейчас»
- Отложено: CLI `tracker sync --artist`; пересчёт роли и удаление связей у уже известных релиз-групп (сейчас роль фиксируется при обнаружении)

## Этап 3. Фильтры и уведомления ✅
- [x] Глобальные фильтры, override в подписке (типы + `include_featured`), `ReleaseFilter`
- [x] Fan-out, `notifications` (outbox)
- [x] `DigestSchedule` (daily/weekly, часовой пояс), планировщик дайджеста
- [x] Email-дайджест (SMTP, шаблоны), тестовое письмо, mailpit в dev
- [x] Atom/RSS-фиды
- [x] UI: настройки (панель усилителя), лента новинок (полка пластинок), фильтр подписки на странице исполнителя

## Этап 4. Импорт ✅
- [x] Парсеры txt/csv (имя, MBID или ссылка MusicBrainz; CSV с `,`/`;`, колонки `name`, `mbid`, `include_featured`)
- [x] Асинхронная обработка (задача `import_item` на строку), эвристика уверенности `domain.AutoMatch`
- [x] UI: загрузка (кассетная дека), прогресс, ручное разрешение (выбор кандидата, свой MBID, пропуск)
- `GET /imports/{id}/items?status=` принимает список статусов через запятую

## Этап 5. Прод
- [ ] Docker-образы, деплой стеком в Portainer

## Дальше
- Адаптивная частота синхронизации
- Обработка merge исполнителей в MusicBrainz
- Featuring на уровне треков (обход recordings)
