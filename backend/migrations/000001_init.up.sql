-- 000001_init: базовая схема системы мониторинга парковки.
-- Порядок таблиц важен: parking_lots -> cameras -> parking_spots ->
-- spot_events -> spot_status.

CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL DEFAULT 'admin'
                  CHECK (role IN ('admin')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE parking_lots (
    id           BIGSERIAL PRIMARY KEY,
    name         TEXT             NOT NULL,
    address      TEXT             NOT NULL DEFAULT '',
    kind         TEXT             NOT NULL
                 CHECK (kind IN ('lot', 'roadside')),
    center_lat   DOUBLE PRECISION NOT NULL
                 CHECK (center_lat BETWEEN -90 AND 90),
    center_lon   DOUBLE PRECISION NOT NULL
                 CHECK (center_lon BETWEEN -180 AND 180),
    entrance_lat DOUBLE PRECISION
                 CHECK (entrance_lat BETWEEN -90 AND 90),
    entrance_lon DOUBLE PRECISION
                 CHECK (entrance_lon BETWEEN -180 AND 180),
    radius_m     INTEGER          NOT NULL DEFAULT 100
                 CHECK (radius_m > 0),
    geojson      JSONB,
    created_at   TIMESTAMPTZ      NOT NULL DEFAULT now()
);

CREATE TABLE cameras (
    id            BIGSERIAL PRIMARY KEY,
    lot_id        BIGINT      NOT NULL
                  REFERENCES parking_lots (id) ON DELETE CASCADE,
    name          TEXT        NOT NULL,
    source_type   TEXT        NOT NULL
                  CHECK (source_type IN ('rtsp', 'webcam', 'file')),
    source        TEXT        NOT NULL,
    status        TEXT        NOT NULL DEFAULT 'offline'
                  CHECK (status IN ('online', 'offline', 'error')),
    last_frame_at TIMESTAMPTZ,
    fps           INTEGER     CHECK (fps IS NULL OR fps > 0),
    resolution    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE parking_spots (
    id          BIGSERIAL PRIMARY KEY,
    lot_id      BIGINT      NOT NULL
                REFERENCES parking_lots (id) ON DELETE CASCADE,
    camera_id   BIGINT
                REFERENCES cameras (id) ON DELETE SET NULL,
    code        TEXT        NOT NULL,
    -- Нормализованные координаты полигона на кадре (0..1).
    polygon     JSONB       NOT NULL,
    map_lat     DOUBLE PRECISION
                CHECK (map_lat BETWEEN -90 AND 90),
    map_lon     DOUBLE PRECISION
                CHECK (map_lon BETWEEN -180 AND 180),
    map_geojson JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (lot_id, code)
);

CREATE TABLE spot_events (
    id            BIGSERIAL PRIMARY KEY,
    spot_id       BIGINT      NOT NULL
                  REFERENCES parking_spots (id) ON DELETE CASCADE,
    camera_id     BIGINT
                  REFERENCES cameras (id) ON DELETE SET NULL,
    status        TEXT        NOT NULL
                  CHECK (status IN ('free', 'occupied', 'unknown')),
    confidence    REAL        CHECK (confidence BETWEEN 0 AND 1),
    captured_at   TIMESTAMPTZ NOT NULL,
    snapshot_path TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Текущий статус места. last_event_id намеренно без FK: таблицы ссылаются
-- друг на друга, целостность обеспечивает приложение.
CREATE TABLE spot_status (
    spot_id       BIGINT      PRIMARY KEY
                  REFERENCES parking_spots (id) ON DELETE CASCADE,
    status        TEXT        NOT NULL DEFAULT 'unknown'
                  CHECK (status IN ('free', 'occupied', 'unknown')),
    confidence    REAL        CHECK (confidence BETWEEN 0 AND 1),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_event_id BIGINT
);

CREATE INDEX spot_events_spot_captured_idx
    ON spot_events (spot_id, captured_at DESC);
CREATE INDEX spot_status_updated_idx
    ON spot_status (updated_at DESC);
CREATE INDEX parking_spots_lot_idx
    ON parking_spots (lot_id);
CREATE INDEX cameras_lot_idx
    ON cameras (lot_id);

-- У каждого нового места сразу появляется строка статуса unknown,
-- чтобы выборка свободных мест не теряла места без событий.
CREATE FUNCTION parking_spots_init_status() RETURNS trigger AS $$
BEGIN
    INSERT INTO spot_status (spot_id) VALUES (NEW.id)
    ON CONFLICT (spot_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER parking_spots_init_status
    AFTER INSERT ON parking_spots
    FOR EACH ROW
    EXECUTE FUNCTION parking_spots_init_status();
