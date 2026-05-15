CREATE TABLE IF NOT EXISTS logs (
    log_id BIGSERIAL PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'processing',
    upload_date TIMESTAMP NOT NULL DEFAULT NOW(),
    nodes_count INT NOT NULL DEFAULT 0,
    ports_count INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_logs_status ON logs(status);
CREATE INDEX idx_logs_upload_date ON logs(upload_date);