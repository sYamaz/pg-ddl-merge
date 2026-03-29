CREATE TABLE categories (
    id integer NOT NULL,
    parent_id integer,
    name text NOT NULL,
    CONSTRAINT categories_pk PRIMARY KEY (id),
    CONSTRAINT categories_parent_fk FOREIGN KEY (parent_id) REFERENCES categories(id)
);
