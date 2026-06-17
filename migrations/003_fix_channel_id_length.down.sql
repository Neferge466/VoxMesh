-- Migration 003 down: Revert channels.id back to VARCHAR(32)
BEGIN;

ALTER TABLE channels
    ALTER COLUMN id TYPE VARCHAR(32);

COMMIT;
