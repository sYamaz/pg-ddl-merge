CREATE TABLE orders (
    id INT,
    user_id INT,
    total NUMERIC,
    CONSTRAINT pk_orders PRIMARY KEY (id),
    CONSTRAINT chk_total CHECK (total >= 0)
);
