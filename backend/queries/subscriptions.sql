-- name: InsertSubscription :execrows
INSERT INTO subscriptions (artist_mbid, primary_types, excluded_secondary_types, include_featured, baseline_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (artist_mbid) DO NOTHING;

-- name: GetSubscription :one
SELECT sqlc.embed(s), sqlc.embed(a)
FROM subscriptions s
JOIN artists a ON a.mbid = s.artist_mbid
WHERE s.artist_mbid = $1;

-- name: DeleteSubscription :execrows
DELETE FROM subscriptions WHERE artist_mbid = $1;

-- Search is a LIKE pattern without the surrounding %; NULL matches everything.

-- name: ListSubscriptions :many
SELECT sqlc.embed(s), sqlc.embed(a)
FROM subscriptions s
JOIN artists a ON a.mbid = s.artist_mbid
WHERE sqlc.narg(search)::text IS NULL
   OR a.name ILIKE '%' || sqlc.narg(search) || '%'
   OR a.sort_name ILIKE '%' || sqlc.narg(search) || '%'
ORDER BY lower(a.sort_name), a.mbid
LIMIT $1 OFFSET $2;

-- name: CountSubscriptions :one
SELECT count(*)
FROM subscriptions s
JOIN artists a ON a.mbid = s.artist_mbid
WHERE sqlc.narg(search)::text IS NULL
   OR a.name ILIKE '%' || sqlc.narg(search) || '%'
   OR a.sort_name ILIKE '%' || sqlc.narg(search) || '%';

-- name: SubscribedAmong :many
SELECT artist_mbid FROM subscriptions WHERE artist_mbid = ANY (@mbids::uuid[]);
