CREATE TABLE events (id SERIAL, payload TEXT, logged_at TIMESTAMPTZ DEFAULT now());

CREATE FUNCTION stamp_logged_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    NEW.logged_at = now();
    RETURN NEW;
END;
$$;
