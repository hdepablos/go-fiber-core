-- +goose Up
-- +goose StatementBegin
ALTER TABLE bulk_jobs
DROP COLUMN IF EXISTS total_processed_items;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE bulk_jobs
ADD COLUMN IF NOT EXISTS total_processed_items INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd
