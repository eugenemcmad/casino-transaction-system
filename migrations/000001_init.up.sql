CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    type VARCHAR(10) NOT NULL CHECK (type IN ('bet', 'win')), -- "type" is more concise and matches the domain model field, and acceptable for PostgreSQL
    amount NUMERIC(15, 2) NOT NULL, -- But in a production system, it's better to use BIGINT for cents
    timestamp TIMESTAMP WITH TIME ZONE, -- Nullable to avoid losing data if event time is missing
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_id_type_time ON transactions(user_id, type, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_timestamp ON transactions(timestamp DESC);
