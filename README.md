# Music Tracker

Сервис для отслеживания новых релизов любимых исполнителей через [MusicBrainz](https://musicbrainz.org/).

- Подписка на исполнителей: поиск через MusicBrainz API или импорт из файла.
- Ежедневная проверка новых релизов с фильтром по типам (альбомы, синглы, EP, live, компиляции и т.д.) и опциональным учётом featuring.
- Уведомления: email-дайджест (ежедневно или еженедельно) и персональная RSS/Atom-лента.
- Однопользовательский режим (личный инстанс), интерфейс в стиле скевоморфизма («Hi-Fi стойка»).

## Стек

| Слой     | Технологии                                        |
|----------|---------------------------------------------------|
| Backend  | Go, PostgreSQL, Clean Architecture                |
| Frontend | React + TypeScript (Vite), MVC                    |
| Внешнее  | MusicBrainz Web Service v2, Cover Art Archive, SMTP |

## Быстрый старт

Нужны Go 1.26+, Node 24+, Docker.

```bash
make setup          # зависимости + .env из .env.example (укажите MB_USER_AGENT)
make db             # postgres + mailpit в docker
make run-backend    # API на :8080, миграции применяются при старте
make run-frontend   # UI на http://localhost:5173
```

Весь стек в Docker: `make up` → UI на http://localhost:8080, письма — http://localhost:8025.

## Деплой в Portainer

1. `make release V=v0.1.0` — тег на HEAD и пуш в GitHub.
2. Portainer → Stacks → Add stack → **Repository**: URL репозитория, reference `refs/tags/v0.1.0`, compose path `docker-compose.yml`.
3. Environment variables → загрузить `.env.example` и заполнить: `POSTGRES_PASSWORD`, `MB_USER_AGENT`, `PUBLIC_BASE_URL` (адрес из CNAME, например `https://music.example.com`), `VERSION`, SMTP.
4. Deploy. приложение (UI + API) на порту `HTTP_PORT` (по умолчанию 8080), на него проксирует ваш nginx, миграции применяются при старте.

Обновление: новый тег → поменять reference и `VERSION` в стеке → Pull and redeploy.

Прочее: `make help` (тесты, линтеры, миграции, sqlc, Storybook).

## Структура

```
backend/    Go: cmd/tracker, internal/{domain,usecase,adapter,infrastructure}, migrations, queries
frontend/   React: src/{app,shared,entities,features}
docs/       документация
```

## Документация

| Документ | Содержание |
|----------|------------|
| [01 — Требования](docs/01-requirements.md) | Функциональные и нефункциональные требования, user stories |
| [02 — Архитектура](docs/02-architecture.md) | Компоненты системы, ключевые решения |
| [03 — Backend](docs/03-backend.md) | Слои Clean Architecture, структура пакетов, соглашения |
| [04 — Frontend](docs/04-frontend.md) | MVC в React, структура, соглашения |
| [05 — Модель данных](docs/05-data-model.md) | Схема PostgreSQL |
| [06 — REST API](docs/06-api.md) | Эндпоинты backend |
| [07 — Интеграция с MusicBrainz](docs/07-musicbrainz.md) | Клиент, rate limit, алгоритм синхронизации |
| [08 — Уведомления](docs/08-notifications.md) | Email, RSS, outbox |
| [09 — Roadmap](docs/09-roadmap.md) | Этапы разработки |
| [10 — Дизайн](docs/10-design.md) | Скевоморфизм: концепция, токены, UI-кит, доступность |
