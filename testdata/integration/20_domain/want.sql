CREATE DOMAIN positive_int AS INTEGER CHECK (VALUE > 0);

CREATE TABLE products (
    id positive_int,
    price positive_int NOT NULL
);
