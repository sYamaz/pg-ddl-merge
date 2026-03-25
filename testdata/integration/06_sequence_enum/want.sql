CREATE SEQUENCE user_id_seq START 1 INCREMENT 1;

CREATE TYPE mood AS ENUM (
    'happy',
    'sad',
    'neutral'
);

CREATE TABLE users (
    id INT DEFAULT nextval('user_id_seq'),
    name TEXT,
    current_mood mood
);
