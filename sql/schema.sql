CREATE TABLE IF NOT EXISTS page (
        page_id INT PRIMARY KEY,
        title TEXT,
        lower_title TEXT
);
/*
** Create index to search page by title, cockroachdb doesn't handle computed index
*/
CREATE INDEX IF NOT EXISTS page_title ON page (lower_title);


/*
** page_content contains plain article content
*/
CREATE TABLE IF NOT EXISTS page_content (
        page_id INT PRIMARY KEY REFERENCES page (page_id) ON DELETE CASCADE,
        content TEXT
);

/* set number of replicas to 1 (keeping only 1 copy) instead of default 3 for page_content
** keep storage used low, since page_content is a heavy table but not necessary for extensive
** data query
**
** set GC to 1 hour instead of 25 to reclaim storage space faster
**/
ALTER TABLE page_content CONFIGURE ZONE USING num_replicas = 1, gc.ttlseconds = 3600;

/* article_reference contains references to other articles
*/
CREATE TABLE IF NOT EXISTS article_reference (page_id INT, refered_page INT, occurrence INT, reference_index INT, PRIMARY KEY (page_id, refered_page));

/* incoming_reference index allows querying incoming reference for a given article
*/
CREATE INDEX IF NOT EXISTS incoming_reference ON article_reference (refered_page);
