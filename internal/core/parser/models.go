package parser

type LogData struct {
	Filename  string
	Nodes     []Node
	Ports     []Port
	NodesInfo []NodeInfo
}

type Node struct {
	NodeDesc string
	NumPorts int
	NodeType int
	NodeGUID string
}

type Port struct {
	NodeGUID      string
	PortGUID      string
	PortNum       int
	LID           int
	LinkWidthActv int
	LinkSpeedActv int
	PortState     int
}

type NodeInfo struct {
	NodeGUID     string
	SerialNumber string
	PartNumber   string
	ProductName  string
	// Switch-specific fields (если NodeType = 2)
	LinearFDBCap int
	MCastFDBCap  int
	LidsPerPort  int
	// Sharp info fields
	Endianness             int
	ReproducibilityDisable int
}
