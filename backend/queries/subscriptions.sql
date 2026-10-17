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

-- Filters: search is a LIKE pattern without the surrounding %; every NULL filter
-- matches everything. synced: whether the first sync is done; overridden: whether
-- any field of the per-artist filter is set.

-- name: ListSubscriptions :many
SELECT sqlc.embed(s), sqlc.embed(a)
FROM subscriptions s
JOIN artists a ON a.mbid = s.artist_mbid
WHERE (sqlc.narg(search)::text IS NULL
       OR a.name ILIKE '%' || sqlc.narg(search) || '%'
       OR a.sort_name ILIKE '%' || sqlc.narg(search) || '%')
  AND (sqlc.narg(synced)::boolean IS NULL OR (a.last_synced_at IS NOT NULL) = sqlc.narg(synced))
  AND (sqlc.narg(artist_type)::text IS NULL OR a.type = sqlc.narg(artist_type))
  AND (sqlc.narg(country)::text IS NULL OR a.country = sqlc.narg(country))
  AND (sqlc.narg(overridden)::boolean IS NULL
       OR (s.primary_types IS NOT NULL OR s.excluded_secondary_types IS NOT NULL
           OR s.include_featured IS NOT NULL) = sqlc.narg(overridden))
ORDER BY CASE WHEN @newest_first::boolean THEN s.created_at END DESC,
         lower(a.sort_name), a.mbid
LIMIT $1 OFFSET $2;

-- name: CountSubscriptions :one
SELECT count(*)
FROM subscriptions s
JOIN artists a ON a.mbid = s.artist_mbid
WHERE (sqlc.narg(search)::text IS NULL
       OR a.name ILIKE '%' || sqlc.narg(search) || '%'
       OR a.sort_name ILIKE '%' || sqlc.narg(search) || '%')
  AND (sqlc.narg(synced)::boolean IS NULL OR (a.last_synced_at IS NOT NULL) = sqlc.narg(synced))
  AND (sqlc.narg(artist_type)::text IS NULL OR a.type = sqlc.narg(artist_type))
  AND (sqlc.narg(country)::text IS NULL OR a.country = sqlc.narg(country))
  AND (sqlc.narg(overridden)::boolean IS NULL
       OR (s.primary_types IS NOT NULL OR s.excluded_secondary_types IS NOT NULL
           OR s.include_featured IS NOT NULL) = sqlc.narg(overridden));

-- name: SubscriptionCountries :many
SELECT DISTINCT a.country::text
FROM subscriptions s
JOIN artists a ON a.mbid = s.artist_mbid
WHERE a.country IS NOT NULL AND a.country <> ''
ORDER BY 1;

-- name: SubscriptionArtistTypes :many
SELECT DISTINCT a.type::text
FROM subscriptions s
JOIN artists a ON a.mbid = s.artist_mbid
WHERE a.type IS NOT NULL AND a.type <> ''
ORDER BY 1;

-- name: SubscribedAmong :many
SELECT artist_mbid FROM subscriptions WHERE artist_mbid = ANY (@mbids::uuid[]);

-- name: UpdateSubscriptionOverride :execrows
UPDATE subscriptions
SET primary_types = $2, excluded_secondary_types = $3, include_featured = $4
WHERE artist_mbid = $1;
