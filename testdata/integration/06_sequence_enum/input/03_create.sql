CREATE TABLE users (
    id INT DEFAULT nextval('user_id_seq'),
    name TEXT,
    current_mood mood
);
