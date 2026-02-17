-- +goose Up
-- +goose StatementBegin

CREATE TABLE catalog_items (
    id              BIGSERIAL PRIMARY KEY,

    name            VARCHAR(150) NOT NULL,
    code            BIGINT NOT NULL,

    parent_id       BIGINT NULL,

    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,

    sort_order      INTEGER NOT NULL DEFAULT 0,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,

    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP NULL,

    CONSTRAINT fk_catalog_parent
        FOREIGN KEY (parent_id)
        REFERENCES catalog_items(id)
        ON DELETE RESTRICT,

    CONSTRAINT chk_no_self_parent
        CHECK (parent_id IS NULL OR parent_id <> id)
);

-- =========================================================
-- UNIQUE: code único por nivel (soft delete aware)
-- =========================================================
CREATE UNIQUE INDEX uq_catalog_code_per_parent
ON catalog_items (code, parent_id)
WHERE deleted_at IS NULL;

-- =========================================================
-- ÍNDICES DE PERFORMANCE
-- =========================================================

-- Árbol jerárquico
CREATE INDEX idx_catalog_parent_id
ON catalog_items(parent_id)
WHERE deleted_at IS NULL;

-- Búsqueda por code
CREATE INDEX idx_catalog_code
ON catalog_items(code)
WHERE deleted_at IS NULL;

-- JSONB indexado (solo si realmente usarás filtros sobre metadata)
CREATE INDEX idx_catalog_metadata_gin
ON catalog_items
USING GIN (metadata);

-- +goose StatementEnd



-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_catalog_metadata_gin;
DROP INDEX IF EXISTS idx_catalog_code;
DROP INDEX IF EXISTS idx_catalog_parent_id;
DROP INDEX IF EXISTS uq_catalog_code_per_parent;

DROP TABLE IF EXISTS catalog_items;

-- +goose StatementEnd
