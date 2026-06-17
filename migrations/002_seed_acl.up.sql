-- Migration 002: Seed ACL for backend services
BEGIN;

-- Backend services get wildcard access (internal, trusted)
INSERT INTO mqtt_acl (username, topic, permission, action) VALUES
    ('svc_audio_mixer',    'voxmesh/#', 'allow', 'pubsub'),
    ('svc_presence',       'voxmesh/#', 'allow', 'pubsub'),
    ('svc_channel',        'voxmesh/#', 'allow', 'pubsub'),
    ('svc_gateway_coord',  'voxmesh/#', 'allow', 'pubsub'),
    ('svc_command',        'voxmesh/#', 'allow', 'pubsub'),
    ('svc_notification',   'voxmesh/#', 'allow', 'pubsub');

-- Each gateway's ACL is inserted dynamically on registration:
-- Pattern: ALLOW pub on voxmesh/gateways/{gw_id}/+
-- Pattern: ALLOW sub on voxmesh/gateways/{gw_id}/command
-- Pattern: ALLOW sub on voxmesh/devices/+/audio/rx/opus
-- Pattern: ALLOW pub on voxmesh/devices/+/audio/tx/opus
-- Pattern: ALLOW sub on voxmesh/channels/+/audio/mixed/opus
-- Pattern: ALLOW sub on voxmesh/system/broadcast

COMMIT;
