CREATE TABLE users (
    id INT,
    name TEXT,
    email TEXT
);

CREATE OR REPLACE VIEW user_names AS SELECT id, name, email FROM users;
