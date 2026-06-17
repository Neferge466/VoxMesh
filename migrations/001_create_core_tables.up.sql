-- VoxMesh Database Schema v1.0.0
-- Migration 001: Core tables

BEGIN;

-- ==========================================================================
-- Users & Authentication
-- ==========================================================================
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(32)  NOT NULL UNIQUE,
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    display_name    VARCHAR(64),
    avatar_url      VARCHAR(512),
    is_active       BOOLEAN      NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_login_at   TIMESTAMPTZ
);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);

CREATE TABLE roles (
    id   SERIAL PRIMARY KEY,
    name VARCHAR(32) NOT NULL UNIQUE
);
INSERT INTO roles (name) VALUES ('admin'), ('moderator'), ('user'), ('guest');

CREATE TABLE user_roles (
    user_id UUID    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(255) NOT NULL UNIQUE,
    device_info VARCHAR(256),
    expires_at  TIMESTAMPTZ  NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);

-- ==========================================================================
-- Channels
-- ==========================================================================
CREATE TABLE channels (
    id              VARCHAR(64)  PRIMARY KEY,
    parent_id       VARCHAR(32)  REFERENCES channels(id) ON DELETE SET NULL,
    name            VARCHAR(64)  NOT NULL,
    description     VARCHAR(512),
    sort_order      INTEGER      NOT NULL DEFAULT 0,
    max_users       INTEGER      NOT NULL DEFAULT -1,   -- -1 = unlimited
    codec_quality   VARCHAR(16)  NOT NULL DEFAULT 'high', -- low, medium, high
    password_hash   VARCHAR(255),
    is_temporary    BOOLEAN      NOT NULL DEFAULT false,
    temporary_ttl_min INTEGER    NOT NULL DEFAULT 60,
    created_by      UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
CREATE INDEX idx_channels_parent ON channels(parent_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_channels_deleted ON channels(deleted_at);

CREATE TABLE channel_memberships (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id  VARCHAR(64)  NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    client_type VARCHAR(16)  NOT NULL CHECK (client_type IN ('web', 'embedded')),
    device_id   VARCHAR(64),
    joined_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    left_at     TIMESTAMPTZ
);
CREATE INDEX idx_channel_members_active ON channel_memberships(channel_id) WHERE left_at IS NULL;
-- A user can rejoin after leaving (new row)
CREATE UNIQUE INDEX idx_channel_members_unique ON channel_memberships(user_id, channel_id, COALESCE(left_at, '1970-01-01'::timestamptz));

-- ==========================================================================
-- Gateways & Devices
-- ==========================================================================
CREATE TABLE gateways (
    id                VARCHAR(32)  PRIMARY KEY,
    name              VARCHAR(64)  NOT NULL,
    api_key_hash      VARCHAR(255) NOT NULL UNIQUE,
    status            VARCHAR(16)  NOT NULL DEFAULT 'offline', -- online, degraded, offline
    ip_address        VARCHAR(45),
    version           VARCHAR(16),
    capabilities      JSONB,
    last_heartbeat_at TIMESTAMPTZ,
    registered_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE mesh_devices (
    id               VARCHAR(32) PRIMARY KEY,
    gateway_id       VARCHAR(32) NOT NULL REFERENCES gateways(id) ON DELETE CASCADE,
    name             VARCHAR(64),
    firmware_version VARCHAR(16),
    capabilities     JSONB,
    last_status      JSONB,
    last_seen_at     TIMESTAMPTZ,
    registered_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_mesh_devices_gateway ON mesh_devices(gateway_id);

-- ==========================================================================
-- API Keys (gateway & service-to-service auth)
-- ==========================================================================
CREATE TABLE api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_prefix  VARCHAR(8)   NOT NULL UNIQUE,  -- gwk_ for gateway, svc_ for service
    key_hash    VARCHAR(255) NOT NULL UNIQUE,
    entity_type VARCHAR(16)  NOT NULL CHECK (entity_type IN ('gateway', 'service')),
    entity_id   VARCHAR(64)  NOT NULL,
    permissions JSONB,
    expires_at  TIMESTAMPTZ,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_api_keys_entity ON api_keys(entity_type, entity_id);

-- ==========================================================================
-- MQTT ACL (EMQX PostgreSQL ACL plugin)
-- ==========================================================================
CREATE TABLE mqtt_acl (
    id         SERIAL PRIMARY KEY,
    username   VARCHAR(64)  NOT NULL,
    topic      VARCHAR(256) NOT NULL,
    permission VARCHAR(8)   NOT NULL CHECK (permission IN ('allow', 'deny')),
    action     VARCHAR(6)   NOT NULL CHECK (action IN ('pub', 'sub', 'pubsub'))
);
CREATE INDEX idx_mqtt_acl_username ON mqtt_acl(username);

COMMIT;
