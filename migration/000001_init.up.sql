CREATE TABLE url(id SERIAL PRIMARY KEY,
    short_url VARCHAR(255) UNIQUE NOT NULL,
    original_url TEXT UNIQUE NOT NULL
);