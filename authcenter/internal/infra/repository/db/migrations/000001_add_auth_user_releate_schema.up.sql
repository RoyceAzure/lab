-- Users 表
CREATE TABLE IF NOT EXISTS "users"  (
    id UUID PRIMARY KEY,
    google_id VARCHAR(255) UNIQUE NULL,
    facebook_id VARCHAR(255) UNIQUE NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    account VARCHAR(30) UNIQUE NULL,
    password_hash text NULL,
    name VARCHAR(255) NULL,
    is_admin BOOLEAN NOT NULL,
    is_active BOOLEAN NOT NULL,
    line_user_id VARCHAR(255) UNIQUE,
    line_linked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Roles 表
CREATE TABLE IF NOT EXISTS "roles"  (
    id INTEGER PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description VARCHAR(255)
);

-- Permissions 表
CREATE TABLE IF NOT EXISTS "permissions"  (
    id INTEGER PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    resource VARCHAR(255) NOT NULL,
    actions VARCHAR(255) NOT NULL,
    description VARCHAR(255)
);

-- User_Roles 表 (多對多關聯)
CREATE TABLE IF NOT EXISTS "user_roles"  (
    user_id UUID NOT NULL,
    role_id INT NOT NULL,
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
);

-- Role_Permissions 表 (多對多關聯)
CREATE TABLE IF NOT EXISTS "role_permissions"  (
    role_id INT NOT NULL,
    permission_id INT NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS "user_sessions"  (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    ip_address INET NOT NULL,
    device_info TEXT NOT NULL,
    region VARCHAR(255),
    user_agent TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    last_activity_at timestamptz  NOT NULL,
    created_at timestamptz  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz 
);


-- email 認證表
CREATE TABLE IF NOT EXISTS "email_vertify"  (
    id VARCHAR(30) PRIMARY KEY NOT NULL,
    email VARCHAR(255) NOT NULL,
    is_valid BOOLEAN NOT NULL DEFAULT FALSE,
    created_at timestamptz  NOT NULL,
    expires_at timestamptz NOT NULL
);



CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_refresh_token ON user_sessions(refresh_token);
CREATE INDEX idx_user_sessions_is_active ON user_sessions(is_active);