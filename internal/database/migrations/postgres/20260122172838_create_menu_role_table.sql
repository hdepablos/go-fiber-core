-- +goose Up
-- +goose StatementBegin
CREATE TABLE menu_role (
    id SERIAL PRIMARY KEY,
    menu_id INT NOT NULL REFERENCES menus(id) ON DELETE CASCADE,
    role_id INT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    deleted_at TIMESTAMP
);

-- Índice único combinado (menu_id, role_id)
CREATE UNIQUE INDEX idx_menu_role_unique ON menu_role (menu_id, role_id);

-- Índice para soft delete
CREATE INDEX idx_menu_role_deleted_at ON menu_role (deleted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_menu_role_deleted_at;
DROP INDEX IF EXISTS idx_menu_role_unique;
DROP TABLE IF EXISTS menu_role;
-- +goose StatementEnd
