-- Release catalog, sync runs and the background job queue (docs/05-data-model.md).
CREATE TABLE release_groups (
    mbid               uuid PRIMARY KEY,
    title              text NOT NULL,
    primary_type       release_primary_type,           -- may be absent in MusicBrainz
    secondary_types    text[] NOT NULL DEFAULT '{}',
    first_release_date text,                           -- 'YYYY' | 'YYYY-MM' | 'YYYY-MM-DD' | NULL
    first_release_on   date,                           -- normalized to the first day, for sorting
    artist_credit      text NOT NULL,                  -- display string, e.g. "A feat. B"
    discovered_at      timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE artist_release_groups (
    artist_mbid        uuid NOT NULL REFERENCES artists (mbid) ON DELETE CASCADE,
    release_group_mbid uuid NOT NULL REFERENCES release_groups (mbid) ON DELETE CASCADE,
    credit_role        credit_role NOT NULL,
    is_baseline        boolean NOT NULL DEFAULT false,
    discovered_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (artist_mbid, release_group_mbid)
);

CREATE TABLE sync_runs (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    trigger       text NOT NULL CHECK (trigger IN ('schedule', 'manual')),
    started_at    timestamptz NOT NULL DEFAULT now(),
    finished_at   timestamptz,
    artists_total int NOT NULL DEFAULT 0,
    artists_done  int NOT NULL DEFAULT 0,
    new_releases  int NOT NULL DEFAULT 0
);
CREATE INDEX ON sync_runs (started_at DESC);

-- Generic background job queue, claimed with FOR UPDATE SKIP LOCKED (ADR-5).
-- A job that used up max_attempts stays locked as "dead" for inspection.
CREATE TABLE jobs (
    id           bigserial PRIMARY KEY,
    kind         text  NOT NULL,
    payload      jsonb NOT NULL,
    dedup_key    text UNIQUE,
    run_at       timestamptz NOT NULL DEFAULT now(),
    attempts     int NOT NULL DEFAULT 0,
    max_attempts int NOT NULL DEFAULT 5,
    locked_at    timestamptz,
    last_error   text,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ON jobs (run_at) WHERE locked_at IS NULL;
