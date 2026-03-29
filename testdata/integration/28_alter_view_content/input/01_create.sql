CREATE TABLE employees (
    id integer NOT NULL,
    dept_id integer,
    salary numeric
);

CREATE OR REPLACE VIEW dept_salaries AS
SELECT dept_id, salary FROM employees;
