-- name: UpsertArtist :exec
INSERT INTO artists (mbid, name, sort_name, disambiguation, type, country)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (mbid) DO UPDATE
SET name           = EXCLUDED.name,
    sort_name      = EXCLUDED.sort_name,
    disambiguation = EXCLUDED.disambiguation,
    type           = EXCLUDED.type,
    country        = EXCLUDED.country;
