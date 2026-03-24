CREATE TABLE fuga (
    id INT PRIMARY KEY,
    hoge_id INT REFERENCES hoge(id)
);
