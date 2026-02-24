-- ============================================================
-- SCRIPT DE CERTIFICACIÓN: LÓGICA DE CICLO DE VIDA DE PROCESOS
-- ============================================================
-- Instrucciones: Ejecuta este script en tu base de datos local
-- para verificar el comportamiento de resolución y herencia.
-- ============================================================

BEGIN; -- Iniciamos transacción para poder hacer rollback al final y no ensuciar

-- 1. Limpieza preliminar (Solo para el test)
TRUNCATE TABLE process_steps CASCADE;
TRUNCATE TABLE process_version_history CASCADE;
TRUNCATE TABLE process_versions CASCADE;
TRUNCATE TABLE process_types CASCADE;

-- 2. Setup: Crear Tipo de Proceso y Usuario (Operator)
INSERT INTO process_types (id, name, is_visible) VALUES (999, 'Test Certification Process', true);
-- Asumimos que existe un usuario con ID 1, si no, crear uno dummy o usar uno existente.

RAISE NOTICE '--- INICIO DE CERTIFICACIÓN ---';

-- ============================================================
-- CASO 1: SOLO VERSIÓN GLOBAL
-- ============================================================
RAISE NOTICE 'CASO 1: Creando Versión Global (ID 100)...';

INSERT INTO process_versions (id, process_type_id, version_number, sede_id, status, operator_id)
VALUES (100, 999, 1, NULL, 'PROD', 1);

INSERT INTO process_steps (process_version_id, step_order, name, execution_key, roadmap)
VALUES (100, 1, 'Paso Global', 'global_step', 0);

-- TEST 1.1: Sede 8 (Cualquiera) -> Debe resolver Global
RAISE NOTICE 'TEST 1.1: Resolviendo para Sede 8...';
DO $$
DECLARE v_id BIGINT;
BEGIN
    SELECT process_version_id INTO v_id FROM resolve_process_version(999, 8, NULL, 0);
    IF v_id = 100 THEN
        RAISE NOTICE '✅ ÉXITO: Sede 8 resolvió a Global (ID 100)';
    ELSE
        RAISE EXCEPTION '❌ FALLO: Sede 8 debió resolver a 100, obtuvo %', v_id;
    END IF;
END $$;

-- TEST 1.2: Sede 5 -> Debe resolver Global (aún no tiene específica)
RAISE NOTICE 'TEST 1.2: Resolviendo para Sede 5...';
DO $$
DECLARE v_id BIGINT;
BEGIN
    SELECT process_version_id INTO v_id FROM resolve_process_version(999, 5, NULL, 0);
    IF v_id = 100 THEN
        RAISE NOTICE '✅ ÉXITO: Sede 5 resolvió a Global (ID 100)';
    ELSE
        RAISE EXCEPTION '❌ FALLO: Sede 5 debió resolver a 100, obtuvo %', v_id;
    END IF;
END $$;


-- ============================================================
-- CASO 2: VERSIÓN ESPECÍFICA (SEDE 5)
-- ============================================================
RAISE NOTICE 'CASO 2: Creando Versión Específica Sede 5 (ID 200)...';

INSERT INTO process_versions (id, process_type_id, version_number, sede_id, status, operator_id)
VALUES (200, 999, 2, 5, 'PROD', 1);

INSERT INTO process_steps (process_version_id, step_order, name, execution_key, roadmap)
VALUES (200, 1, 'Paso Sede 5', 'sede5_step', 0);

-- TEST 2.1: Sede 8 -> Debe SEGUIR resolviendo Global
RAISE NOTICE 'TEST 2.1: Resolviendo para Sede 8...';
DO $$
DECLARE v_id BIGINT;
BEGIN
    SELECT process_version_id INTO v_id FROM resolve_process_version(999, 8, NULL, 0);
    IF v_id = 100 THEN
        RAISE NOTICE '✅ ÉXITO: Sede 8 sigue resolviendo a Global (ID 100)';
    ELSE
        RAISE EXCEPTION '❌ FALLO: Sede 8 debió resolver a 100, obtuvo %', v_id;
    END IF;
END $$;

-- TEST 2.2: Sede 5 -> Debe resolver Específica
RAISE NOTICE 'TEST 2.2: Resolviendo para Sede 5...';
DO $$
DECLARE v_id BIGINT;
BEGIN
    SELECT process_version_id INTO v_id FROM resolve_process_version(999, 5, NULL, 0);
    IF v_id = 200 THEN
        RAISE NOTICE '✅ ÉXITO: Sede 5 ahora resuelve a Específica (ID 200)';
    ELSE
        RAISE EXCEPTION '❌ FALLO: Sede 5 debió resolver a 200, obtuvo %', v_id;
    END IF;
END $$;


-- ============================================================
-- CASO 3: PROMOCIÓN Y HISTORIAL (SEDE 5)
-- ============================================================
RAISE NOTICE 'CASO 3: Actualizando Sede 5 (Promoción ID 200 -> ID 300)...';

-- Simular nueva versión en TEST
INSERT INTO process_versions (id, process_type_id, version_number, sede_id, status, operator_id)
VALUES (300, 999, 3, 5, 'TEST', 1); -- Empieza en TEST

-- Ejecutar Promoción usando la función almacenada
SELECT promote_process_version(300, 1, 'Upgrade Sede 5');

-- TEST 3.1: Verificar estado de ID 200 (Debe ser HISTORY)
RAISE NOTICE 'TEST 3.1: Verificando estado anterior (ID 200)...';
DO $$
DECLARE v_status text;
BEGIN
    SELECT status::text INTO v_status FROM process_versions WHERE id = 200;
    IF v_status = 'HISTORY' THEN
        RAISE NOTICE '✅ ÉXITO: Versión anterior (200) pasó a HISTORY';
    ELSE
        RAISE EXCEPTION '❌ FALLO: Versión 200 debió pasar a HISTORY, está en %', v_status;
    END IF;
END $$;

-- TEST 3.2: Verificar estado de ID 300 (Debe ser PROD)
RAISE NOTICE 'TEST 3.2: Verificando estado nuevo (ID 300)...';
DO $$
DECLARE v_status text;
BEGIN
    SELECT status::text INTO v_status FROM process_versions WHERE id = 300;
    IF v_status = 'PROD' THEN
        RAISE NOTICE '✅ ÉXITO: Nueva versión (300) pasó a PROD';
    ELSE
        RAISE EXCEPTION '❌ FALLO: Versión 300 debió pasar a PROD, está en %', v_status;
    END IF;
END $$;

-- TEST 3.3: Verificar Historial
RAISE NOTICE 'TEST 3.3: Verificando registro en historial...';
DO $$
DECLARE v_count int;
BEGIN
    SELECT COUNT(*) INTO v_count 
    FROM process_version_history 
    WHERE process_version_id = 300 AND process_type_id = 999;
    
    IF v_count = 1 THEN
        RAISE NOTICE '✅ ÉXITO: Se creó registro en process_version_history para ID 300';
    ELSE
        RAISE EXCEPTION '❌ FALLO: No se encontró historial para ID 300';
    END IF;
END $$;

RAISE NOTICE '--- CERTIFICACIÓN COMPLETADA EXITOSAMENTE ---';

ROLLBACK; -- Deshacer todo para no dejar basura en la DB
