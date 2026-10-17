# 06 — REST API

Базовый путь: `/api/v1`. Формат: JSON. Аутентификация: HttpOnly cookie `session` (кроме `/auth/*`, `/feeds/*` и служебных эндпоинтов). Если пароль в конфиге не задан — аутентификация отключена.

## 1. Общие соглашения

**Пагинация** (курсорная для лент, offset — для небольших списков):
```
GET /api/v1/releases?limit=50&cursor=<opaque>
→ { "items": [...], "next_cursor": "..." | null }
```

**Ошибки** — [RFC 9457 Problem Details](https://www.rfc-editor.org/rfc/rfc9457):
```json
{
  "type": "https://music-tracker/errors/validation",
  "title": "Validation failed",
  "status": 422,
  "detail": "primary_types: unknown value 'lp'",
  "errors": [{ "field": "primary_types[0]", "message": "unknown value" }]
}
```

| Доменная ошибка | HTTP |
|-----------------|------|
| `ErrValidation` | 422 |
| `ErrNotFound` | 404 |
| `ErrAlreadyExists` | 409 |
| `ErrUnauthorized` | 401 |
| `ErrUpstreamUnavailable` (MusicBrainz, SMTP) | 503 |

**CSRF**: для изменяющих запросов backend проверяет заголовок `Origin` против `PUBLIC_BASE_URL`; cookie `SameSite=Strict`.

## 2. Доступ

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/auth/login` | `{password}` → 204 + `Set-Cookie` |
| POST | `/auth/logout` | 204, cookie сбрасывается |
| GET  | `/auth/session` | `{authenticated: bool, auth_required: bool}` |

Защита от перебора: не более 5 неудачных попыток в минуту с IP.

## 3. Каталог (прокси к MusicBrainz)

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/catalog/artists?q=<query>&limit=10` | Поиск исполнителей |
| GET | `/catalog/artists/{mbid}` | Детали исполнителя |

```json
// GET /catalog/artists?q=nirvana
{
  "items": [
    {
      "mbid": "5b11f4ce-a62d-471e-81fc-a69a8278c7da",
      "name": "Nirvana",
      "sort_name": "Nirvana",
      "disambiguation": "90s US grunge band",
      "type": "Group",            // null, если неизвестно
      "country": "US",            // null, если неизвестно
      "life_span": { "begin": "1987", "end": "1994-04-05", "ended": true },
      "score": 100,
      "subscribed": false
    }
  ]
}
```

## 4. Подписки

Подписка идентифицируется MBID исполнителя.

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/subscriptions?q=&limit=&offset=` | Список подписок |
| POST | `/subscriptions` | `{mbid, include_featured?, primary_types?, excluded_secondary_types?}` → 201 + подписка (повтор → 200, переопределения не меняются); отсутствующее/`null` поле = глобальная настройка, `[]` — явное переопределение |
| GET | `/subscriptions/{mbid}` | Подписка (формат — как элемент списка); каталог — `GET /artists/{mbid}/release-groups` |
| PATCH | `/subscriptions/{mbid}` | Переопределения `{primary_types?, excluded_secondary_types?, include_featured?}`; `null` = сброс на глобальные |
| DELETE | `/subscriptions/{mbid}` | 204 |
| POST | `/subscriptions/{mbid}/sync` | Ручная проверка исполнителя → 202 без тела (задача в очереди; пока она не выполнена, повтор не добавляет новую). Завершение видно по `last_synced_at` |

```json
// GET /subscriptions
{
  "items": [
    {
      "artist": { "mbid": "…", "name": "Radiohead", "sort_name": "Radiohead", "disambiguation": "", "type": "Group", "country": "GB" },
      "filter": {
        "primary_types":            { "value": ["album","ep"], "overridden": true },
        "excluded_secondary_types": { "value": ["Compilation","DJ-mix"], "overridden": false },
        "include_featured":         { "value": true, "overridden": true }
      },
      "last_synced_at": "2026-10-17T06:03:12Z",   // null до первой синхронизации
      "baseline_at": "2026-09-01T10:00:00Z",
      "created_at": "2026-09-01T10:00:00Z"
    }
  ],
  "total": 42
}
```

## 5. Импорт

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/imports` | `multipart/form-data`, поле `file` (≤ 1 МБ, ≤ 5000 строк) → 202 `{id}` |
| GET | `/imports` | Список импортов |
| GET | `/imports/{id}` | Статус + счётчики по статусам строк |
| GET | `/imports/{id}/items?status=needs_review` | Строки с кандидатами |
| POST | `/imports/{id}/items/{itemId}/resolve` | `{mbid, include_featured?}` или `{skip: true}` |

```json
// GET /imports/{id}
{
  "id": "…",
  "filename": "artists.csv",
  "total": 120,
  "counts": { "pending": 10, "matched": 95, "needs_review": 12, "not_found": 3, "skipped": 0, "failed": 0 },
  "finished": false
}
```

## 6. Релизы

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/releases?limit=&cursor=&type=album&role=featured` | Лента найденных новинок |
| GET | `/artists/{mbid}/release-groups?role=` | Полный каталог исполнителя (из БД): новые и без даты сверху. Элемент: `mbid, title, artist_credit, credit_role, primary_type, secondary_types, first_release_date, is_baseline, is_announcement, cover_url, musicbrainz_url, discovered_at` |

```json
{
  "items": [
    {
      "mbid": "…",
      "title": "New Album",
      "artist_credit": "Radiohead feat. Björk",
      "found_via": { "mbid": "…", "name": "Björk", "credit_role": "featured" },
      "primary_type": "album",
      "secondary_types": [],
      "first_release_date": "2026-11-07",
      "is_announcement": true,
      "cover_url": "https://coverartarchive.org/release-group/…/front-250",
      "musicbrainz_url": "https://musicbrainz.org/release-group/…",
      "discovered_at": "2026-10-17T06:03:12Z"
    }
  ],
  "next_cursor": null
}
```

## 7. Настройки

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/settings` | Текущие настройки (+ вычисленный `next_digest_at`) |
| PUT | `/settings` | Обновление |
| POST | `/settings/email/test` | Отправить тестовое письмо → 204 |
| POST | `/settings/feed-token/rotate` | Перевыпуск RSS-токена → `{feed_url}` |
| GET | `/meta/release-types` | Справочник primary/secondary types для UI |

```json
// PUT /settings
{
  "filters": {
    "primary_types": ["album", "ep", "single"],
    "excluded_secondary_types": ["Compilation", "Live"],
    "include_featured": false,
    "notify_announcements": true
  },
  "email": {
    "enabled": true,
    "to": "me@example.com",
    "digest": { "frequency": "weekly", "weekday": 5, "time": "09:00", "timezone": "Europe/Moscow" }
  },
  "rss": { "enabled": true }
}
```

## 8. Синхронизация

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/sync` | Ручной запуск для всех подписок → 202 + прогон; если прогон уже идёт — возвращается он |
| GET | `/sync/runs/latest` | Последний прогон или `null` |

```json
// GET /sync/runs/latest
{
  "id": "…",
  "trigger": "manual",              // manual | schedule
  "started_at": "2026-10-17T06:00:00Z",
  "finished_at": null,              // null — прогон идёт
  "artists_total": 40,
  "artists_done": 12,
  "new_releases": 1                 // не baseline и не устаревшие
}
```

## 9. Фиды (без сессии, по токену)

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/feeds/{token}.atom` | Atom 1.0 |
| GET | `/feeds/{token}.rss` | RSS 2.0 |

Поддерживаются `ETag` / `If-None-Match` и `Last-Modified` → `304`.

## 10. Служебные

`GET /healthz` (liveness), `GET /readyz` (БД доступна), `GET /metrics` (Prometheus, закрыть от внешнего доступа).
