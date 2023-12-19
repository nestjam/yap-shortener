BEGIN;

CREATE TABLE url(id SERIAL PRIMARY KEY,
    short_url VARCHAR(255),
    original_url TEXT UNIQUE
);

CREATE INDEX idx_url_short 
ON url(short_url);

COMMIT;