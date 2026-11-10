DROP TRIGGER IF EXISTS parking_spots_init_status ON parking_spots;
DROP FUNCTION IF EXISTS parking_spots_init_status();

DROP TABLE IF EXISTS spot_status;
DROP TABLE IF EXISTS spot_events;
DROP TABLE IF EXISTS parking_spots;
DROP TABLE IF EXISTS cameras;
DROP TABLE IF EXISTS parking_lots;
DROP TABLE IF EXISTS users;
