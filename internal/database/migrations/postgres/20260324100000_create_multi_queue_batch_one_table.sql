-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS multi_queue_batch_one_table (
    id BIGSERIAL PRIMARY KEY,
    status VARCHAR(64) NOT NULL DEFAULT 'pending',
    run_id TEXT NULL,
    detail TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mqb1t_status_id ON multi_queue_batch_one_table (status, id);
CREATE INDEX IF NOT EXISTS idx_mqb1t_run_id_status_id ON multi_queue_batch_one_table (run_id, status, id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS multi_queue_batch_one_table;

-- +goose StatementEnd
