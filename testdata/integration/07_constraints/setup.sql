-- Prerequisite table referenced by FK constraints in input files.
-- This file is applied before both sequential and merged runs, but is
-- intentionally excluded from the merger input directory.
CREATE TABLE users (
    id INT PRIMARY KEY
);
