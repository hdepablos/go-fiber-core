-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_banks_name_gin_trgm ON banks USING GIN (name gin_trgm_ops);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_banks_name_gin_trgm;
-- +goose StatementEnd

