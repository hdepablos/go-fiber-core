-- +goose Up
-- +goose StatementBegin

CREATE TYPE bulk_job_status AS ENUM (
    'IMPORTING',
    'IMPORTED',
    'PROCESSING',
    'PROCESSED',
    'ERROR_IMPORTING',
    'ERROR_PROCESS',
    'PROCESSED_WITH_DETAILS'
);

CREATE TYPE log_severity AS ENUM (
    'INFO',
    'WARNING',
    'ERROR'
);

CREATE TABLE bulk_jobs (
    id BIGSERIAL PRIMARY KEY,
    operator_id BIGINT NOT NULL,
    branch_id BIGINT NOT NULL DEFAULT 0,
    key_code VARCHAR(20),
    ref_code VARCHAR(255) NOT NULL,
    status_code bulk_job_status NOT NULL DEFAULT 'IMPORTING',
    total_detail_items INTEGER NOT NULL DEFAULT 0,
    total_processed_items INTEGER NOT NULL DEFAULT 0,
    file_name VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX ux_bulk_jobs_key_code
ON bulk_jobs (key_code)
WHERE key_code IS NOT NULL;

CREATE INDEX idx_bulk_jobs_operator_id
ON bulk_jobs (operator_id);

CREATE INDEX idx_bulk_jobs_branch_id
ON bulk_jobs (branch_id);

CREATE INDEX idx_bulk_jobs_ref_code
ON bulk_jobs (ref_code);

CREATE INDEX idx_bulk_jobs_status_code
ON bulk_jobs (status_code);


CREATE TABLE bulk_job_items (
    id BIGSERIAL PRIMARY KEY,
    bulk_job_id BIGINT NOT NULL REFERENCES bulk_jobs(id) ON DELETE CASCADE,
    row_number INTEGER NOT NULL,
    reference_key VARCHAR(255) NOT NULL,
    data JSONB NOT NULL,
    status_code bulk_job_status NOT NULL DEFAULT 'IMPORTED',
    last_detail_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (bulk_job_id, row_number)
);

CREATE INDEX idx_bulk_job_items_job_id
ON bulk_job_items (bulk_job_id);

CREATE INDEX idx_bulk_job_items_reference_key
ON bulk_job_items (reference_key);

CREATE INDEX idx_bulk_job_items_status_code
ON bulk_job_items (status_code);


CREATE TABLE bulk_job_item_messages (
    id BIGSERIAL PRIMARY KEY,
    bulk_job_item_id BIGINT NOT NULL REFERENCES bulk_job_items(id) ON DELETE CASCADE,
    severity log_severity NOT NULL DEFAULT 'INFO',
    code VARCHAR(64),
    detail_message TEXT NOT NULL,
    meta JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bulk_job_item_messages_item_id
ON bulk_job_item_messages (bulk_job_item_id);

CREATE INDEX idx_bulk_job_item_messages_severity
ON bulk_job_item_messages (severity);


CREATE TABLE bulk_job_configs (
    id BIGSERIAL PRIMARY KEY,
    operator_id BIGINT NOT NULL,
    ref_code VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    config JSONB NOT NULL,
    archived_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (operator_id, ref_code)
);

CREATE UNIQUE INDEX ux_bulk_job_configs_active
ON bulk_job_configs (operator_id, ref_code)
WHERE is_active = TRUE AND archived_at IS NULL;

CREATE INDEX idx_bulk_job_configs_ref_code
ON bulk_job_configs (ref_code);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS bulk_job_configs;
DROP TABLE IF EXISTS bulk_job_item_messages;
DROP TABLE IF EXISTS bulk_job_items;
DROP TABLE IF EXISTS bulk_jobs;
DROP TYPE IF EXISTS log_severity;
DROP TYPE IF EXISTS bulk_job_status;

-- +goose StatementEnd
