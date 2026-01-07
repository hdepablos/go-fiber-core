-- *************************************
--	Only link
-- *************************************

DO $$
DECLARE
    v_item_type   TEXT := 'link';
    v_item_name   TEXT := 'contabilidades';
    v_to_path     TEXT := '/contabilidades';
    v_icon        TEXT := 'mdi-calculator-variant';
    v_parent_id   BIGINT := 18;
    v_order_index INT := 190;
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

select * from menus m;

select * from menus m  order by m.order_index;