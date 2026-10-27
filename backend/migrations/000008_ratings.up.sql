-- Community rating of a release group from MusicBrainz, refreshed by every sync.
ALTER TABLE release_groups
    ADD COLUMN rating       double precision CHECK (rating BETWEEN 0 AND 5),
    ADD COLUMN rating_votes int NOT NULL DEFAULT 0;

-- Cache of CritiqueBrainz reviews: one answer per release group, refreshed after a TTL.
CREATE TABLE release_reviews (
    release_group_mbid uuid PRIMARY KEY REFERENCES release_groups (mbid) ON DELETE CASCADE,
    payload            jsonb       NOT NULL,
    fetched_at         timestamptz NOT NULL
);
