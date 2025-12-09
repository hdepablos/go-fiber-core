-- *************************************************
-- Cerrar conexiones activas del usuario (necesario antes de borrarlo)
-- *************************************************

SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE usename = 'gorm_reader';


-- *************************************************
-- Revocar privilegios en todos los esquemas
-- *************************************************
-- ==========================================================
-- 🔹 Limpieza completa y eliminación del rol gorm_reader
-- ==========================================================

-- Paso 1: Revocar privilegios en todas las bases donde pueda tener acceso
REVOKE ALL PRIVILEGES ON DATABASE go_bank_file_gen FROM gorm_reader;
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM gorm_reader;
REVOKE ALL PRIVILEGES ON DATABASE template1 FROM gorm_reader;

-- Paso 2: Revocar privilegios en esquema public (base actual)
REVOKE ALL PRIVILEGES ON SCHEMA public FROM gorm_reader;
REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM gorm_reader;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM gorm_reader;

-- Paso 3: Revocar los privilegios por defecto heredados de 'postgres'
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public REVOKE ALL ON TABLES FROM gorm_reader;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public REVOKE ALL ON SEQUENCES FROM gorm_reader;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public REVOKE EXECUTE ON FUNCTIONS FROM gorm_reader;

-- ==========================================================
-- Paso 4: Revocar privilegios sobre funciones del esquema public
-- ==========================================================

DO
$$
DECLARE
    func RECORD;
BEGIN
    FOR func IN
        SELECT n.nspname AS schema_name,
                p.proname AS function_name,
                pg_catalog.pg_get_function_identity_arguments(p.oid) AS args
        FROM pg_proc p
        JOIN pg_namespace n ON n.oid = p.pronamespace
        WHERE has_function_privilege('gorm_reader', p.oid, 'EXECUTE')
            AND n.nspname = 'public'
    LOOP
        RAISE NOTICE 'Revocando privilegios sobre función: %.%(%).', func.schema_name, func.function_name, func.args;
        EXECUTE format('REVOKE ALL ON FUNCTION %I.%I(%s) FROM gorm_reader;', func.schema_name, func.function_name, func.args);
    END LOOP;
END
$$;

-- ==========================================================
-- Paso 6: Finalmente, eliminar el rol
-- ==========================================================

DROP ROLE IF EXISTS gorm_reader;


