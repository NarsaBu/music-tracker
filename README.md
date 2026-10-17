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
