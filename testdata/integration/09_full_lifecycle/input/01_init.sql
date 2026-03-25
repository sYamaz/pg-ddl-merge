CREATE SEQUENCE order_id_seq;

CREATE TABLE customers (
    id INT,
    email TEXT NOT NULL,
    name TEXT
);

CREATE TABLE orders (
    id INT DEFAULT nextval('order_id_seq'),
    customer_id INT,
    amount NUMERIC
);
