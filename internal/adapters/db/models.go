package db

import "time"

type NodeData struct {
	NodeID      int64  `db:"node_id"`
	NodeGUID    string `db:"node_guid"`
	NodeDesc    string `db:"node_desc"`
	NodeType    int    `db:"node_type"`
	NumPorts    int    `db:"num_ports"`
	ProductName string `db:"product_name"`
	LID         int    `db:"lid"`
}

type PortData struct {
	NodeGUID      string `db:"node_guid"`
	PortGUID      string `db:"port_guid"`
	PortNum       int    `db:"port_num"`
	LID           int    `db:"lid"`
	LinkWidthActv int    `db:"link_width_actv"`
	LinkSpeedActv int    `db:"link_speed_actv"`
	PortState     int    `db:"port_state"`
}

type NodeDetailsData struct {
	NodeID                 int64  `db:"node_id"`
	NodeGUID               string `db:"node_guid"`
	NodeDesc               string `db:"node_desc"`
	NodeType               int    `db:"node_type"`
	NumPorts               int    `db:"num_ports"`
	ProductName            string `db:"product_name"`
	SerialNumber           string `db:"serial_number"`
	PartNumber             string `db:"part_number"`
	LID                    int    `db:"lid"`
	LogID                  int64  `db:"log_id"`
	Endianness             int    `db:"endianness"`
	ReproducibilityDisable int    `db:"reproducibility_disable"`
}

type LogData struct {
	LogID      int64     `db:"log_id"`
	Filename   string    `db:"filename"`
	Status     string    `db:"status"`
	UploadDate time.Time `db:"upload_date"`
	NodesCount int       `db:"nodes_count"`
	PortsCount int       `db:"ports_count"`
}
