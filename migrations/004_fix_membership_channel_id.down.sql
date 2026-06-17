-- Migration 004 down: revert channel_memberships.channel_id back to VARCHAR(32)
BEGIN;

ALTER TABLE channel_memberships
    DROP CONSTRAINT IF EXISTS channel_memberships_channel_id_fkey;

-- Truncate any data longer than 32 chars (dev DB only, production would need migration)
DELETE FROM channel_memberships WHERE length(channel_id) > 32;

ALTER TABLE channel_memberships
    ALTER COLUMN channel_id TYPE VARCHAR(32);

ALTER TABLE channel_memberships
    ADD CONSTRAINT channel_memberships_channel_id_fkey
    FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE;

COMMIT;
