-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users
(
    uuid       UUID PRIMARY KEY        DEFAULT uuid_generate_v4(),
    email      VARCHAR UNIQUE NOT NULL,
    name       VARCHAR        NOT NULL,
    password   VARCHAR        NOT NULL,
    created_at TIMESTAMP      NOT NULL DEFAULT NOW()
);



-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
