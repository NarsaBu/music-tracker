-- Cache of release page details from MusicBrainz: genres, tracklist, credited artists.
CREATE TABLE release_details (
    release_group_mbid uuid PRIMARY KEY REFERENCES release_groups (mbid) ON DELETE CASCADE,
    payload            jsonb       NOT NULL,
    fetched_at         timestamptz NOT NULL
);
