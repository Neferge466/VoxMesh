BEGIN;
DELETE FROM mqtt_acl WHERE username IN (
    'svc_audio_mixer', 'svc_presence', 'svc_channel',
    'svc_gateway_coord', 'svc_command', 'svc_notification'
);
COMMIT;
