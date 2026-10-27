-- One notification per release group and channel (FR-5.8). On a conflict the
-- found-via artist is replaced only to promote a main credit.
-- name: InsertNotification :exec
INSERT INTO notifications (release_group_mbid, artist_mbid, credit_role, channel, status, created_at, next_attempt_at, sent_at)
VALUES ($1, $2, $3, $4, $5, $6, $6, $7)
ON CONFLICT (release_group_mbid, channel) DO UPDATE
SET artist_mbid = EXCLUDED.artist_mbid, credit_role = EXCLUDED.credit_role
WHERE notifications.credit_role <> 'main' AND EXCLUDED.credit_role = 'main';

-- name: ClaimPendingEmail :many
SELECT n.id, n.credit_role, n.created_at, sqlc.embed(rg), sqlc.embed(a)
FROM notifications n
JOIN release_groups rg ON rg.mbid = n.release_group_mbid
JOIN artists a ON a.mbid = n.artist_mbid
WHERE n.channel = 'email' AND n.status = 'pending'
ORDER BY n.created_at
FOR UPDATE OF n SKIP LOCKED;

-- name: MarkNotificationsSent :exec
UPDATE notifications SET status = 'sent', sent_at = @at WHERE id = ANY (@ids::uuid[]);

-- Retries are paced by the send_digest job; here only the attempt is counted.
-- After attempt_limit attempts the notification is failed and leaves the digest.
-- name: MarkNotificationsFailed :exec
UPDATE notifications
SET attempts   = attempts + 1,
    last_error = @last_error,
    status     = CASE WHEN attempts + 1 >= @attempt_limit::int THEN 'failed'::notification_status ELSE status END
WHERE id = ANY (@ids::uuid[]);

-- The web feed and RSS read the rss channel, newest first, with a keyset cursor.
-- name: ListFeed :many
SELECT n.id, n.credit_role, n.created_at, sqlc.embed(rg), sqlc.embed(a)
FROM notifications n
JOIN release_groups rg ON rg.mbid = n.release_group_mbid
JOIN artists a ON a.mbid = n.artist_mbid
WHERE n.channel = 'rss'
  AND (sqlc.narg(before_at)::timestamptz IS NULL OR (n.created_at, n.id) < (sqlc.narg(before_at), sqlc.narg(before_id)::uuid))
  AND (sqlc.narg(primary_type)::release_primary_type IS NULL OR rg.primary_type = sqlc.narg(primary_type))
  AND (sqlc.narg(role)::credit_role IS NULL OR n.credit_role = sqlc.narg(role))
ORDER BY n.created_at DESC, n.id DESC
LIMIT @lim;

-- The same feed by release date, newest first; undated releases go last.
-- name: ListFeedByRelease :many
SELECT n.id, n.credit_role, n.created_at, sqlc.embed(rg), sqlc.embed(a)
FROM notifications n
JOIN release_groups rg ON rg.mbid = n.release_group_mbid
JOIN artists a ON a.mbid = n.artist_mbid
WHERE n.channel = 'rss'
  AND (sqlc.narg(before_on)::date IS NULL
       OR (COALESCE(rg.first_release_on, '0001-01-01'::date), n.id) < (sqlc.narg(before_on), sqlc.narg(before_id)::uuid))
  AND (sqlc.narg(primary_type)::release_primary_type IS NULL OR rg.primary_type = sqlc.narg(primary_type))
  AND (sqlc.narg(role)::credit_role IS NULL OR n.credit_role = sqlc.narg(role))
ORDER BY COALESCE(rg.first_release_on, '0001-01-01'::date) DESC, n.id DESC
LIMIT @lim;
