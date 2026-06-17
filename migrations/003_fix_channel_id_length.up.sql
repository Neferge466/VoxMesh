-- Migration 003: Fix channels.id length for UUID support
BEGIN;

ALTER TABLE channels
    ALTER COLUMN id TYPE VARCHAR(64);

COMMIT;
