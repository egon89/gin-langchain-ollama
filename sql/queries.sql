-- name: CreateProcessedFile :exec
INSERT INTO processed_files (id, file_name, checksum, processed_at)
VALUES ($1, $2, $3, $4);

-- name: FindByChecksum :one
SELECT id, file_name, checksum, processed_at
FROM processed_files
WHERE checksum = $1
LIMIT 1;
