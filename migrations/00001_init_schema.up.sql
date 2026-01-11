CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    login VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_login ON users(login);

CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    number VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'NEW',
    accrual NUMERIC(10, 2) DEFAULT 0,
    uploaded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_number ON orders(number);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

CREATE TABLE IF NOT EXISTS balance (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    current NUMERIC(10, 2) NOT NULL DEFAULT 0,
    withdrawn NUMERIC(10, 2) NOT NULL DEFAULT 0,
    CONSTRAINT balance_current_non_negative CHECK (current >= 0)
);

CREATE TABLE IF NOT EXISTS withdrawals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_number VARCHAR(255) NOT NULL,
    sum NUMERIC(10, 2) NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT withdrawals_sum_positive CHECK (sum > 0)
);

CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals(user_id);
