CREATE TABLE users (id INT, name TEXT, email TEXT);
CREATE VIEW user_names AS SELECT id, name FROM users;
