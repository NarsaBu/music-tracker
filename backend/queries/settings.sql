-- name: GetSettings :one
SELECT * FROM settings WHERE id = 1;

-- name: GetFilterDefaults :one
SELECT primary_types, excluded_secondary_types, include_featured FROM settings WHERE id = 1;

-- name: UpdateSettings :exec
UPDATE settings
SET primary_types            = @primary_types,
    excluded_secondary_types = @excluded_secondary_types,
    include_featured         = @include_featured,
    notify_announcements     = @notify_announcements,
    email_enabled            = @email_enabled,
    email_to                 = @email_to,
    digest_frequency         = @digest_frequency,
    digest_weekday           = @digest_weekday,
    digest_time              = @digest_time,
    timezone                 = @timezone,
    next_digest_at           = @next_digest_at,
    rss_enabled              = @rss_enabled
WHERE id = 1;

-- name: SetFeedToken :exec
UPDATE settings SET feed_token = $1 WHERE id = 1;

-- name: SetDigestTimes :exec
UPDATE settings SET last_digest_at = coalesce(sqlc.narg(last_digest_at), last_digest_at), next_digest_at = @next_digest_at WHERE id = 1;
