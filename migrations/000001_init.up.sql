CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    type VARCHAR(10) NOT NULL CHECK (type IN ('bet', 'win')),
    amount NUMERIC(15, 2) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Idempotency: prevent exact same transaction from being recorded twice.
    -- Using NULLS NOT DISTINCT (Postgres 15+) to handle cases with missing timestamps.
    UNIQUE NULLS NOT DISTINCT (user_id, type, amount, timestamp)
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_id_type_time ON transactions(user_id, type, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_timestamp ON transactions(timestamp DESC);
