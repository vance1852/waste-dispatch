-- Users table
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name     TEXT NOT NULL DEFAULT '',
    phone         TEXT NOT NULL DEFAULT '',
    email         TEXT NOT NULL DEFAULT '',
    role          TEXT NOT NULL DEFAULT 'resident',
    status        TEXT NOT NULL DEFAULT 'active',
    created_at    DATETIME NOT NULL,
    updated_at    DATETIME NOT NULL,
    version       INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   TEXT NOT NULL UNIQUE,
    user_agent   TEXT NOT NULL DEFAULT '',
    ip_address   TEXT NOT NULL DEFAULT '',
    expires_at   DATETIME NOT NULL,
    created_at   DATETIME NOT NULL,
    last_seen_at DATETIME NOT NULL,
    revoked_at   DATETIME
);

CREATE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- Vehicles table
CREATE TABLE IF NOT EXISTS vehicles (
    id               TEXT PRIMARY KEY,
    plate_number     TEXT NOT NULL UNIQUE,
    type             TEXT NOT NULL,
    capacity_kg      REAL NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'idle',
    driver_id        TEXT,
    last_serviced_at DATETIME,
    notes            TEXT NOT NULL DEFAULT '',
    created_at       DATETIME NOT NULL,
    updated_at       DATETIME NOT NULL,
    version          INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_vehicles_status ON vehicles(status);
CREATE INDEX IF NOT EXISTS idx_vehicles_plate ON vehicles(plate_number);

-- Collection points table
CREATE TABLE IF NOT EXISTS collection_points (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    address          TEXT NOT NULL,
    latitude         REAL NOT NULL DEFAULT 0,
    longitude        REAL NOT NULL DEFAULT 0,
    district         TEXT NOT NULL DEFAULT '',
    waste_categories TEXT NOT NULL DEFAULT '[]',
    capacity_kg      REAL NOT NULL DEFAULT 0,
    current_load_kg  REAL NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'active',
    notes            TEXT NOT NULL DEFAULT '',
    created_at       DATETIME NOT NULL,
    updated_at       DATETIME NOT NULL,
    version          INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_points_status ON collection_points(status);
CREATE INDEX IF NOT EXISTS idx_points_district ON collection_points(district);

-- Collection tasks table
CREATE TABLE IF NOT EXISTS collection_tasks (
    id                   TEXT PRIMARY KEY,
    point_id             TEXT NOT NULL REFERENCES collection_points(id),
    vehicle_id           TEXT NOT NULL REFERENCES vehicles(id),
    driver_id            TEXT NOT NULL DEFAULT '',
    status               TEXT NOT NULL DEFAULT 'scheduled',
    priority             TEXT NOT NULL DEFAULT 'normal',
    scheduled_at         DATETIME NOT NULL,
    started_at           DATETIME,
    completed_at         DATETIME,
    collected_weight_kg  REAL NOT NULL DEFAULT 0,
    notes                TEXT NOT NULL DEFAULT '',
    failure_reason       TEXT NOT NULL DEFAULT '',
    created_by           TEXT NOT NULL DEFAULT '',
    created_at           DATETIME NOT NULL,
    updated_at           DATETIME NOT NULL,
    version              INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON collection_tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_vehicle_id ON collection_tasks(vehicle_id);
CREATE INDEX IF NOT EXISTS idx_tasks_driver_id ON collection_tasks(driver_id);
CREATE INDEX IF NOT EXISTS idx_tasks_point_id ON collection_tasks(point_id);
CREATE INDEX IF NOT EXISTS idx_tasks_scheduled_at ON collection_tasks(scheduled_at);

-- Incidents table
CREATE TABLE IF NOT EXISTS incidents (
    id          TEXT PRIMARY KEY,
    type        TEXT NOT NULL,
    severity    TEXT NOT NULL DEFAULT 'medium',
    status      TEXT NOT NULL DEFAULT 'open',
    point_id    TEXT,
    vehicle_id  TEXT,
    task_id     TEXT,
    reported_by TEXT NOT NULL DEFAULT '',
    assigned_to TEXT,
    description TEXT NOT NULL DEFAULT '',
    resolution  TEXT NOT NULL DEFAULT '',
    occurred_at DATETIME NOT NULL,
    resolved_at DATETIME,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL,
    version     INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);
CREATE INDEX IF NOT EXISTS idx_incidents_type ON incidents(type);
CREATE INDEX IF NOT EXISTS idx_incidents_point_id ON incidents(point_id);
CREATE INDEX IF NOT EXISTS idx_incidents_vehicle_id ON incidents(vehicle_id);

-- Resident credits table
CREATE TABLE IF NOT EXISTS resident_credits (
    id          TEXT PRIMARY KEY,
    resident_id TEXT NOT NULL UNIQUE,
    balance     INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL,
    version     INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_credits_resident_id ON resident_credits(resident_id);

-- Credit transactions table
CREATE TABLE IF NOT EXISTS credit_transactions (
    id              TEXT PRIMARY KEY,
    resident_id     TEXT NOT NULL REFERENCES resident_credits(resident_id),
    type            TEXT NOT NULL,
    amount          INTEGER NOT NULL,
    balance_after   INTEGER NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    ref_id          TEXT,
    description     TEXT NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL,
    created_by      TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_credit_tx_resident_id ON credit_transactions(resident_id);
CREATE INDEX IF NOT EXISTS idx_credit_tx_idempotency_key ON credit_transactions(idempotency_key);

-- Audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id          TEXT PRIMARY KEY,
    actor_id    TEXT NOT NULL DEFAULT '',
    actor_role  TEXT NOT NULL DEFAULT '',
    action      TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id   TEXT NOT NULL,
    old_value   TEXT,
    new_value   TEXT,
    request_id  TEXT,
    ip_address  TEXT,
    created_at  DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_logs(created_at);
