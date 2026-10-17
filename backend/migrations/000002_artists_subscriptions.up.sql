-- Artists known to the system and the subscriptions to them (docs/05-data-model.md).
CREATE TABLE artists (
    mbid           uuid PRIMARY KEY,
    name           text NOT NULL,
    sort_name      text NOT NULL,
    disambiguation text NOT NULL DEFAULT '',
    type           text,                    -- Person, Group, …
    country        text,
    last_synced_at timestamptz,             -- NULL = baseline not done yet
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscriptions (
    artist_mbid              uuid PRIMARY KEY REFERENCES artists (mbid) ON DELETE CASCADE,
    -- overrides of the global filter: NULL = use settings
    primary_types            release_primary_type[],
    excluded_secondary_types text[],
    include_featured         boolean,
    baseline_at              timestamptz NOT NULL DEFAULT now(),
    created_at               timestamptz NOT NULL DEFAULT now()
);
