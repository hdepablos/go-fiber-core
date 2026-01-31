-- +goose Up
-- +goose StatementBegin

-- agregar columna
ALTER TABLE menu_user
ADD COLUMN operator_id INT;

-- foreign key hacia users
ALTER TABLE menu_user
ADD CONSTRAINT fk_menu_user_operator
FOREIGN KEY (operator_id)
REFERENCES users(id)
ON DELETE SET NULL;

-- índice para performance
CREATE INDEX idx_menu_user_operator_id ON menu_user(operator_id);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_menu_user_operator_id;

ALTER TABLE menu_user
DROP CONSTRAINT IF EXISTS fk_menu_user_operator;

ALTER TABLE menu_user
DROP COLUMN IF EXISTS operator_id;

-- +goose StatementEnd
