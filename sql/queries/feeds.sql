-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetFeeds :many
SELECT * FROM feeds;

-- name: JoinFeedsTable :many
SELECT feeds.name, feeds.url, users.name 
FROM feeds 
JOIN users ON feeds.user_id = users.id;

-- name: MarkFeedFetched :exec
UPDATE feeds 
SET updated_at = NOW(), last_fetched_at = NOW()
WHERE feeds.id = $1;

-- name: GetNextFeedToFetch :one
SELECT feeds.url FROM feeds ORDER BY last_fetched_at NULLS FIRST LIMIT 1;