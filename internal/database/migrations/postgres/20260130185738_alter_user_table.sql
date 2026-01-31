-- +goose Up
-- +goose StatementBegin

ALTER TABLE users
ADD COLUMN operator_id BIGINT NULL REFERENCES users(id);

CREATE INDEX idx_users_operator_id ON users(operator_id);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_users_operator_id;
ALTER TABLE users DROP COLUMN IF EXISTS operator_id;

-- +goose StatementEnd
