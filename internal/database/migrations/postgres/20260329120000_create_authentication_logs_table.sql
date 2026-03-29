-- +goose Up
-- +goose StatementBegin

CREATE TABLE authentication_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT,
    email_snapshot VARCHAR(255),
    event_type VARCHAR(32) NOT NULL,
    failure_reason VARCHAR(64),
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT NOT NULL,
    browser VARCHAR(64) NOT NULL,
    operating_system VARCHAR(64) NOT NULL,
    device_type VARCHAR(16) NOT NULL,
    country VARCHAR(64),
    city VARCHAR(64),
    request_id VARCHAR(64),
    origin VARCHAR(32),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_authentication_logs_user_id ON authentication_logs (user_id);
CREATE INDEX idx_authentication_logs_ip_address ON authentication_logs (ip_address);
CREATE INDEX idx_authentication_logs_created_at ON authentication_logs (created_at);
CREATE INDEX idx_authentication_logs_event_type ON authentication_logs (event_type);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_authentication_logs_event_type;
DROP INDEX IF EXISTS idx_authentication_logs_created_at;
DROP INDEX IF EXISTS idx_authentication_logs_ip_address;
DROP INDEX IF EXISTS idx_authentication_logs_user_id;
DROP TABLE IF EXISTS authentication_logs;

-- +goose StatementEnd
