-- ============================================================
-- roles_setup.sql
-- Crea roles gorm_writer y gorm_reader, otorga permisos y
-- configura ALTER DEFAULT PRIVILEGES para que gorm_reader
-- tenga acceso automático (solo lectura) a objetos creados
-- por gorm_writer en el esquema "public".
-- Ejecutar como superusuario (ej: postgres).
-- ============================================================

-- ---------- Crear rol escritura (si no existe) ----------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'gorm_writer') THEN
        CREATE ROLE gorm_writer
            WITH LOGIN
                PASSWORD 'gorm_write_secret'
                NOSUPERUSER
                NOCREATEDB
                NOCREATEROLE
                NOINHERIT
                CONNECTION LIMIT -1;
    END IF;
END$$;

-- ---------- Crear rol lectura (si no existe) ----------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'gorm_reader') THEN
        CREATE ROLE gorm_reader
            WITH LOGIN
                PASSWORD 'gorm_read_secret'
                NOSUPERUSER
                NOCREATEDB
                NOCREATEROLE
                NOINHERIT
                CONNECTION LIMIT -1;
    END IF;
END$$;

-- ---------- Conceder CONNECT a la base ----------
GRANT CONNECT ON DATABASE go_bank_file_gen TO gorm_writer;
GRANT CONNECT ON DATABASE go_bank_file_gen TO gorm_reader;

-- ---------- Uso del esquema ----------
GRANT USAGE ON SCHEMA public TO gorm_writer;
GRANT USAGE ON SCHEMA public TO gorm_reader;

-- ---------- Permisos para gorm_writer (creación y manipulación) ----------
GRANT CREATE ON SCHEMA public TO gorm_writer;

-- Permisos sobre objetos existentes (tablas/sequences/functions)
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO gorm_writer;
GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO gorm_writer;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO gorm_writer;

-- ---------- Permisos para gorm_reader (solo lectura) ----------
GRANT SELECT ON ALL TABLES IN SCHEMA public TO gorm_reader;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO gorm_reader;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO gorm_reader;

-- ---------- DEFAULT PRIVILEGES: IMPORTANTÍSIMO ----------
-- Estas líneas hacen que TODO lo que cree gorm_writer en el esquema public
-- (tablas, secuencias, funciones) otorgue automáticamente permisos a gorm_reader.

-- Importante: "FOR ROLE gorm_writer" -> se aplicará a los objetos creados POR gorm_writer.
ALTER DEFAULT PRIVILEGES FOR ROLE gorm_writer IN SCHEMA public
    GRANT SELECT ON TABLES TO gorm_reader;

ALTER DEFAULT PRIVILEGES FOR ROLE gorm_writer IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO gorm_reader;

ALTER DEFAULT PRIVILEGES FOR ROLE gorm_writer IN SCHEMA public
    GRANT EXECUTE ON FUNCTIONS TO gorm_reader;

-- ---------- Nota final ----------
-- Si tus migraciones las ejecuta otro rol distinto a gorm_writer,
-- reemplaza FOR ROLE gorm_writer por el rol que realmente crea las tablas.
