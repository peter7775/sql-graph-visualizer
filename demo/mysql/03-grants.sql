-- The application's performance monitoring reads MySQL performance_schema / sys.
-- demo_user already owns ecommerce_demo.* (ALL, which includes CREATE/DROP INDEX
-- used by the demo "Apply suggested index" action); add read access to the monitoring schemas.
GRANT SELECT ON performance_schema.* TO 'demo_user'@'%';
GRANT SELECT ON sys.* TO 'demo_user'@'%';
GRANT PROCESS ON *.* TO 'demo_user'@'%';
FLUSH PRIVILEGES;
