-- Same index name with IF NOT EXISTS: must be skipped (index already exists)
CREATE UNIQUE INDEX IF NOT EXISTS idx_status ON orders (status);
