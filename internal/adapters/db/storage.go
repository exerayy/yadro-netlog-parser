package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/exerayy/yadro-netlog-parser/internal/core/parser"
	"github.com/exerayy/yadro-netlog-parser/internal/core/topology"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

const (
	maxOpenConns    = 25
	maxIdleConns    = 10
	connMaxLifetime = 30 * time.Minute
	connMaxIdleTime = 5 * time.Minute
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {
	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetConnMaxIdleTime(connMaxIdleTime)

	return &DB{
		log:  log,
		conn: db,
	}, nil
}

func (db *DB) Close() {
	err := db.conn.Close()
	if err != nil {
		db.log.Error("failed to close connection", "error", err)
	}
}

func (db *DB) SaveLogData(ctx context.Context, data *parser.LogData) (int64, error) {
	tx, err := db.conn.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		err = tx.Rollback()
		if err != nil {
			db.log.Error("failed to rollback transaction", "error", err)
		}
	}()

	var logID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO logs (filename, status, upload_date, nodes_count, ports_count)
		VALUES ($1, 'processing', $2, $3, $4)
		RETURNING log_id
	`, data.Filename, time.Now(), len(data.Nodes), len(data.Ports)).Scan(&logID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert log: %w", err)
	}

	for _, node := range data.Nodes {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO nodes (log_id, node_desc, num_ports, node_type, node_guid)
			VALUES ($1, $2, $3, $4, $5)
		`, logID, node.NodeDesc, node.NumPorts, node.NodeType, node.NodeGUID)
		if err != nil {
			return 0, fmt.Errorf("failed to insert node %s: %w", node.NodeGUID, err)
		}
	}

	for _, port := range data.Ports {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO ports (log_id, node_guid, port_guid, port_num, lid, 
			                  link_width_actv, link_speed_actv, port_state)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, logID, port.NodeGUID, port.PortGUID, port.PortNum, port.LID,
			port.LinkWidthActv, port.LinkSpeedActv, port.PortState)
		if err != nil {
			return 0, fmt.Errorf("failed to insert port %s/%d: %w", port.NodeGUID, port.PortNum, err)
		}
	}

	for _, info := range data.NodesInfo {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO nodes_info (log_id, node_guid, serial_number, part_number, 
			                       product_name, linear_fdb_cap, mcast_fdb_cap, 
			                       lids_per_port, endianness, reproducibility_disable)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, logID, info.NodeGUID, info.SerialNumber, info.PartNumber,
			info.ProductName, info.LinearFDBCap, info.MCastFDBCap,
			info.LidsPerPort, info.Endianness, info.ReproducibilityDisable)
		if err != nil {
			return 0, fmt.Errorf("failed to insert node_info %s: %w", info.NodeGUID, err)
		}
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE logs SET status = 'completed' WHERE log_id = $1
	`, logID)
	if err != nil {
		return 0, fmt.Errorf("failed to update log status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return logID, nil
}

// GetNodesByLogID получает все узлы для указанного лога
func (db *DB) GetNodesByLogID(ctx context.Context, logID int64) ([]topology.NodeData, error) {
	const op = "DB.GetNodesByLogID"

	query := `
		SELECT n.node_id, n.node_guid, n.node_desc, n.node_type, n.num_ports,
		       COALESCE(ni.product_name, '') as product_name,
		       COALESCE(MIN(p.lid), 0) as lid
		FROM nodes n
		LEFT JOIN nodes_info ni ON n.log_id = ni.log_id AND n.node_guid = ni.node_guid
		LEFT JOIN ports p ON n.log_id = p.log_id AND n.node_guid = p.node_guid AND p.lid > 0
		WHERE n.log_id = $1
		GROUP BY n.node_id, n.node_guid, n.node_desc, n.node_type, n.num_ports, ni.product_name
		ORDER BY n.node_type, n.node_desc
	`

	var nodes []NodeData
	err := db.conn.SelectContext(ctx, &nodes, query, logID)
	if err != nil {
		db.log.Error("failed to get nodes", slog.String("op", op), slog.Int64("log_id", logID), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	resNodes := make([]topology.NodeData, len(nodes))
	for i, node := range nodes {
		resNodes[i] = topology.NodeData{
			NodeID:      node.NodeID,
			NodeGUID:    node.NodeGUID,
			NodeDesc:    node.NodeDesc,
			NodeType:    node.NodeType,
			NumPorts:    node.NumPorts,
			ProductName: node.ProductName,
			LID:         node.LID,
		}
	}

	return resNodes, nil
}

// GetPortsByLogID получает все активные порты для указанного лога
func (db *DB) GetPortsByLogID(ctx context.Context, logID int64) ([]topology.PortData, error) {
	const op = "DB.GetPortsByLogID"

	query := `
		SELECT node_guid, port_guid, port_num, lid, link_width_actv, 
		       link_speed_actv, port_state
		FROM ports
		WHERE log_id = $1
		ORDER BY node_guid, port_num
	`

	var ports []PortData
	err := db.conn.SelectContext(ctx, &ports, query, logID)
	if err != nil {
		db.log.Error("failed to get ports", slog.String("op", op), slog.Int64("log_id", logID), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	resPorts := make([]topology.PortData, len(ports))
	for i, port := range ports {
		resPorts[i] = topology.PortData{
			NodeGUID:      port.NodeGUID,
			PortGUID:      port.PortGUID,
			PortNum:       port.PortNum,
			LID:           port.LID,
			LinkWidthActv: port.LinkWidthActv,
			LinkSpeedActv: port.LinkSpeedActv,
			PortState:     port.PortState,
		}
	}

	return resPorts, nil
}

func (db *DB) GetNodeByID(ctx context.Context, nodeID int64) (*topology.NodeDetails, error) {
	const op = "DB.GetNodeByID"

	query := `
		SELECT n.node_id, n.node_guid, n.node_desc, n.node_type, n.num_ports,
		       COALESCE(ni.product_name, '') as product_name,
		       COALESCE(ni.serial_number, '') as serial_number,
		       COALESCE(ni.part_number, '') as part_number,
		       COALESCE(MIN(p.lid), 0) as lid,
		       n.log_id,
		       COALESCE(ni.endianness, 0) as endianness,
		       COALESCE(ni.reproducibility_disable, 0) as reproducibility_disable
		FROM nodes n
		LEFT JOIN nodes_info ni ON n.log_id = ni.log_id AND n.node_guid = ni.node_guid
		LEFT JOIN ports p ON n.log_id = p.log_id AND n.node_guid = p.node_guid AND p.lid > 0
		WHERE n.node_id = $1
		GROUP BY n.node_id, n.node_guid, n.node_desc, n.node_type, n.num_ports, 
		         ni.product_name, ni.serial_number, ni.part_number, n.log_id,
		         ni.endianness, ni.reproducibility_disable
	`

	var node NodeDetailsData
	err := db.conn.GetContext(ctx, &node, query, nodeID)
	if err != nil {
		db.log.Error("failed to get node", slog.String("op", op), slog.Int64("node_id", nodeID), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	nodeDetails := topology.NodeDetails{
		NodeID:                 nodeID,
		NodeGUID:               node.NodeGUID,
		NodeDesc:               node.NodeDesc,
		NodeType:               topology.NodeTypeToString(node.NodeType),
		NumPorts:               node.NumPorts,
		ProductName:            node.ProductName,
		SerialNumber:           node.SerialNumber,
		PartNumber:             node.PartNumber,
		LID:                    node.LID,
		LogID:                  node.LogID,
		Endianness:             node.Endianness,
		ReproducibilityDisable: node.ReproducibilityDisable,
	}

	return &nodeDetails, nil
}

func (db *DB) GetPortsByNodeID(ctx context.Context, nodeID int64) ([]topology.PortData, error) {
	const op = "DB.GetPortsByNodeID"

	query := `
		SELECT p.node_guid, p.port_guid, p.port_num, p.lid, p.link_width_actv, 
		       p.link_speed_actv, p.port_state
		FROM ports p
		JOIN nodes n ON p.log_id = n.log_id AND p.node_guid = n.node_guid
		WHERE n.node_id = $1
		ORDER BY p.port_num
	`

	var ports []PortData
	err := db.conn.SelectContext(ctx, &ports, query, nodeID)
	if err != nil {
		db.log.Error("failed to get ports", slog.String("op", op), slog.Int64("node_id", nodeID), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	resPorts := make([]topology.PortData, len(ports))
	for i, port := range ports {
		resPorts[i] = topology.PortData{
			NodeGUID:      port.NodeGUID,
			PortGUID:      port.PortGUID,
			PortNum:       port.PortNum,
			LID:           port.LID,
			LinkWidthActv: port.LinkWidthActv,
			LinkSpeedActv: port.LinkSpeedActv,
			PortState:     port.PortState,
		}
	}

	return resPorts, nil
}

func (db *DB) GetLogByID(ctx context.Context, logID int64) (*topology.LogInfo, error) {
	const op = "DB.GetLogByID"

	query := `
		SELECT log_id, filename, status, upload_date, nodes_count, ports_count
		FROM logs
		WHERE log_id = $1
	`

	var logData LogData
	err := db.conn.GetContext(ctx, &logData, query, logID)
	if err != nil {
		db.log.Error("failed to get log", slog.String("op", op), slog.Int64("log_id", logID), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	logInfo := topology.LogInfo{
		LogID:      logID,
		Filename:   logData.Filename,
		Status:     logData.Status,
		UploadDate: logData.UploadDate,
		NodesCount: logData.NodesCount,
		PortsCount: logData.PortsCount,
	}

	return &logInfo, nil
}
