-- +goose Up
-- +goose StatementBegin

------------------------------------------------------------
-- ENUM STATUS
------------------------------------------------------------

CREATE TYPE process_version_status AS ENUM (
    'DRAFT',
    'TEST',
    'PROD',
    'HISTORY'
);

------------------------------------------------------------
-- PROCESS TYPES
------------------------------------------------------------

CREATE TABLE process_types (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    is_visible BOOLEAN NOT NULL DEFAULT TRUE,
    archived_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

------------------------------------------------------------
-- PROCESS VERSIONS
------------------------------------------------------------

CREATE TABLE process_versions (
    id BIGSERIAL PRIMARY KEY,
    process_type_id BIGINT NOT NULL REFERENCES process_types(id),
    version_number INTEGER NOT NULL,
    sede_id BIGINT NULL,
    status process_version_status NOT NULL DEFAULT 'DRAFT',
    operator_id BIGINT NOT NULL,
    archived_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (id, process_type_id)
);

-- Único PROD por tipo + sede
CREATE UNIQUE INDEX ux_unique_prod_per_type_sede
ON process_versions(process_type_id, sede_id)
WHERE status = 'PROD' AND archived_at IS NULL;

------------------------------------------------------------
-- PROCESS STEPS
------------------------------------------------------------

CREATE TABLE process_steps (
    id BIGSERIAL PRIMARY KEY,
    process_version_id BIGINT NOT NULL REFERENCES process_versions(id) ON DELETE CASCADE,
    step_order INTEGER NOT NULL,
    name VARCHAR(150) NOT NULL,
    execution_key VARCHAR(150) NOT NULL,
    config JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

------------------------------------------------------------
-- PROCESS VERSION HISTORY
------------------------------------------------------------

CREATE TABLE process_version_history (
    id BIGSERIAL PRIMARY KEY,
    process_version_id BIGINT NOT NULL,
    process_type_id BIGINT NOT NULL,
    promoted_from_status process_version_status NOT NULL,
    promoted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    promoted_by BIGINT NOT NULL,
    comment VARCHAR(300) NOT NULL,
    FOREIGN KEY (process_version_id, process_type_id)
        REFERENCES process_versions(id, process_type_id)
);

------------------------------------------------------------
-- FUNCIÓN: PROMOTE VERSION
------------------------------------------------------------

CREATE OR REPLACE FUNCTION promote_process_version(
    p_process_version_id BIGINT,
    p_operator_id BIGINT,
    p_comment VARCHAR
)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    v_process_type_id BIGINT;
    v_sede_id BIGINT;
    v_current_prod_id BIGINT;
    v_step_count INTEGER;
    v_old_status process_version_status;
BEGIN

    IF length(p_comment) > 300 THEN
        RAISE EXCEPTION 'Promotion comment exceeds 300 characters';
    END IF;

    SELECT process_type_id, sede_id, status
    INTO v_process_type_id, v_sede_id, v_old_status
    FROM process_versions
    WHERE id = p_process_version_id
    AND archived_at IS NULL
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Process version not found or archived';
    END IF;

    SELECT COUNT(*) INTO v_step_count
    FROM process_steps
    WHERE process_version_id = p_process_version_id;

    IF v_step_count = 0 THEN
        RAISE EXCEPTION 'Cannot promote version without steps';
    END IF;

    IF v_old_status NOT IN ('TEST', 'HISTORY') THEN
        RAISE EXCEPTION 'Only TEST or HISTORY versions can be promoted to PROD';
    END IF;

    SELECT id INTO v_current_prod_id
    FROM process_versions
    WHERE process_type_id = v_process_type_id
    AND sede_id IS NOT DISTINCT FROM v_sede_id
    AND status = 'PROD'
    AND archived_at IS NULL
    FOR UPDATE;

    IF v_current_prod_id IS NOT NULL THEN
        UPDATE process_versions
        SET status = 'HISTORY',
            updated_at = NOW()
        WHERE id = v_current_prod_id;
    END IF;

    UPDATE process_versions
    SET status = 'PROD',
        updated_at = NOW()
    WHERE id = p_process_version_id;

    INSERT INTO process_version_history(
        process_version_id,
        process_type_id,
        promoted_from_status,
        promoted_by,
        comment
    )
    VALUES (
        p_process_version_id,
        v_process_type_id,
        v_old_status,
        p_operator_id,
        p_comment
    );

END;
$$;

------------------------------------------------------------
-- FUNCIÓN: REPLICATE SCENARIO
------------------------------------------------------------

CREATE OR REPLACE FUNCTION replicate_process_version(
    p_process_version_id BIGINT,
    p_operator_id BIGINT
)
RETURNS BIGINT
LANGUAGE plpgsql
AS $$
DECLARE
    v_new_version_id BIGINT;
    v_process_type_id BIGINT;
    v_sede_id BIGINT;
    v_next_version_number INTEGER;
BEGIN

    SELECT process_type_id, sede_id
    INTO v_process_type_id, v_sede_id
    FROM process_versions
    WHERE id = p_process_version_id
    AND archived_at IS NULL;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Process version not found or archived';
    END IF;

    SELECT COALESCE(MAX(version_number),0) + 1
    INTO v_next_version_number
    FROM process_versions
    WHERE process_type_id = v_process_type_id;

    INSERT INTO process_versions(
        process_type_id,
        version_number,
        sede_id,
        status,
        operator_id
    )
    VALUES (
        v_process_type_id,
        v_next_version_number,
        v_sede_id,
        'DRAFT',
        p_operator_id
    )
    RETURNING id INTO v_new_version_id;

    INSERT INTO process_steps(
        process_version_id,
        step_order,
        name,
        execution_key,
        config
    )
    SELECT
        v_new_version_id,
        step_order,
        name,
        execution_key,
        config
    FROM process_steps
    WHERE process_version_id = p_process_version_id;

    RETURN v_new_version_id;

END;
$$;

------------------------------------------------------------
-- FUNCIÓN: RESOLVE VERSION (con override + sede fallback)
------------------------------------------------------------

CREATE OR REPLACE FUNCTION resolve_process_version(
    p_process_type_id BIGINT,
    p_sede_id BIGINT,
    p_override_process_version_id BIGINT DEFAULT NULL
)
RETURNS TABLE (
    process_version_id BIGINT,
    process_steps JSONB
)
LANGUAGE plpgsql
AS $$
DECLARE
    v_process_version_id BIGINT;
BEGIN

    PERFORM 1
    FROM process_types
    WHERE id = p_process_type_id
    AND archived_at IS NULL;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Process type does not exist or is archived';
    END IF;

    IF p_override_process_version_id IS NOT NULL THEN
        SELECT id
        INTO v_process_version_id
        FROM process_versions
        WHERE id = p_override_process_version_id
        AND process_type_id = p_process_type_id
        AND archived_at IS NULL;

        IF v_process_version_id IS NULL THEN
            RAISE EXCEPTION 'Override version invalid';
        END IF;
    ELSE
        SELECT id INTO v_process_version_id
        FROM process_versions
        WHERE process_type_id = p_process_type_id
        AND status = 'PROD'
        AND sede_id = p_sede_id
        AND archived_at IS NULL
        LIMIT 1;

        IF v_process_version_id IS NULL THEN
            SELECT id INTO v_process_version_id
            FROM process_versions
            WHERE process_type_id = p_process_type_id
            AND status = 'PROD'
            AND sede_id IS NULL
            AND archived_at IS NULL
            LIMIT 1;
        END IF;
    END IF;

    IF v_process_version_id IS NULL THEN
        RAISE EXCEPTION 'No active version found';
    END IF;

    RETURN QUERY
    SELECT
        v_process_version_id AS process_version_id,
        COALESCE(
            (
                SELECT jsonb_agg(
                        jsonb_build_object(
                            'name', ps.name,
                            'execution_key', ps.execution_key,
                            'config', COALESCE(ps.config, '{}'::jsonb),
                            'step_order', ps.step_order
                        )
                        ORDER BY ps.step_order
                    )
                FROM process_steps ps
                WHERE ps.process_version_id = v_process_version_id
            ),
            '[]'::jsonb
        ) AS process_steps;

END;
$$;

CREATE OR REPLACE FUNCTION move_process_version_to_test(
    p_process_version_id BIGINT
)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    v_current_status process_version_status;
BEGIN

    SELECT status
    INTO v_current_status
    FROM process_versions
    WHERE id = p_process_version_id
    AND archived_at IS NULL
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Process version not found or archived';
    END IF;

    IF v_current_status <> 'DRAFT' THEN
        RAISE EXCEPTION 'Only DRAFT versions can be moved to TEST';
    END IF;

    UPDATE process_versions
    SET status = 'TEST',
        updated_at = NOW()
    WHERE id = p_process_version_id;

END;
$$;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP FUNCTION IF EXISTS resolve_process_version(BIGINT, BIGINT, BIGINT);
DROP FUNCTION IF EXISTS replicate_process_version(BIGINT);
DROP FUNCTION IF EXISTS promote_process_version(BIGINT, BIGINT, VARCHAR);
DROP FUNCTION IF EXISTS move_process_version_to_test(BIGINT);

DROP TABLE IF EXISTS process_version_history;
DROP TABLE IF EXISTS process_steps;
DROP TABLE IF EXISTS process_versions;
DROP TABLE IF EXISTS process_types;

DROP TYPE IF EXISTS process_version_status;

-- +goose StatementEnd
