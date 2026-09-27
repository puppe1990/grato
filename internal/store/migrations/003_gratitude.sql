-- Gratitude domain: daily checkpoints, resonance tags and evening rituals.
--
-- `day` is denormalized from happened_at on purpose: every calendar query
-- (today, month grid, streaks, year counts) then compares plain
-- 'YYYY-MM-DD' strings, which is index-friendly and immune to the
-- timezone/format drift of raw DATETIME binding.

ALTER TABLE users ADD COLUMN name TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN intention TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS moments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    moment_of_day TEXT NOT NULL DEFAULT 'tarde',
    mood TEXT NOT NULL DEFAULT 'sereno',
    title TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    serenity INTEGER NOT NULL DEFAULT 3,
    day TEXT NOT NULL,
    happened_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_moments_user_day ON moments(user_id, day DESC);
CREATE INDEX IF NOT EXISTS idx_moments_user_happened ON moments(user_id, happened_at DESC);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    UNIQUE (user_id, slug)
);

CREATE TABLE IF NOT EXISTS moment_tags (
    moment_id INTEGER NOT NULL REFERENCES moments(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (moment_id, tag_id)
);

CREATE TABLE IF NOT EXISTS rituals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 0,
    remind_at TEXT NOT NULL DEFAULT '21:30',
    UNIQUE (user_id, kind)
);
