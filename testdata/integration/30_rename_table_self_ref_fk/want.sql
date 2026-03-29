CREATE TABLE product_categories (
    id integer NOT NULL,
    parent_id integer,
    name text NOT NULL,
    CONSTRAINT categories_pk PRIMARY KEY (id),
    CONSTRAINT categories_parent_fk FOREIGN KEY (parent_id) REFERENCES product_categories(id)
);
