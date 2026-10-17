-- name: CreateSyncRun :one
INSERT INTO sync_runs (trigger, started_at, artists_total, finished_at)
VALUES ($1, $2, $3, sqlc.narg(finished_at))
RETURNING *;

-- A run older than the cutoff is considered abandoned (e.g. its jobs were lost).
-- name: GetRunningSyncRun :one
SELECT * FROM sync_runs
WHERE finished_at IS NULL AND started_at > $1
ORDER BY started_at DESC
LIMIT 1;

-- name: GetLatestSyncRun :one
SELECT * FROM sync_runs ORDER BY started_at DESC LIMIT 1;

-- name: SyncRunArtistDone :exec
UPDATE sync_runs
SET artists_done = artists_done + 1,
    new_releases = new_releases + @new_releases,
    finished_at  = CASE WHEN artists_done + 1 >= artists_total THEN @now::timestamptz ELSE finished_at END
WHERE id = @id AND finished_at IS NULL;
