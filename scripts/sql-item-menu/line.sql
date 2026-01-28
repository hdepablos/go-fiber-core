-- *************************************
--	Only line
-- *************************************

DO $$
DECLARE
    v_item_type   TEXT := 'line';
    v_item_name   TEXT := 'linea de config';
    v_to_path     TEXT := NULL;
    v_icon        TEXT := NULL;
    v_parent_id   BIGINT := NULL;
    v_order_index INT := 163;
    v_id          BIGINT;
BEGIN
    INSERT INTO menus (
        item_type,
        item_name,
        to_path,
        icon,
        parent_id,
        order_index,
        is_active,
        created_at,
        updated_at,
        deleted_at
    )
    VALUES (
        v_item_type,
        v_item_name,
        v_to_path,
        v_icon,
        v_parent_id,
        v_order_index,
        true,
        now(),
        now(),
        NULL
    )
    RETURNING id INTO v_id;

    -- Ejemplo de uso posterior del id
    RAISE NOTICE 'ID insertado: %', v_id;
END $$;

select * from menus m  order by m.order_index;

select * from menus m;