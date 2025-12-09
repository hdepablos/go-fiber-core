SELECT rolname FROM pg_roles WHERE rolname IN ('gorm_writer', 'gorm_reader');

SELECT 
    grantee, 
    privilege_type 
FROM 
    information_schema.role_table_grants 
WHERE 
    table_name = 'users' AND grantee IN ('gorm_writer', 'gorm_reader');
