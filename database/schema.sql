-- Giveaways table
CREATE TABLE IF NOT EXISTS giveaways (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT UNIQUE NOT NULL,
    channel_id TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    host_id TEXT NOT NULL,
    prize TEXT NOT NULL,
    winners_count INTEGER NOT NULL,
    end_time INTEGER NOT NULL,
    ended INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    custom_message TEXT,
    
    -- Requirements
    role_requirement TEXT,
    invite_requirement INTEGER,
    account_age_requirement INTEGER,
    server_age_requirement INTEGER,
    captcha_requirement INTEGER DEFAULT 0,
    message_required INTEGER,
    voice_requirement INTEGER
);

-- Captcha sessions table
CREATE TABLE IF NOT EXISTS captcha_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    giveaway_id INTEGER NOT NULL,
    code TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    UNIQUE(user_id, giveaway_id),
    FOREIGN KEY (giveaway_id) REFERENCES giveaways(id) ON DELETE CASCADE
);

-- Participants table
CREATE TABLE IF NOT EXISTS participants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    giveaway_id INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    joined_at INTEGER NOT NULL,
    FOREIGN KEY (giveaway_id) REFERENCES giveaways(id) ON DELETE CASCADE,
    UNIQUE(giveaway_id, user_id)
);

-- Winners table
CREATE TABLE IF NOT EXISTS winners (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    giveaway_id INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    won_at INTEGER NOT NULL,
    FOREIGN KEY (giveaway_id) REFERENCES giveaways(id) ON DELETE CASCADE
);

-- User stats table (for tracking messages and voice time)
CREATE TABLE IF NOT EXISTS user_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    message_count INTEGER DEFAULT 0,
    voice_minutes INTEGER DEFAULT 0,
    UNIQUE(guild_id, user_id)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_giveaways_guild ON giveaways(guild_id);
CREATE INDEX IF NOT EXISTS idx_giveaways_ended ON giveaways(ended);
CREATE INDEX IF NOT EXISTS idx_participants_giveaway ON participants(giveaway_id);
CREATE INDEX IF NOT EXISTS idx_participants_user ON participants(user_id);
CREATE INDEX IF NOT EXISTS idx_winners_giveaway ON winners(giveaway_id);
CREATE INDEX IF NOT EXISTS idx_user_stats_guild_user ON user_stats(guild_id, user_id);
CREATE INDEX IF NOT EXISTS idx_captcha_sessions_user_giveaway ON captcha_sessions(user_id, giveaway_id);
