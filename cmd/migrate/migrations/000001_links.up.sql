CREATE TABLE IF NOT EXISTS links(
   id SERIAL PRIMARY KEY,
   original_url VARCHAR NOT NULL,
   short_code VARCHAR UNIQUE,
   created_at TIMESTAMP DEFAULT NOW(),
   expires_at TIMESTAMP NOT NULL,
   last_time_accessed TIMESTAMP
);

