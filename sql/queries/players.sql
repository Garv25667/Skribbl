-- name: CreatePlayer :one
INSERT INTO players (username)
VALUES ($1)
RETURNING *;


-- name: GetPlayerByUsername :one
SELECT * FROM players 
WHERE username = $1;