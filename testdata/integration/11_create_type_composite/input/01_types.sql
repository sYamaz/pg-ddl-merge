CREATE TYPE address AS (
    street text,
    city   text,
    zip    varchar(10)
);

CREATE TYPE floatrange AS RANGE (
    subtype      = float8,
    subtype_diff = float8mi
);

CREATE TYPE status AS ENUM ('pending', 'active', 'closed');
