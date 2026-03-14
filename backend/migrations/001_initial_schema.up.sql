-- Banking Application - Initial Schema
-- All monetary values stored as BIGINT (cents) to avoid floating point issues

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table
CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      VARCHAR(255) NOT NULL UNIQUE,
    full_name  VARCHAR(255) NOT NULL,
    pin_hash   VARCHAR(255) NOT NULL,
    status     VARCHAR(20)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended')),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_email ON users (email);

-- Accounts table
CREATE TABLE accounts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID         NOT NULL REFERENCES users(id),
    account_type VARCHAR(20)  NOT NULL CHECK (account_type IN ('savings', 'checking')),
    balance      BIGINT       NOT NULL DEFAULT 0 CHECK (balance >= 0),
    currency     VARCHAR(3)   NOT NULL DEFAULT 'USD',
    status       VARCHAR(20)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'frozen', 'closed')),
    version      INT          NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_accounts_user_id ON accounts (user_id);

-- Transactions table
CREATE TABLE transactions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_account_id  UUID REFERENCES accounts(id),
    to_account_id    UUID REFERENCES accounts(id),
    amount           BIGINT       NOT NULL CHECK (amount > 0),
    currency         VARCHAR(3)   NOT NULL DEFAULT 'USD',
    transaction_type VARCHAR(20)  NOT NULL CHECK (transaction_type IN ('deposit', 'withdrawal', 'transfer')),
    status           VARCHAR(20)  NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'failed', 'reversed')),
    idempotency_key  VARCHAR(255) NOT NULL UNIQUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT chk_transaction_accounts CHECK (
        (transaction_type = 'deposit'    AND to_account_id IS NOT NULL) OR
        (transaction_type = 'withdrawal' AND from_account_id IS NOT NULL) OR
        (transaction_type = 'transfer'   AND from_account_id IS NOT NULL AND to_account_id IS NOT NULL)
    )
);

CREATE INDEX idx_transactions_from ON transactions (from_account_id);
CREATE INDEX idx_transactions_to   ON transactions (to_account_id);
CREATE INDEX idx_transactions_idempotency ON transactions (idempotency_key);

-- Saga state table (for distributed transaction orchestration)
CREATE TABLE saga_state (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    saga_type    VARCHAR(50)  NOT NULL,
    status       VARCHAR(20)  NOT NULL DEFAULT 'started' CHECK (status IN ('started', 'completed', 'compensating', 'failed')),
    payload      JSONB        NOT NULL DEFAULT '{}',
    current_step INT          NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);
