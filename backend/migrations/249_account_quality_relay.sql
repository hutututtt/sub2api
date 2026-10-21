-- The public relay is shared by all group inspection settings. Probe keys are
-- derived per account with a server secret; they never enter the business key table.
CREATE TABLE IF NOT EXISTS account_quality_relay_config (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    base_url TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS account_quality_relay_leases (
    run_id BIGINT NOT NULL REFERENCES account_quality_runs(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '11 minutes',
    requests_used INTEGER NOT NULL DEFAULT 0,
    in_flight BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (run_id, account_id)
);
CREATE INDEX IF NOT EXISTS account_quality_relay_account_idx ON account_quality_relay_leases(account_id);
