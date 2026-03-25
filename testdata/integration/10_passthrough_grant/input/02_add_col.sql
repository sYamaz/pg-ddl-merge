-- カラム追加
ALTER TABLE users ADD COLUMN email TEXT;

GRANT SELECT ON users TO reporting_role;
