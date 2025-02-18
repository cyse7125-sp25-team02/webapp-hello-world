-- migrations/001_create_health_check_table.sql
CREATE TABLE IF NOT EXISTS health_check (
    check_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    datetime DATETIME NOT NULL
);
