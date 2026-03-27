CREATE TRIGGER trg_stamp_logged_at
    BEFORE INSERT ON events
    FOR EACH ROW EXECUTE FUNCTION stamp_logged_at();
