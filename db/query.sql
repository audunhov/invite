-- name: GetPerson :one
SELECT * FROM persons WHERE id = $1;
