CREATE TABLE products (
    id INTEGER PRIMARY KEY,
    code TEXT UNIQUE,
    supplier_id INTEGER REFERENCES suppliers(id)
);
