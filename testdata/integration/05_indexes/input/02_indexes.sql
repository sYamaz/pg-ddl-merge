CREATE INDEX idx_orders_user_id ON orders (user_id);
CREATE UNIQUE INDEX idx_orders_status_user ON orders (status, user_id);
CREATE INDEX idx_to_drop ON orders (id);
