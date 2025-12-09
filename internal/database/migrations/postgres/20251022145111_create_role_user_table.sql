-- +goose Up
-- +goose StatementBegin
CREATE TABLE role_user (
    role_id INTEGER REFERENCES roles(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    deleted_at TIMESTAMP,
    PRIMARY KEY (role_id, user_id)
);

CREATE INDEX idx_role_user_deleted_at ON role_user (deleted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_role_user_deleted_at;
DROP TABLE IF EXISTS role_user;
-- +goose StatementEnd
