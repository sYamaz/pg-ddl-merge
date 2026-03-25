CREATE TYPE status AS ENUM (
    'pending',
    'active',
    'closed',
    'archived'
);

CREATE TYPE mailing_address AS (
    street text,
    city   text,
    zip    varchar(10)
);
