-- +goose Up
-- +goose StatementBegin
-- SQL in this section is executed when the migration is applied.

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_users_deleted_at ON users (deleted_at);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
-- SQL in this section is executed when the migration is rolled back.
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
