-- ===gm Up===
CREATE TABLE test_sql_migration
(
    id         SERIAL PRIMARY KEY,
    name       varchar(255) NOT NULL,
    created_at timestamp    NOT NULL default now()
);

-- ===gm Down===
DROP TABLE test_sql_migration;
