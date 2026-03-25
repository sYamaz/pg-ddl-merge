CREATE SEQUENCE order_id_seq;

CREATE TABLE customers (
    id INT,
    email VARCHAR(255) NOT NULL,
    name TEXT,
    created_at TIMESTAMP,
    CONSTRAINT pk_customers PRIMARY KEY (id)
);

CREATE TABLE orders (
    id INT DEFAULT nextval('order_id_seq'),
    customer_id INT,
    notes TEXT,
    total NUMERIC NOT NULL DEFAULT 0.00
);

CREATE INDEX idx_orders_customer ON orders (customer_id);

CREATE UNIQUE INDEX idx_customers_email ON customers (email);
