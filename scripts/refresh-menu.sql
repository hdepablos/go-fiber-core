truncate table menus cascade;

INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(1, 'link', 'Dashboard', '/', 'mdi-view-dashboard', NULL, 10, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(2, 'separator', 'Gestión', NULL, NULL, NULL, 20, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(3, 'link', 'Mensajes', '/messages', 'mdi-chat-processing-outline', NULL, 30, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(4, 'link', 'Segmentos', '/segments', 'mdi-account-group', NULL, 40, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(5, 'line', 'Línea Configuración', NULL, NULL, NULL, 50, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(6, 'group', 'Tablas', NULL, 'mdi-table-large', NULL, 60, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(7, 'link', 'Bancos V1', '/banks-v1', 'mdi-alpha-v-box-outline', 6, 70, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(8, 'link', 'Bancos V2', '/banks-v2', 'mdi-alpha-v-box-outline', 6, 80, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(9, 'link', 'Bancos V3', '/banks-v3', 'mdi-alpha-v-box-outline', 6, 90, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(10, 'link', 'Bancos V4', '/banks-v4', 'mdi-alpha-v-box-outline', 6, 100, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(12, 'line', 'Línea Configuración', NULL, NULL, NULL, 120, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(13, 'separator', 'Archivos', NULL, NULL, NULL, 130, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(14, 'link', 'Mis Archivos', '/brand-category-file-all', 'mdi-file-document-multiple', NULL, 140, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(15, 'link', 'Notificaciones', '/messages', 'mdi-chat-outline', NULL, 150, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(16, 'link', 'Bancos V6', '/banks-v6', 'mdi-alpha-v-box-outline', 6, 115, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(24, 'line', 'Línea de Configuración', NULL, NULL, NULL, 200, true, '2026-01-12 12:30:58.424', '2026-01-12 12:30:58.424', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(17, 'link', 'Notificaciones', '/notificaciones', 'mdi-notification-clear-all', NULL, 160, true, '2026-01-07 12:19:22.814', '2026-01-07 12:19:22.814', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(23, 'line', 'Linea de config', NULL, NULL, NULL, 163, true, '2026-01-07 12:32:51.460', '2026-01-07 12:32:51.460', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(22, 'separator', 'Gestion finanzas', NULL, NULL, NULL, 165, true, '2026-01-07 12:30:33.542', '2026-01-07 12:30:33.542', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(18, 'group', 'Finanzas', NULL, 'mdi-finance', NULL, 170, true, '2026-01-07 12:23:13.204', '2026-01-07 12:23:13.204', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(19, 'link', 'Contabilidad', '/contabilidad', 'mdi-calculator-variant', 18, 180, true, '2026-01-07 12:25:32.250', '2026-01-07 12:25:32.250', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(21, 'link', 'Contabilidades', '/contabilidades', 'mdi-calculator-variant', 18, 190, true, '2026-01-07 12:27:22.786', '2026-01-07 12:27:22.786', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(11, 'link', 'Bancos V5', '/banks-v5', 'mdi-alpha-v-box-outline', 6, 110, true, '2025-12-09 21:10:48.716', '2025-12-09 21:10:48.716', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(26, 'separator', 'Configuraciones', NULL, NULL, NULL, 205, true, '2026-01-12 12:37:20.560', '2026-01-12 12:37:20.560', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(28, 'link', 'AllUsuarios', '/users', 'mdi-account-plus-outline', 27, 220, true, '2026-01-23 17:14:10.489', '2026-01-23 17:14:10.489', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(27, 'group', 'Usuarios', NULL, 'mdi-card-account-details-outline', NULL, 210, true, '2026-01-22 11:35:51.510', '2026-01-22 11:35:51.510', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(25, 'link', 'Menu y Usuarios', '/menu-users', 'mdi-account-group', 27, 240, true, '2026-01-12 12:34:28.047', '2026-01-12 12:34:28.047', NULL);
INSERT INTO public.menus
(id, item_type, item_name, to_path, icon, parent_id, order_index, is_active, created_at, updated_at, deleted_at)
VALUES(29, 'link', 'Menus y Roles', '/menu-roles', 'mdi-account-cog', 27, 230, true, '2026-01-30 15:11:03.203', '2026-01-30 15:11:03.203', NULL);

INSERT INTO public.menu_user
(id, menu_id, user_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(2, 2, 1, true, '2025-12-09 18:10:48.935', '2025-12-09 18:10:48.935', NULL, NULL);

INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(1, 1, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(2, 2, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(3, 3, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(4, 4, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(5, 5, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(6, 6, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(7, 7, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(8, 8, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(9, 9, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(10, 10, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(11, 12, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(12, 13, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(13, 14, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(14, 15, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(15, 16, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(16, 24, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(17, 25, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(18, 17, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(19, 23, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(20, 22, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(21, 18, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(22, 19, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(23, 21, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(24, 26, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(25, 27, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(26, 11, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(27, 28, 1, true, '2026-01-28 19:17:55.897', '2026-01-28 19:17:55.897', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(28, 1, 4, true, '2026-01-28 19:30:13.439', '2026-01-28 19:30:13.439', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(29, 3, 4, true, '2026-01-28 19:30:13.439', '2026-01-28 19:30:13.439', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(30, 14, 4, true, '2026-01-28 19:30:13.439', '2026-01-28 19:30:13.439', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(31, 15, 4, true, '2026-01-28 19:30:13.439', '2026-01-28 19:30:13.439', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(32, 17, 4, true, '2026-01-28 19:30:13.439', '2026-01-28 19:30:13.439', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(33, 28, 4, true, '2026-01-28 19:30:15.707', '2026-01-28 19:30:15.707', NULL, NULL);
INSERT INTO public.menu_role
(id, menu_id, role_id, is_active, created_at, updated_at, deleted_at, operator_id)
VALUES(61, 29, 1, true, '2026-01-30 15:13:18.198', '2026-01-30 15:13:18.198', NULL, NULL);