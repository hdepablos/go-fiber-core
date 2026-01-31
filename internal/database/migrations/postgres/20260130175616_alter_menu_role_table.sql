-- +goose Up
-- +goose StatementBegin

-- agregar columna
ALTER TABLE menu_role
ADD COLUMN operator_id INT;

-- foreign key hacia users
ALTER TABLE menu_role
ADD CONSTRAINT fk_menu_role_operator
FOREIGN KEY (operator_id)
REFERENCES users(id)
ON DELETE SET NULL;

-- índice para performance
CREATE INDEX idx_menu_role_operator_id ON menu_role(operator_id);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_menu_role_operator_id;

ALTER TABLE menu_role
DROP CONSTRAINT IF EXISTS fk_menu_role_operator;

ALTER TABLE menu_role
DROP COLUMN IF EXISTS operator_id;

-- +goose StatementEnd
