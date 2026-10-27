-- Cache of ListenBrainz similar artists: one answer per artist, refreshed after a TTL.
-- No foreign key: the cache may outlive a subscription and is cheap to refill.
CREATE TABLE similar_artists (
    artist_mbid uuid PRIMARY KEY,
    items       jsonb       NOT NULL,
    fetched_at  timestamptz NOT NULL
);
