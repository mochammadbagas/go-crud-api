-- name: GetUserProfileByUserID :one
SELECT user_id, full_name, bio, phone_number, avatar_url, created_at, updated_at
FROM user_profiles
WHERE user_id = ?
LIMIT 1;

-- name: CreateUserProfile :exec
INSERT INTO user_profiles (user_id, full_name, bio, phone_number, avatar_url, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateUserProfile :exec
UPDATE user_profiles
SET full_name = ?, bio = ?, phone_number = ?, updated_at = ?
WHERE user_id = ?;

-- name: UpdateAvatar :exec
UPDATE user_profiles
SET avatar_url = ?, updated_at = ?
WHERE user_id = ?;

-- name: DeleteAvatar :exec
UPDATE user_profiles
SET avatar_url = NULL, updated_at = ?
WHERE user_id = ?;

-- name: GetUserWithProfileByID :one
SELECT 
    u.id, u.username, u.email, u.role, u.status, u.email_verified_at, u.created_at, u.updated_at,
    p.full_name, p.bio, p.phone_number, p.avatar_url
FROM users u
LEFT JOIN user_profiles p ON u.id = p.user_id
WHERE u.id = ?
LIMIT 1;
