CREATE TABLE sessions (
    id INT,
    token TEXT NOT NULL,
    old_column TEXT
);
CREATE INDEX idx_session_token ON sessions (token);
