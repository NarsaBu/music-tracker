-- name: CreateImportJob :one
INSERT INTO import_jobs (filename, total, created_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: InsertImportItem :batchone
INSERT INTO import_items (job_id, line_no, raw_value, input_mbid, include_featured)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: GetImportJob :one
SELECT * FROM import_jobs WHERE id = $1;

-- name: ListImportJobs :many
SELECT * FROM import_jobs ORDER BY created_at DESC, id LIMIT $1;

-- name: CountImportItems :many
SELECT job_id, status, count(*)::int AS n
FROM import_items
WHERE job_id = ANY (@job_ids::uuid[])
GROUP BY job_id, status;

-- An item of the given job, or of any job when job_id is NULL; locked for the transaction.
-- name: GetImportItem :one
SELECT sqlc.embed(i), a.name AS artist_name
FROM import_items i
LEFT JOIN artists a ON a.mbid = i.artist_mbid
WHERE i.id = @id AND (sqlc.narg(job_id)::uuid IS NULL OR i.job_id = sqlc.narg(job_id))
FOR UPDATE OF i;

-- name: ListImportItems :many
SELECT sqlc.embed(i), a.name AS artist_name
FROM import_items i
LEFT JOIN artists a ON a.mbid = i.artist_mbid
WHERE i.job_id = @job_id AND (cardinality(@statuses::import_item_status[]) = 0 OR i.status = ANY (@statuses))
ORDER BY i.line_no, i.id;

-- name: UpdateImportItem :exec
UPDATE import_items
SET status = $2, candidates = $3, artist_mbid = $4, error = $5
WHERE id = $1;

-- name: FinishImportJobIfDone :exec
UPDATE import_jobs j
SET finished_at = @now
WHERE j.id = @id AND j.finished_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM import_items i WHERE i.job_id = j.id AND i.status = 'pending');
