ALTER TABLE customers ADD COLUMN created_at TIMESTAMP;
ALTER TABLE customers ALTER COLUMN email TYPE VARCHAR(255);
ALTER TABLE orders ADD COLUMN notes TEXT;
ALTER TABLE orders DROP COLUMN amount;
ALTER TABLE orders ADD COLUMN total NUMERIC NOT NULL DEFAULT 0;
CREATE INDEX idx_orders_customer ON orders (customer_id);
CREATE UNIQUE INDEX idx_customers_email ON customers (email);
