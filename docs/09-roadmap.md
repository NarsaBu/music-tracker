# 09 — Roadmap

## Этап 0. Каркас ✅
- [x] Монорепо: `backend/`, `frontend/`, `docker-compose.yml`, `Makefile`
- [x] Backend: `cmd/tracker` (serve, migrate, hash-password), config, slog, chi, `/healthz`, `/readyz`, golang-migrate (встроенные миграции), sqlc
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
- Отложено: метрики уведомлений из [08 §3](08-notifications.md#3-наблюдаемость) — вместе с Prometheus на этапе 5

## Этап 4. Импорт
- [ ] Парсеры txt/csv
- [ ] Асинхронная обработка, эвристика уверенности
- [ ] UI: загрузка (кассетная дека), прогресс, ручное разрешение

## Этап 5. Прод
- [ ] Вход по паролю, cookie-сессия, CSRF, rate limit на вход
- [ ] Метрики Prometheus, дашборд
- [ ] Docker-образы, деплой
- [ ] e2e-тесты Playwright

## Дальше
- Telegram / webhook каналы
- Импорт из Last.fm / Spotify
- Адаптивная частота синхронизации
- Обработка merge исполнителей в MusicBrainz
- Featuring на уровне треков (обход recordings)
