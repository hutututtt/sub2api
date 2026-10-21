-- Advertising attribution is independent of upstream channels and affiliate rebates.
CREATE TABLE IF NOT EXISTS promotion_settings (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    attribution_model VARCHAR(16) NOT NULL DEFAULT 'last_touch' CHECK (attribution_model IN ('first_touch', 'last_touch')),
    window_days INTEGER NOT NULL DEFAULT 30 CHECK (window_days BETWEEN 1 AND 90)
);
INSERT INTO promotion_settings(id) VALUES (1) ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS promotion_channels (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    website VARCHAR(255) NOT NULL DEFAULT '',
    placement VARCHAR(255) NOT NULL DEFAULT '',
    landing_path VARCHAR(1000) NOT NULL DEFAULT '/register',
    notes TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS promotion_visitors (
    id BIGSERIAL PRIMARY KEY,
    token_hash CHAR(64) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS promotion_visits (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL UNIQUE,
    visitor_id BIGINT NOT NULL REFERENCES promotion_visitors(id),
    channel_id BIGINT NOT NULL REFERENCES promotion_channels(id),
    is_effective BOOLEAN NOT NULL,
    referrer_host VARCHAR(255) NOT NULL DEFAULT '',
    landing_path VARCHAR(1000) NOT NULL DEFAULT '',
    device VARCHAR(20) NOT NULL DEFAULT 'desktop',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_promotion_visits_channel_time ON promotion_visits(channel_id, created_at);
CREATE INDEX IF NOT EXISTS idx_promotion_visits_visitor_time ON promotion_visits(visitor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_promotion_visits_time ON promotion_visits(created_at);

CREATE TABLE IF NOT EXISTS promotion_attributions (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    visitor_id BIGINT NOT NULL REFERENCES promotion_visitors(id),
    channel_id BIGINT NOT NULL REFERENCES promotion_channels(id),
    first_channel_id BIGINT NOT NULL REFERENCES promotion_channels(id),
    last_channel_id BIGINT NOT NULL REFERENCES promotion_channels(id),
    visit_id BIGINT NOT NULL REFERENCES promotion_visits(id),
    attribution_model VARCHAR(16) NOT NULL,
    window_days INTEGER NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_promotion_attributions_channel_time ON promotion_attributions(channel_id, registered_at);
CREATE INDEX IF NOT EXISTS idx_promotion_attributions_visitor ON promotion_attributions(visitor_id);

CREATE TABLE IF NOT EXISTS promotion_expenses (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES promotion_channels(id),
    spent_on DATE NOT NULL,
    amount NUMERIC(20,4) NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    notes VARCHAR(1000) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_promotion_expenses_channel_date ON promotion_expenses(channel_id, spent_on);
