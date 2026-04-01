CREATE TABLE IF NOT EXISTS link (
    original_url TEXT PRIMARY KEY,
    short_code  VARCHAR(10) NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_links_short_code ON link(short_code);