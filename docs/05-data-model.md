# 05 — Модель данных (PostgreSQL)

Приложение однопользовательское, поэтому таблицы `users` нет: глобальные настройки хранятся в singleton-таблице `settings`.

## 1. ER-диаграмма

```
settings (1 строка)

subscriptions ──1:1── artists ──1:N── artist_release_groups ──N:1── release_groups
                                       (credit_role)                     │
notifications ──────────────────────────────────────────────N:1──────────┘
import_jobs ──1:N── import_items

sync_runs, jobs — служебные
```

## 2. Типы

```sql
CREATE TYPE release_primary_type AS ENUM ('album','single','ep','broadcast','other');
CREATE TYPE credit_role          AS ENUM ('main','collaboration','featured');
CREATE TYPE notification_channel AS ENUM ('email','rss');
CREATE TYPE notification_status  AS ENUM ('pending','sent','failed');
CREATE TYPE digest_frequency     AS ENUM ('daily','weekly');
CREATE TYPE import_item_status   AS ENUM ('pending','matched','needs_review','not_found','skipped','failed');
```

Secondary types хранятся как `text[]` (их список в MusicBrainz расширяется — enum неудобен).

## 3. Таблицы

```sql
CREATE TABLE settings (
    id                       smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    -- фильтры по умолчанию
    primary_types            release_primary_type[] NOT NULL DEFAULT '{album,ep,single}',
    excluded_secondary_types text[] NOT NULL DEFAULT
        '{Compilation,DJ-mix,Interview,Spokenword,Audiobook,Audio drama,Demo,Field recording}',
    include_featured         boolean NOT NULL DEFAULT false,
    notify_announcements     boolean NOT NULL DEFAULT true,
    -- email
    email_enabled            boolean NOT NULL DEFAULT false,
    email_to                 text,
    digest_frequency         digest_frequency NOT NULL DEFAULT 'daily',
    digest_weekday           smallint CHECK (digest_weekday BETWEEN 0 AND 6),  -- 0 = воскресенье; для weekly
    digest_time              time NOT NULL DEFAULT '09:00',
    timezone                 text NOT NULL DEFAULT 'UTC',
    last_digest_at           timestamptz,
    next_digest_at           timestamptz,                                     -- пересчитывается при изменении расписания и после отправки
    -- rss
    rss_enabled              boolean NOT NULL DEFAULT true,
    feed_token               text NOT NULL,
    CHECK (digest_frequency = 'daily' OR digest_weekday IS NOT NULL)
);

CREATE TABLE artists (
    mbid             uuid PRIMARY KEY,
    name             text NOT NULL,
    sort_name        text NOT NULL,
    disambiguation   text NOT NULL DEFAULT '',
    type             text,                    -- Person, Group, …
    country          text,
    last_synced_at   timestamptz,             -- NULL = baseline ещё не выполнен
    created_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscriptions (
    artist_mbid              uuid PRIMARY KEY REFERENCES artists(mbid) ON DELETE CASCADE,
    -- переопределения: NULL = использовать settings
    primary_types            release_primary_type[],
    excluded_secondary_types text[],
    include_featured         boolean,
    baseline_at              timestamptz NOT NULL DEFAULT now(),
    created_at               timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE release_groups (
    mbid               uuid PRIMARY KEY,
    title              text NOT NULL,
    primary_type       release_primary_type,           -- может отсутствовать в MB
    secondary_types    text[] NOT NULL DEFAULT '{}',
    first_release_date text,                           -- 'YYYY' | 'YYYY-MM' | 'YYYY-MM-DD' | NULL
    first_release_on   date,                           -- нормализованная (для сортировки/сравнений)
    artist_credit      text NOT NULL,                  -- строка для отображения, напр. "A feat. B"
    discovered_at      timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE artist_release_groups (
    artist_mbid        uuid NOT NULL REFERENCES artists(mbid) ON DELETE CASCADE,
    release_group_mbid uuid NOT NULL REFERENCES release_groups(mbid) ON DELETE CASCADE,
    credit_role        credit_role NOT NULL,            -- роль исполнителя в credit этой релиз-группы
    is_baseline        boolean NOT NULL DEFAULT false,  -- найден при первичной загрузке
    discovered_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (artist_mbid, release_group_mbid)
);

CREATE TABLE notifications (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    release_group_mbid uuid NOT NULL REFERENCES release_groups(mbid),
    artist_mbid        uuid NOT NULL REFERENCES artists(mbid),   -- через кого найден
    credit_role        credit_role NOT NULL,
    channel            notification_channel NOT NULL,
    status             notification_status  NOT NULL DEFAULT 'pending',
    attempts           int NOT NULL DEFAULT 0,
    next_attempt_at    timestamptz NOT NULL DEFAULT now(),
    last_error         text,
    created_at         timestamptz NOT NULL DEFAULT now(),
    sent_at            timestamptz,
    UNIQUE (release_group_mbid, channel)                 -- FR-5.8: совместный релиз двух подписок — одно уведомление
);
CREATE INDEX ON notifications (channel, status, next_attempt_at);
CREATE INDEX ON notifications (channel, created_at DESC);  -- лента / RSS

CREATE TABLE import_jobs (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    filename    text NOT NULL,
    total       int  NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz
);

CREATE TABLE import_items (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id           uuid NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    line_no          int  NOT NULL,
    raw_value        text NOT NULL,
    include_featured boolean,                    -- из CSV, если указано
    status           import_item_status NOT NULL DEFAULT 'pending',
    candidates       jsonb,                      -- [{mbid,name,disambiguation,country,score}]
    artist_mbid      uuid REFERENCES artists(mbid),
    error            text
);
CREATE INDEX ON import_items (job_id, status);

CREATE TABLE sync_runs (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    trigger       text NOT NULL,              -- 'schedule' | 'manual'
    started_at    timestamptz NOT NULL DEFAULT now(),
    finished_at   timestamptz,
    artists_total int NOT NULL DEFAULT 0,
    artists_done  int NOT NULL DEFAULT 0,
    new_releases  int NOT NULL DEFAULT 0
);

-- Универсальная очередь фоновых задач (ADR-5)
CREATE TABLE jobs (
    id           bigserial PRIMARY KEY,
    kind         text  NOT NULL,              -- sync_artist | import_item | send_digest
    payload      jsonb NOT NULL,
    dedup_key    text UNIQUE,                 -- напр. 'sync_artist:<run>:<mbid>'
    run_at       timestamptz NOT NULL DEFAULT now(),
    attempts     int NOT NULL DEFAULT 0,
    max_attempts int NOT NULL DEFAULT 5,
    locked_at    timestamptz,
    last_error   text,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ON jobs (run_at) WHERE locked_at IS NULL;
```

## 4. Примечания

- **Частичные даты.** MusicBrainz отдаёт `first-release-date` как `2024`, `2024-05` или `2024-05-17`. Исходная строка хранится для отображения, `first_release_on` — нормализация к первому дню периода (`2024` → `2024-01-01`) для сравнений.
- **`credit_role` хранится в связке «исполнитель — релиз-группа»**, а не в релиз-группе: для `"A feat. B"` у A роль `main`, у B — `featured`.
- **Featuring-релизы сохраняются всегда**, даже если `include_featured` выключен. Включение флага позже не создаёт уведомлений задним числом (они уже прошли baseline/discovery), но featuring-релизы сразу появляются в каталоге исполнителя.
- **Отписка** удаляет строку `subscriptions`; `artists` и каталог остаются для повторной подписки, синхронизация их пропускает. Очистка — периодической задачей.
