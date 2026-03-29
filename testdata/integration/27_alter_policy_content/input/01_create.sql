CREATE TABLE articles (
    id integer,
    author_id integer
);
CREATE POLICY author_policy ON articles FOR SELECT USING (true);
