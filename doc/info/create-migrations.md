| Acción        | Prefijo        | Ejemplo                               |
|--------------|----------------|----------------------------------------|
| Crear tabla   | create_        | create_users_table                    |
| Agregar campo | add_           | add_is_active_to_users                |
| Renombrar     | rename_        | rename_username_to_user_name_in_users |
| Alterar tipo  | alter_         | alter_users_age_type_to_integer       |
| Eliminar      | drop_          | drop_address_from_users               |
| Proceso       | create_process_| create_process_lifecycle_manager      |

Nota: Para las tablas pivote ambas singular role_user orden alfabético
	ejemplo: role → user


Para crear una migración de tabla:

	make create-migration name=create_users_table

Para crear una migración que hace referencia a un proceso (workflow, lifecycle, etc.):

	make create-migration name=create_process_lifecycle_manager

En estos casos, el nombre sigue la convención:

- Prefijo `create_process_`
- Descripción corta del proceso en snake_case (por ejemplo `lifecycle_manager`)
