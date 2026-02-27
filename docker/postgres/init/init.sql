-- Crear roles
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'gorm_writer') THEN
        CREATE ROLE gorm_writer WITH LOGIN PASSWORD 'gorm_write_secret' SUPERUSER CREATEDB;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'gorm_reader') THEN
        CREATE ROLE gorm_reader WITH LOGIN PASSWORD 'gorm_read_secret';
    END IF;
END$$;

-- Conceder permisos en la base de datos actual (la que se crea con POSTGRES_DB)
GRANT ALL PRIVILEGES ON DATABASE go_fiber_core TO gorm_writer;
GRANT ALL PRIVILEGES ON DATABASE go_fiber_core TO gorm_reader;

-- Esquema public
GRANT USAGE ON SCHEMA public TO gorm_writer;
GRANT USAGE ON SCHEMA public TO gorm_reader;

GRANT CREATE ON SCHEMA public TO gorm_writer;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO gorm_writer;
GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO gorm_writer;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO gorm_writer;

GRANT SELECT ON ALL TABLES IN SCHEMA public TO gorm_reader;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO gorm_reader;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO gorm_reader;

-- Default privileges
ALTER DEFAULT PRIVILEGES FOR ROLE gorm_writer IN SCHEMA public GRANT SELECT ON TABLES TO gorm_reader;
ALTER DEFAULT PRIVILEGES FOR ROLE gorm_writer IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO gorm_reader;
ALTER DEFAULT PRIVILEGES FOR ROLE gorm_writer IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO gorm_reader;
