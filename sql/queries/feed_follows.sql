-- name: CreateFeedFollow :one
WITH inserted_follow AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES (
        $1,
        $2,
        $3,
        $4,
        $5
    )
    RETURNING *
)
SELECT
    inserted_follow.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM inserted_follow
INNER JOIN users ON inserted_follow.user_id = users.id
INNER JOIN feeds ON inserted_follow.feed_id = feeds.id;

-- name: FeedLookup :one
SELECT id, name FROM feeds WHERE url = $1;

-- name: GetFeedFollowsForUser :many
SELECT
    feed_follows.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM feed_follows
INNER JOIN users ON feed_follows.user_id = users.id
INNER JOIN feeds ON feed_follows.feed_id = feeds.id
WHERE feed_follows.user_id = $1;

-- name: UnfollowFeed :exec
DELETE FROM feed_follows WHERE feed_follows.user_id = $1 AND feed_follows.feed_id = $2;