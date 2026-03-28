CREATE TABLE products (
    id INTEGER,
    name TEXT COMPRESSION pglz,
    price NUMERIC NOT NULL,
    description TEXT STORAGE EXTERNAL
);
