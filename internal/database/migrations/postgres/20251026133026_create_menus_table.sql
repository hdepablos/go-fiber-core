-- +goose Up
-- +goose StatementBegin
CREATE TABLE menus (
    id SERIAL PRIMARY KEY,
    item_type VARCHAR(20) NOT NULL CHECK (item_type IN ('link', 'separator', 'group', 'line')),
    item_name VARCHAR(100) NOT NULL,
    to_path VARCHAR(255),
    icon VARCHAR(100),
    parent_id INT REFERENCES menus(id) ON DELETE CASCADE,
    order_index INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
	created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    deleted_at TIMESTAMP
);

-- 🔒 Restricción única ignorando mayúsculas/minúsculas
CREATE UNIQUE INDEX unique_menus_parent_item_name_ci
    ON menus (parent_id, LOWER(item_name));

CREATE INDEX idx_menus_deleted_at ON menus (deleted_at);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS unique_menus_parent_item_name_ci;
DROP INDEX IF EXISTS idx_menus_deleted_at;
DROP TABLE IF EXISTS menus;
-- +goose StatementEnd
