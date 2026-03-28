-- Add GENERATED AS IDENTITY to id column
ALTER TABLE products ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (START WITH 1 INCREMENT BY 1);

-- Set non-default storage type for description column
ALTER TABLE products ALTER COLUMN description SET STORAGE EXTERNAL;

-- Set compression for name column
ALTER TABLE products ALTER COLUMN name SET COMPRESSION pglz;
