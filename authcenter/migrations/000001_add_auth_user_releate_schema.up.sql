-- Users 表
CREATE TABLE user  IF NOT EXISTS (
    id UUID PRIMARY KEY,
    google_id VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_admin BOOLEAN NOT NULL,
    line_user_id VARCHAR(255) UNIQUE,
    line_linked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Roles 表
CREATE TABLE role  IF NOT EXISTS (
    id INTEGER PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description VARCHAR(255)
);

-- Permissions 表
CREATE TABLE permission IF NOT EXISTS (
    id INTEGER PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description VARCHAR(255)
);

-- User_Roles 表 (多對多關聯)
CREATE TABLE user_role IF NOT EXISTS (
    user_id INT NOT NULL,
    role_id INT NOT NULL,
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
);

-- Role_Permissions 表 (多對多關聯)
CREATE TABLE role_permission IF NOT EXISTS (
    role_id INT NOT NULL,
    permission_id INT NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

CREATE TABLE user_session (
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

CREATE INDEX idx_user_session_user_id ON user_session(user_id);
CREATE INDEX idx_user_session_refresh_token ON user_session(refresh_token);
CREATE INDEX idx_user_session_is_active ON user_session(is_active);