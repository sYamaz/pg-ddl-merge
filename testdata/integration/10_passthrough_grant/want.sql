CREATE TABLE users (
    id INT,
    name TEXT NOT NULL,
    email TEXT
);

CREATE INDEX idx_users_name ON users (name);

GRANT SELECT ON users TO readonly_role;

GRANT INSERT, UPDATE ON users TO app_role;

GRANT SELECT ON users TO reporting_role;
