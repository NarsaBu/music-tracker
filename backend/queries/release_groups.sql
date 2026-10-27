-- name: UpsertReleaseGroup :batchexec
INSERT INTO release_groups (mbid, title, primary_type, secondary_types, first_release_date, first_release_on, artist_credit, rating, rating_votes, discovered_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
ON CONFLICT (mbid) DO UPDATE
SET title              = EXCLUDED.title,
    primary_type       = EXCLUDED.primary_type,
    secondary_types    = EXCLUDED.secondary_types,
    first_release_date = EXCLUDED.first_release_date,
    first_release_on   = EXCLUDED.first_release_on,
    artist_credit      = EXCLUDED.artist_credit,
    rating             = EXCLUDED.rating,
    rating_votes       = EXCLUDED.rating_votes,
    updated_at         = EXCLUDED.updated_at;

-- name: GetReleaseGroup :one
SELECT * FROM release_groups WHERE mbid = $1;

-- name: GetReleaseReviews :one
SELECT payload, fetched_at FROM release_reviews WHERE release_group_mbid = $1;

-- name: PutReleaseReviews :exec
INSERT INTO release_reviews (release_group_mbid, payload, fetched_at)
VALUES ($1, $2, $3)
ON CONFLICT (release_group_mbid) DO UPDATE SET payload = EXCLUDED.payload, fetched_at = EXCLUDED.fetched_at;

-- name: LinkArtistReleaseGroup :batchexec
INSERT INTO artist_release_groups (artist_mbid, release_group_mbid, credit_role, is_baseline, discovered_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (artist_mbid, release_group_mbid) DO NOTHING;

-- name: KnownReleaseGroups :many
SELECT release_group_mbid FROM artist_release_groups WHERE artist_mbid = $1;

-- Newest first; undated releases (often fresh announcements) on top.
-- name: ListArtistReleaseGroups :many
SELECT sqlc.embed(rg), l.credit_role, l.is_baseline, l.discovered_at AS linked_at
FROM artist_release_groups l
JOIN release_groups rg ON rg.mbid = l.release_group_mbid
WHERE l.artist_mbid = $1
  AND (sqlc.narg(role)::credit_role IS NULL OR l.credit_role = sqlc.narg(role))
ORDER BY rg.first_release_on DESC NULLS FIRST, rg.title;

-- name: GetReleaseDetails :one
SELECT payload, fetched_at FROM release_details WHERE release_group_mbid = $1;

-- name: PutReleaseDetails :exec
INSERT INTO release_details (release_group_mbid, payload, fetched_at)
VALUES ($1, $2, $3)
ON CONFLICT (release_group_mbid) DO UPDATE SET payload = EXCLUDED.payload, fetched_at = EXCLUDED.fetched_at;
