CREATE TABLE IF NOT EXISTS nodes_info (
      info_id BIGSERIAL PRIMARY KEY,
      log_id BIGINT NOT NULL REFERENCES logs(log_id) ON DELETE CASCADE,
      node_guid VARCHAR(255) NOT NULL,
      serial_number VARCHAR(255) DEFAULT '',
      part_number VARCHAR(255) DEFAULT '',
      product_name VARCHAR(255) DEFAULT '',
      linear_fdb_cap INT DEFAULT 0,
      mcast_fdb_cap INT DEFAULT 0,
      lids_per_port INT DEFAULT 0,
      endianness INT DEFAULT 0,
      reproducibility_disable INT DEFAULT 0,
      UNIQUE(log_id, node_guid)
);

CREATE INDEX idx_nodes_info_log_id ON nodes_info(log_id);
CREATE INDEX idx_nodes_info_node_guid ON nodes_info(node_guid);