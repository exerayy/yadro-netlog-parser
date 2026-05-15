CREATE TABLE IF NOT EXISTS ports (
    port_id BIGSERIAL PRIMARY KEY,
    log_id BIGINT NOT NULL REFERENCES logs(log_id) ON DELETE CASCADE,
    node_guid VARCHAR(255) NOT NULL,
    port_guid VARCHAR(255) NOT NULL,
    port_num INT NOT NULL,
    lid INT NOT NULL DEFAULT 0,
    link_width_actv INT NOT NULL DEFAULT 0,
    link_speed_actv INT NOT NULL DEFAULT 0,
    port_state INT NOT NULL DEFAULT 0,
    UNIQUE(log_id, node_guid, port_num)
);

CREATE INDEX idx_ports_log_id ON ports(log_id);
CREATE INDEX idx_ports_node_guid ON ports(node_guid);
CREATE INDEX idx_ports_port_state ON ports(port_state);