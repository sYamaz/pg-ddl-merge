CREATE TEMPORARY TABLE tmp_session (
    token text,
    user_id int
);

CREATE UNLOGGED TABLE cache_data (
    key text,
    value text
);

CREATE TABLE events (
    id int,
    region text
) PARTITION BY RANGE (region);

CREATE TABLE events_us PARTITION OF events FOR VALUES FROM ('US') TO ('US~');
