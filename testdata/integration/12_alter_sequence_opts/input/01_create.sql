CREATE SEQUENCE orders_id_seq
    INCREMENT BY 1
    MINVALUE 1
    NO MAXVALUE
    START WITH 1
    CACHE 1;

CREATE TABLE orders (
    id bigint NOT NULL DEFAULT nextval('orders_id_seq'),
    status text
);

CREATE TABLE order_items (
    id integer NOT NULL,
    order_id bigint
) PARTITION BY RANGE (id);

CREATE TABLE order_items_2024
    PARTITION OF order_items
    FOR VALUES FROM (1) TO (100000);
