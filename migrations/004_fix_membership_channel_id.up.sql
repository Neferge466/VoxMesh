-- Migration 004: Fix channel_memberships.channel_id to match channels.id (VARCHAR(64))
BEGIN;

-- Drop FK so we can alter the column
ALTER TABLE channel_memberships
    DROP CONSTRAINT IF EXISTS channel_memberships_channel_id_fkey;

ALTER TABLE channel_memberships
    ALTER COLUMN channel_id TYPE VARCHAR(64);

-- Re-add FK with same ON DELETE CASCADE
ALTER TABLE channel_memberships
    ADD CONSTRAINT channel_memberships_channel_id_fkey
    FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE;

COMMIT;
