CREATE TABLE products (
    id INTEGER NOT NULL,
    name TEXT COMPRESSION pglz,
    price NUMERIC NOT NULL,
    description TEXT STORAGE EXTERNAL
);
