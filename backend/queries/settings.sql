-- name: GetSettings :one
SELECT * FROM settings WHERE id = 1;

-- name: GetFilterDefaults :one
SELECT primary_types, excluded_secondary_types, include_featured FROM settings WHERE id = 1;
