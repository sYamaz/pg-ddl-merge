CREATE TABLE orders (
    id int,
    user_id int,
    status text
);

CREATE INDEX CONCURRENTLY idx_user ON orders (user_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_status ON orders (status);
