CREATE TABLE orders (
    id INT,
    user_id INT,
    status TEXT
);

CREATE INDEX idx_orders_user_id ON orders (user_id);

CREATE UNIQUE INDEX idx_orders_status_user ON orders (status, user_id);
