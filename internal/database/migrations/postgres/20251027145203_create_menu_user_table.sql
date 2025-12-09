-- +goose Up
-- +goose StatementBegin
CREATE TABLE menu_user (
    id SERIAL PRIMARY KEY,
    menu_id INT NOT NULL REFERENCES menus(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    deleted_at TIMESTAMP
);

-- Índice único combinado (menu_id, user_id)
CREATE UNIQUE INDEX idx_menu_user_unique ON menu_user (menu_id, user_id);

-- Índice para soft delete
CREATE INDEX idx_menu_user_deleted_at ON menu_user (deleted_at);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_menu_user_deleted_at;
DROP INDEX IF EXISTS idx_menu_user_unique;
DROP TABLE IF EXISTS menu_user;
-- +goose StatementEnd
