-- name: GetUserPermissions :many
SELECT p.*
FROM users u
JOIN user_roles ur ON u.id = ur.user_id
JOIN role_permissions rp ON ur.role_id = rp.role_id
JOIN permissions p ON p.id = rp.permission_id
WHERE u.id = $1;