# 03 — Backend

## 1. Принципы

- **Clean Architecture**: зависимости направлены внутрь. `domain` ничего не импортирует из проекта; `usecase` зависит только от `domain`; адаптеры зависят от `usecase`/`domain`; `cmd` собирает всё вместе.
- **SOLID**:
  - *SRP* — один use case = одна бизнес-операция (`Subscribe`, `ImportArtists`, `SyncArtist`…).
  - *OCP* — новый канал уведомлений = новая реализация `Notifier`, без правок use cases.
  - *LSP* — любой адаптер порта взаимозаменяем (Postgres ↔ in-memory в тестах).
  - *ISP* — узкие интерфейсы: `ReleaseGroupReader` и `ReleaseGroupWriter` вместо одного «толстого» репозитория, если потребителю нужна только часть.
  - *DIP* — интерфейсы объявлены в `usecase`, реализации — в `adapter`.
- **DRY**: общие вещи (пагинация, ошибки, транзакции, retry) — в небольших пакетах `pkg/`; не плодить абстракции «на будущее».

## 2. Структура

```
backend/
├── cmd/
│   └── tracker/
│       └── main.go              # composition root: config → адаптеры → use cases → транспорт
├── internal/
│   ├── domain/                  # сущности и бизнес-правила, без внешних зависимостей
│   │   ├── artist.go
│   │   ├── release_group.go     # ReleaseGroup, PrimaryType, SecondaryType, ReleaseDate (частичные даты)
│   │   ├── credit.go            # ArtistCredit, CreditRole, RoleOf(artist) — классификация main/collaboration/featured
│   │   ├── subscription.go
│   │   ├── settings.go          # глобальные настройки, DigestSchedule.Next()
│   │   ├── release_filter.go    # ReleaseFilter.Matches(ReleaseGroup, CreditRole) bool, EffectiveFilter()
│   │   ├── notification.go      # Notification, Channel, Status
│   │   ├── import.go            # ImportJob, ImportItem, статусы
│   │   └── errors.go            # ErrNotFound, ErrAlreadyExists, ErrValidation …
│   ├── usecase/
│   │   ├── ports.go             # интерфейсы репозиториев и внешних сервисов
│   │   ├── artist/              # SearchArtists, Subscribe, Unsubscribe, ListSubscriptions
│   │   ├── importer/            # CreateImport, ProcessImportItem, ResolveImportItem
│   │   ├── sync/                # ScheduleSyncRun, SyncArtist, DetectNewReleases
│   │   ├── notify/              # FanOutNotifications, ScheduleDigest, SendDigest, SendTestEmail
│   │   ├── feed/                # BuildFeed, RotateFeedToken
│   │   └── settings/            # GetSettings, UpdateSettings
│   ├── adapter/
│   │   ├── http/                # chi-роутер, handlers, DTO, middleware (logging, recover)
│   │   ├── feed/                # рендер Atom/RSS (gorilla/feeds)
│   │   ├── postgres/            # репозитории (pgx + sqlc), TxManager
│   │   ├── musicbrainz/         # HTTP-клиент, маппинг JSON → domain, rate limiter
│   │   ├── email/               # SMTP-отправщик, html/text шаблоны
│   │   └── importfile/          # парсеры txt/csv
│   └── infrastructure/
│       ├── config/              # env → struct (caarlos0/env)
│       ├── scheduler/           # robfig/cron
│       ├── jobs/                # воркеры очереди задач (SKIP LOCKED)
│       └── observability/       # slog
├── migrations/                  # golang-migrate, NNNN_name.up.sql / .down.sql
├── queries/                     # SQL для sqlc
├── sqlc.yaml
└── go.mod
```

## 3. Порты (пример)

```go
// internal/usecase/ports.go
package usecase

type MusicCatalog interface {
    SearchArtists(ctx context.Context, query string, limit int) ([]domain.ArtistCandidate, error)
    GetArtist(ctx context.Context, mbid domain.MBID) (domain.Artist, error)
    // ReleaseGroup содержит ArtistCredit — роль вычисляется в domain
    ListReleaseGroups(ctx context.Context, artist domain.MBID) ([]domain.ReleaseGroup, error)
}

type ArtistRepository interface {
    Upsert(ctx context.Context, a domain.Artist) error
    Get(ctx context.Context, mbid domain.MBID) (domain.Artist, error)
    ListTracked(ctx context.Context) ([]domain.MBID, error)
    MarkChecked(ctx context.Context, mbid domain.MBID, at time.Time) error
}

type ReleaseGroupRepository interface {
    KnownMBIDs(ctx context.Context, artist domain.MBID) (map[domain.MBID]struct{}, error)
    InsertMany(ctx context.Context, artist domain.MBID, links []domain.ArtistReleaseGroup) error
}

type SettingsRepository interface {
    Get(ctx context.Context) (domain.Settings, error)
    Save(ctx context.Context, s domain.Settings) error
}

type Notifier interface {
    Channel() domain.Channel
    SendDigest(ctx context.Context, items []domain.NotificationItem) error
}

type TxManager interface {
    WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type Clock interface{ Now() time.Time }
```

Use case получает зависимости через конструктор:

```go
type SyncArtist struct {
    catalog  usecase.MusicCatalog
    artists  usecase.ArtistRepository
    releases usecase.ReleaseGroupRepository
    fanout   *notify.FanOut
    tx       usecase.TxManager
    clock    usecase.Clock
    cfg      Config // StaleThreshold и т.п.
}
```

## 4. Соглашения

| Тема | Правило |
|------|---------|
| Go | 1.23+, `gofmt`, `golangci-lint` (errcheck, govet, staticcheck, revive, gocritic) |
| Роутер | `go-chi/chi/v5` |
| БД | `jackc/pgx/v5` + `sqlc`; транзакция передаётся через `context` в `TxManager` |
| Миграции | `golang-migrate`, только вперёд в проде; каждая миграция с `down` |
| Ошибки | доменные ошибки в `domain/errors.go`; оборачивание `fmt.Errorf("sync artist %s: %w", id, err)`; маппинг в HTTP-коды — только в `adapter/http` |
| Логи | `log/slog`, JSON; `request_id`, `artist_mbid`, `job_id` как атрибуты |
| Контекст | первый аргумент любой I/O-функции; таймауты на внешние вызовы |
| Время | только через `Clock` (тестируемость); хранение в UTC (`timestamptz`) |
| Тесты | unit — табличные, моки через `go.uber.org/mock` или ручные фейки; интеграционные — `testcontainers-go`, build tag `integration` |
| DTO | HTTP DTO ≠ доменные сущности; маппинг в `adapter/http/dto` |

## 5. Конфигурация (env)

Локально переменные задаются в `.env` в корне репозитория (шаблон — `.env.example`, сам `.env` в git не попадает). Его читают `make run-backend` / `make migrate-*` и docker compose; compose собирает `DATABASE_URL` из `POSTGRES_*`, а `make up` подставляет `PUBLIC_BASE_URL=http://localhost:8080`. Пустые `SMTP_HOST`/`SMTP_PORT` при `make up` / `make run-backend` означают ловушку mailpit (`mailpit:1025` / `localhost:1025`): письма видны в http://localhost:8025, но никуда не уходят. Для настоящей доставки укажите SMTP своего почтового провайдера.

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `HTTP_ADDR` | `:8080` | Адрес HTTP-сервера |
| `DATABASE_URL` | — | DSN PostgreSQL |
| `PUBLIC_BASE_URL` | `http://localhost:8080` | Для ссылок в письмах и фидах |
| `STATIC_DIR` | — | Каталог собранного SPA; пусто = только API (в Docker-образе `/web`) |
| `MB_BASE_URL` | `https://musicbrainz.org/ws/2` | API MusicBrainz |
| `MB_USER_AGENT` | — | Напр. `MusicTracker/0.1 ( you@example.com )` — **обязательно** |
| `MB_RATE_LIMIT` | `1` | Запросов в секунду |
| `SYNC_CRON` | `0 6 * * *` | Расписание ежедневной синхронизации (UTC) |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASSWORD` / `SMTP_FROM` | — | Почта (адрес получателя и расписание дайджеста — в настройках UI) |
| `WORKER_CONCURRENCY` | `4` | Воркеры очереди (синк всё равно ограничен rate limiter-ом) |
