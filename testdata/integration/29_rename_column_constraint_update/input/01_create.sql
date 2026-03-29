CREATE TABLE orders (
    id integer NOT NULL,
    customer_id integer NOT NULL,
    amount numeric NOT NULL,
    CONSTRAINT orders_pk PRIMARY KEY (id),
    CONSTRAINT orders_positive_amount CHECK (amount > 0),
    CONSTRAINT orders_positive_customer CHECK (customer_id > 0)
);
