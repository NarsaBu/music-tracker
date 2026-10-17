-- Artist import from a file (FR-2.3, docs/05-data-model.md).
CREATE TABLE import_jobs (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    filename    text NOT NULL,
    total       int  NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz                          -- set when no line is pending any more
);
CREATE INDEX ON import_jobs (created_at DESC);

CREATE TABLE import_items (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id           uuid NOT NULL REFERENCES import_jobs (id) ON DELETE CASCADE,
    line_no          int  NOT NULL,
    raw_value        text NOT NULL,                  -- name (or MBID) as written in the file
    input_mbid       uuid,                           -- MBID from the file: looked up, not searched
    include_featured boolean,                        -- from CSV, if given
    status           import_item_status NOT NULL DEFAULT 'pending',
    candidates       jsonb,                          -- search results to choose from (needs_review)
    artist_mbid      uuid REFERENCES artists (mbid), -- the subscribed artist (matched)
    error            text
);
CREATE INDEX ON import_items (job_id, status);
