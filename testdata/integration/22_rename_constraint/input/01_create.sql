CREATE TABLE orders (
    id INTEGER NOT NULL,
    amount NUMERIC NOT NULL,
    status TEXT NOT NULL,
    CONSTRAINT orders_pkey PRIMARY KEY (id),
    CONSTRAINT orders_amount_positive CHECK (amount > 0),
    CONSTRAINT orders_status_check CHECK (status IN ('pending', 'done'))
);
