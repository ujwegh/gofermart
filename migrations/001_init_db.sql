-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users
(
    uuid          UUID PRIMARY KEY        DEFAULT uuid_generate_v4(),
    login         VARCHAR UNIQUE NOT NULL,
    password_hash VARCHAR        NOT NULL,
    created_at    TIMESTAMP      NOT NULL DEFAULT NOW()
);

CREATE TYPE status AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');

CREATE TABLE orders
(
    id         VARCHAR PRIMARY KEY,
    user_uuid  UUID      NOT NULL REFERENCES users (uuid) ON DELETE CASCADE,
    status     status    NOT NULL DEFAULT 'NEW',
    accrual    NUMERIC,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    constraint accrual_positive check (accrual > 0)
);

CREATE TABLE withdrawals
(
    id         BIGSERIAL PRIMARY KEY,
    user_uuid  UUID      NOT NULL REFERENCES users (uuid) ON DELETE CASCADE,
    order_id   VARCHAR   NOT NULL,
    amount     NUMERIC   NOT NULL default 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    constraint amount_positive check (amount > 0)
);


CREATE TABLE wallets
(
    id         BIGSERIAL PRIMARY KEY,
    user_uuid  UUID UNIQUE REFERENCES users (uuid) ON DELETE CASCADE,
    credits    NUMERIC   NOT NULL DEFAULT 0,
    debits     NUMERIC   NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    constraint credits_positive check (credits >= 0),
    constraint debits_positive check (debits >= 0)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE wallets;
DROP TABLE withdrawals;
DROP TABLE orders;
DROP TYPE status;
DROP TABLE users;

-- +goose StatementEnd
