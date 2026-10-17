# 09 — Roadmap

## Этап 0. Каркас
- [ ] Монорепо: `backend/`, `frontend/`, `docker-compose.yml`, `Makefile`
- [ ] Backend: `cmd/tracker`, config, slog, chi, `/healthz`, golang-migrate, sqlc
- [ ] Frontend: Vite + React + TS, роутер, `HttpClient`
- [ ] Дизайн-токены, шрифты, текстуры; Storybook
- [ ] UI-кит: Panel, PushButton, ToggleSwitch, RotarySelector, Led, LcdDisplay, EngravedInput
- [ ] Layout «стойки», эталонная страница для утверждения стиля
- [ ] CI: lint + test для обеих частей

## Этап 1. MusicBrainz и подписки
- [ ] `adapter/musicbrainz`: клиент, rate limiter, retry, фикстуры
- [ ] Use cases: `SearchArtists`, `Subscribe`, `Unsubscribe`, `ListSubscriptions`
- [ ] Миграции: `settings`, `artists`, `subscriptions`
- [ ] UI: поиск исполнителя (тюнер), список подписок (картотека)

## Этап 2. Синхронизация
- [ ] Таблицы `release_groups`, `artist_release_groups`, `jobs`, `sync_runs`
- [ ] Очередь задач на `SKIP LOCKED`, воркеры
- [ ] `domain.RoleOf` — классификация main/collaboration/featured
- [ ] `SyncArtist` с baseline, stale-фильтром и ролями
- [ ] Планировщик + ручной запуск
- [ ] UI: каталог исполнителя, кнопка «Проверить сейчас»

## Этап 3. Фильтры и уведомления
- [ ] Глобальные фильтры, override в подписке (типы + `include_featured`), `ReleaseFilter`
- [ ] Fan-out, `notifications` (outbox)
- [ ] `DigestSchedule` (daily/weekly, часовой пояс), планировщик дайджеста
- [ ] Email-дайджест (SMTP, шаблоны), тестовое письмо, mailpit в dev
- [ ] Atom/RSS-фиды
- [ ] UI: настройки (панель усилителя), лента новинок (полка пластинок)

## Этап 4. Импорт
- [ ] Парсеры txt/csv
- [ ] Асинхронная обработка, эвристика уверенности
- [ ] UI: загрузка (кассетная дека), прогресс, ручное разрешение

## Этап 5. Прод
- [ ] Вход по паролю, cookie-сессия, CSRF, rate limit на вход
- [ ] Ночная тема
- [ ] Метрики Prometheus, дашборд
- [ ] Docker-образы, деплой
- [ ] e2e-тесты Playwright

## Дальше
- Telegram / webhook каналы
- Импорт из Last.fm / Spotify
- Адаптивная частота синхронизации
- Обработка merge исполнителей в MusicBrainz
- Featuring на уровне треков (обход recordings)
