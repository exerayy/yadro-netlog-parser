CREATE TABLE IF NOT EXISTS nodes (
    node_id BIGSERIAL PRIMARY KEY,
    log_id BIGINT NOT NULL REFERENCES logs(log_id) ON DELETE CASCADE,
    node_desc VARCHAR(255) NOT NULL,
    num_ports INT NOT NULL,
    node_type INT NOT NULL,
    node_guid VARCHAR(255) NOT NULL,
    UNIQUE(log_id, node_guid)
);

CREATE INDEX idx_nodes_log_id ON nodes(log_id);
CREATE INDEX idx_nodes_node_guid ON nodes(node_guid);