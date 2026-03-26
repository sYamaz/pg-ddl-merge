CREATE SEQUENCE orders_id_seq MINVALUE 1 START WITH 1 INCREMENT BY 5 MAXVALUE 9999999999 CACHE 10 NO CYCLE;

CREATE TABLE orders (
    id bigint NOT NULL DEFAULT nextval('orders_id_seq'),
    status text,
    total numeric(12,2)
);

CREATE TABLE order_items (
    id integer NOT NULL,
    order_id bigint
) PARTITION BY RANGE (id);

CREATE TABLE order_items_2024
    PARTITION OF order_items
    FOR VALUES FROM (1) TO (100000);

TRUNCATE orders;
