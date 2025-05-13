# 認證中心資料庫設計

本文檔詳細描述認證中心系統的資料庫結構設計，包括表結構、關係、索引和優化考量。

## 1. 資料庫概述

### 1.1 設計原則

- **安全優先**: 敏感資料加密存儲，最小權限原則
- **高效能**: 適當的索引和分區策略，優化查詢效率
- **可擴展性**: 考慮未來擴展，避免過度耦合
- **資料完整性**: 強化約束和關係確保資料完整性
- **審計能力**: 完整的資料變更歷史記錄

### 1.2 使用的資料庫技術

- 主資料庫: PostgreSQL 15+
- 快取層: Redis
- 索引技術: B-tree, GIN (適用於全文搜索)

## 2. 核心表結構

### 2.1 用戶與身份管理

#### users (用戶表)

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone_number VARCHAR(50) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    salt VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    active BOOLEAN DEFAULT TRUE,
    email_verified BOOLEAN DEFAULT FALSE,
    phone_verified BOOLEAN DEFAULT FALSE,
    require_mfa BOOLEAN DEFAULT FALSE,
    mfa_type VARCHAR(50),
    mfa_secret TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ,
    password_updated_at TIMESTAMPTZ,
    account_locked_until TIMESTAMPTZ,
    failed_login_attempts INT DEFAULT 0,
    metadata JSONB
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_phone ON users(phone_number);
CREATE INDEX idx_users_active ON users(active);
```

#### user_external_identities (外部身份關聯表)

```sql
CREATE TABLE user_external_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- "google", "facebook", "github", etc.
    external_id VARCHAR(255) NOT NULL,
    external_username VARCHAR(255),
    external_email VARCHAR(255),
    access_token TEXT,
    refresh_token TEXT,
    token_expires_at TIMESTAMPTZ,
    profile_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, external_id)
);

CREATE INDEX idx_user_external_provider_id ON user_external_identities(provider, external_id);
CREATE INDEX idx_user_external_user_id ON user_external_identities(user_id);
```

#### user_sessions (用戶會話表)

```sql
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) UNIQUE,
    ip_address VARCHAR(45),
    user_agent TEXT,
    device_info JSONB,
    location JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    revoked_at TIMESTAMPTZ,
    last_activity_at TIMESTAMPTZ
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);
CREATE INDEX idx_user_sessions_token ON user_sessions(refresh_token_hash);
```

### 2.2 授權與權限管理

#### roles (角色表)

```sql
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    is_system_role BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_roles_name ON roles(name);
```

#### permissions (權限表)

```sql
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (resource, action)
);

CREATE INDEX idx_permissions_resource_action ON permissions(resource, action);
```

#### role_permissions (角色權限關聯表)

```sql
CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission_id ON role_permissions(permission_id);
```

#### user_roles (用戶角色關聯表)

```sql
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
```

### 2.3 組織與租戶

#### organizations (組織表)

```sql
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    domain VARCHAR(255),
    logo_url TEXT,
    settings JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_organizations_name ON organizations(name);
CREATE INDEX idx_organizations_domain ON organizations(domain);
```

#### organization_members (組織成員表)

```sql
CREATE TABLE organization_members (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL, -- "owner", "admin", "member"
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (organization_id, user_id)
);

CREATE INDEX idx_org_members_org_id ON organization_members(organization_id);
CREATE INDEX idx_org_members_user_id ON organization_members(user_id);
```

#### teams (團隊表)

```sql
CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, name)
);

CREATE INDEX idx_teams_organization_id ON teams(organization_id);
```

#### team_members (團隊成員表)

```sql
CREATE TABLE team_members (
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL, -- "lead", "member"
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (team_id, user_id)
);

CREATE INDEX idx_team_members_team_id ON team_members(team_id);
CREATE INDEX idx_team_members_user_id ON team_members(user_id);
```

### 2.4 API 和令牌管理

#### api_clients (API 客戶端表)

```sql
CREATE TABLE api_clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    client_id VARCHAR(100) UNIQUE NOT NULL,
    client_secret_hash VARCHAR(255) NOT NULL,
    redirect_uris TEXT[],
    allowed_grant_types TEXT[],
    allowed_scopes TEXT[],
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);

CREATE INDEX idx_api_clients_org_id ON api_clients(organization_id);
CREATE INDEX idx_api_clients_client_id ON api_clients(client_id);
```

#### access_tokens (訪問令牌表)

```sql
CREATE TABLE access_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    client_id VARCHAR(100) REFERENCES api_clients(client_id) ON DELETE CASCADE,
    scopes TEXT[],
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    revoked_at TIMESTAMPTZ,
    session_id UUID REFERENCES user_sessions(id) ON DELETE CASCADE
);

CREATE INDEX idx_access_tokens_user_id ON access_tokens(user_id);
CREATE INDEX idx_access_tokens_client_id ON access_tokens(client_id);
CREATE INDEX idx_access_tokens_token_hash ON access_tokens(token_hash);
CREATE INDEX idx_access_tokens_expires_at ON access_tokens(expires_at);
```

### 2.5 安全與審計

#### user_security_events (用戶安全事件表)

```sql
CREATE TABLE user_security_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event_type VARCHAR(50) NOT NULL, -- "login", "logout", "password_change", "mfa_enabled", etc.
    ip_address VARCHAR(45),
    user_agent TEXT,
    device_info JSONB,
    location JSONB,
    success BOOLEAN NOT NULL,
    failure_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB
);

CREATE INDEX idx_security_events_user_id ON user_security_events(user_id);
CREATE INDEX idx_security_events_event_type ON user_security_events(event_type);
CREATE INDEX idx_security_events_created_at ON user_security_events(created_at);
```

#### audit_logs (審計日誌表)

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    resource_type VARCHAR(100) NOT NULL, -- "user", "role", "permission", etc.
    resource_id UUID,
    action VARCHAR(50) NOT NULL, -- "create", "update", "delete", "grant", "revoke"
    old_values JSONB,
    new_values JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_organization_id ON audit_logs(organization_id);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
```

#### rate_limits (速率限制表)

```sql
CREATE TABLE rate_limits (
    id VARCHAR(255) PRIMARY KEY, -- 可以是 IP, user_id, client_id 等識別符
    limit_type VARCHAR(50) NOT NULL, -- "login", "api", "registration", etc.
    counter INT NOT NULL DEFAULT 1,
    first_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reset_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_rate_limits_reset_at ON rate_limits(reset_at);
CREATE INDEX idx_rate_limits_id_type ON rate_limits(id, limit_type);
```

## 3. 資料關係圖

```
users
 ├── user_external_identities
 ├── user_sessions
 │    └── access_tokens
 ├── user_roles
 │    └── roles
 │         └── role_permissions
 │              └── permissions
 ├── organization_members
 │    └── organizations
 │         └── teams
 │              └── team_members
 └── user_security_events

api_clients
 └── access_tokens

audit_logs
```

## 4. 資料遷移與版本控制

資料庫變更通過版本化的遷移腳本進行管理，遵循以下格式：

```
migrations/
├── 000001_create_users_table.up.sql
├── 000001_create_users_table.down.sql
├── 000002_create_roles_and_permissions.up.sql
├── 000002_create_roles_and_permissions.down.sql
...
```

每個遷移腳本包含：
- 表結構變更
- 索引創建或修改
- 初始數據導入（如有必要）
- 相應的回滾操作

## 5. 效能優化考量

### 5.1 索引策略

- 為常用查詢條件創建適當的索引
- 使用複合索引優化多條件查詢
- 在大表上考慮部分索引
- 定期分析索引使用情況並優化

### 5.2 分區策略

對於大量數據的表（如 audit_logs, user_security_events），考慮按時間進行表分區：

```sql
CREATE TABLE audit_logs (
    id UUID NOT NULL,
    -- other columns
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

CREATE TABLE audit_logs_y2023m01 PARTITION OF audit_logs
    FOR VALUES FROM ('2023-01-01') TO ('2023-02-01');

-- Additional partitions
```

### 5.3 讀寫分離

- 考慮將只讀查詢導向副本數據庫
- 使用連接池和負載均衡分發查詢
- 為報表和分析用途使用專用的只讀副本

## 6. 安全考量

### 6.1 敏感數據保護

- 密碼使用 Argon2id 或 bcrypt 進行哈希處理
- 令牌哈希存儲，而非明文
- 考慮使用數據庫級別的欄位加密
- 使用 PII (Personally Identifiable Information) 識別和保護策略

### 6.2 資料訪問控制

- 採用最小權限原則的數據庫用戶
- 使用行級安全策略（Row-Level Security）
- 實施數據掩碼（Data Masking）保護敏感字段

### 6.3 數據隱私

- 設計支持「被遺忘權」(Right to be Forgotten)
- 實施數據保留策略
- 提供數據導出功能

## 7. 實施注意事項

### 7.1 初始化數據

創建必要的系統角色和權限：

```sql
-- 系統角色
INSERT INTO roles (name, description, is_system_role) VALUES
('system_admin', '系統管理員，具有所有權限', TRUE),
('org_admin', '組織管理員，可管理組織內部所有資源', TRUE),
('user', '普通用戶，基本權限', TRUE);

-- 基本權限
INSERT INTO permissions (name, description, resource, action) VALUES
('users:read', '查看用戶資訊', 'users', 'read'),
('users:create', '創建用戶', 'users', 'create'),
('users:update', '更新用戶資訊', 'users', 'update'),
('users:delete', '刪除用戶', 'users', 'delete');

-- 角色權限關聯
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.name = 'system_admin';
```

### 7.2 高併發處理

- 使用樂觀鎖處理並發更新
- 實施適當的重試策略
- 考慮使用事務隔離級別優化並發操作

### 7.3 資料一致性

- 確保關鍵操作在事務中執行
- 考慮使用外鍵維護引用完整性
- 實施資料驗證約束

## 8. 監控與維護

### 8.1 性能監控

- 追蹤慢查詢
- 監控表增長
- 定期執行 VACUUM 和 ANALYZE

### 8.2 數據備份

- 實施定期完整備份
- 配置時間點恢復 (PITR)
- 測試恢復流程 