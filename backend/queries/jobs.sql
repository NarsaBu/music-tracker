-- A duplicate dedup_key is ignored, unless the existing job is dead: then it is revived.
-- name: EnqueueJob :execrows
INSERT INTO jobs (kind, payload, dedup_key, run_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (dedup_key) DO UPDATE
SET kind = EXCLUDED.kind, payload = EXCLUDED.payload, run_at = EXCLUDED.run_at,
    attempts = 0, locked_at = NULL, last_error = NULL
WHERE jobs.attempts >= jobs.max_attempts;

-- name: ClaimJob :one
UPDATE jobs
SET locked_at = now(), attempts = attempts + 1
WHERE id = (
    SELECT id FROM jobs
    WHERE locked_at IS NULL AND run_at <= now()
    ORDER BY run_at, id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, kind, payload, attempts, max_attempts;

-- name: DeleteJob :exec
DELETE FROM jobs WHERE id = $1;

-- name: RetryJob :exec
UPDATE jobs SET locked_at = NULL, run_at = $2, last_error = $3 WHERE id = $1;

-- name: BuryJob :exec
UPDATE jobs SET last_error = $2 WHERE id = $1;

-- Unlocks jobs of a worker that died mid-flight; dead jobs stay locked.
-- name: ReleaseStaleJobs :execrows
UPDATE jobs SET locked_at = NULL
WHERE locked_at < $1 AND attempts < max_attempts;

-- Same as EnqueueJob, for many jobs in one round trip.
-- name: EnqueueJobs :batchexec
INSERT INTO jobs (kind, payload, dedup_key, run_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (dedup_key) DO UPDATE
SET kind = EXCLUDED.kind, payload = EXCLUDED.payload, run_at = EXCLUDED.run_at,
    attempts = 0, locked_at = NULL, last_error = NULL
WHERE jobs.attempts >= jobs.max_attempts;
