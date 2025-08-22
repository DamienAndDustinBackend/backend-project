-- name: ListFiles :many
SELECT * FROM files;

-- name: GetFile :one
SELECT * FROM files
WHERE id = ? LIMIT 1;

-- name: CreateFile :exec
INSERT INTO files (
    name, description, type, file_path
) VALUES (
  ?, ?, ?, ?
);

-- name: UpdateFile :exec
UPDATE files SET name = ?, description = ?, type = ?, type = ?, file_path = ?
WHERE id = ?;

-- name: DeleteFile :exec
DELETE FROM files WHERE id = ?;

-- name: ListTags :one
SELECT * FROM tags;

-- name: GetTag :one
SELECT * FROM tags
WHERE id = ? LIMIT 1;

-- name: CreateTag :exec
INSERT INTO tags (
    name
) VALUES (
    ?
);

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = ?;

-- name: UpdateTag :exec
UPDATE tags SET name = ?
WHERE id = ?;

-- name: GetFilesByTag :many
SELECT *
FROM files;
-- JOIN files_tags ON files.id = file_tags.file_id
-- JOIN tags ON file_tags.tag_id = tags.id

-- name: GetTagsForFile :many
SELECT *
FROM tags;
-- JOIN files_tags ON ? = file_tags.file_id
-- JOIN tags ON file_tags.tag_id = tags.id
