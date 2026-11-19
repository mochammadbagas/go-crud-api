-- name: GetSettingByKey :one
SELECT id, setting_key, setting_value, updated_at
FROM settings
WHERE setting_key = ?
LIMIT 1;

-- name: ListSettings :many
SELECT id, setting_key, setting_value, updated_at
FROM settings
ORDER BY setting_key ASC;

-- name: UpsertSetting :exec
INSERT INTO settings (id, setting_key, setting_value, updated_at)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    setting_value = VALUES(setting_value),
    updated_at = VALUES(updated_at);
