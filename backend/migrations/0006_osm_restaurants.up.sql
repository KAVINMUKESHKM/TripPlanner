CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS restaurants (
    id BIGSERIAL PRIMARY KEY,
    source_id TEXT NOT NULL UNIQUE,
    source_url TEXT NOT NULL,
    name TEXT NOT NULL,
    amenity TEXT NOT NULL,
    cuisine TEXT,
    address TEXT,
    opening_hours TEXT,
    phone TEXT,
    website TEXT,
    location GEOGRAPHY(Point, 4326) NOT NULL
);

CREATE INDEX IF NOT EXISTS restaurants_location_gix ON restaurants USING GIST (location);
