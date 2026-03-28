ALTER TABLE orders RENAME CONSTRAINT orders_amount_positive TO chk_orders_amount;
ALTER TABLE orders RENAME CONSTRAINT orders_status_check TO chk_orders_status;
