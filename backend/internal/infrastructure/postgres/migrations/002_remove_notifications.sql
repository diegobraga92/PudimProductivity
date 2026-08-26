-- Drop the notifications dedup table. The async notifications worker (Phase 3)
-- and its FCM push pipeline were removed; nothing writes or reads this table
-- anymore.
DROP TABLE IF EXISTS notifications;
