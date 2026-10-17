-- name: UpsertArtist :exec
INSERT INTO artists (mbid, name, sort_name, disambiguation, type, country)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (mbid) DO UPDATE
SET name           = EXCLUDED.name,
    sort_name      = EXCLUDED.sort_name,
    disambiguation = EXCLUDED.disambiguation,
    type           = EXCLUDED.type,
    country        = EXCLUDED.country;

-- name: GetArtist :one
SELECT * FROM artists WHERE mbid = $1;

-- Tracked artists are the ones with a subscription (unsubscribed artists are kept but skipped).
-- name: ListTrackedArtists :many
SELECT a.mbid FROM artists a JOIN subscriptions s ON s.artist_mbid = a.mbid ORDER BY a.mbid;

-- name: MarkArtistSynced :exec
UPDATE artists SET last_synced_at = $2 WHERE mbid = $1;
