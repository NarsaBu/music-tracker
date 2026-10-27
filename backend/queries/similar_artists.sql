-- name: GetSimilarArtists :one
SELECT items, fetched_at FROM similar_artists WHERE artist_mbid = $1;

-- name: PutSimilarArtists :exec
INSERT INTO similar_artists (artist_mbid, items, fetched_at)
VALUES ($1, $2, $3)
ON CONFLICT (artist_mbid) DO UPDATE SET items = EXCLUDED.items, fetched_at = EXCLUDED.fetched_at;
