CREATE TABLE orders (
    id INTEGER NOT NULL,
    amount NUMERIC NOT NULL,
    status TEXT NOT NULL,
    CONSTRAINT orders_pkey PRIMARY KEY (id),
    CONSTRAINT chk_orders_amount CHECK (amount > 0),
    CONSTRAINT chk_orders_status CHECK (status IN ('pending', 'done'))
);
