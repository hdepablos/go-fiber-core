
SELECT pid, usename, application_name, client_addr
FROM pg_stat_activity
WHERE datname = 'go_bank_file_gen';

-- Esto desconecta todas las conexiones y luego dropea la base
-- ⚠️ Usar solo si estás seguro
UPDATE pg_database SET datallowconn = FALSE WHERE datname = 'go_bank_file_gen';
SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'go_bank_file_gen';
DROP DATABASE go_bank_file_gen;