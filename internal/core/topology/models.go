package topology

import (
	"fmt"
	"time"
)

const (
	nodeTypeHost = iota + 1
	nodeTypeSwitch
)

const (
	portStateNoChange = iota
	portStateDown
	portStateInit
	portStateArmed
	portStateActive
)

type NodeData struct {
	NodeID      int64
	NodeGUID    string
	NodeDesc    string
	NodeType    int
	NumPorts    int
	ProductName string
	LID         int
}

type PortData struct {
	NodeGUID      string
	PortGUID      string
	PortNum       int
	LID           int
	LinkWidthActv int
	LinkSpeedActv int
	PortState     int
}

type Topology struct {
	LogID  int64
	Nodes  []Node
	Edges  []Edge
	Groups []Group
}

type Node struct {
	NodeID      int64
	NodeGUID    string
	NodeDesc    string
	NodeType    string
	NumPorts    int
	ProductName string
	LID         int
}

type Edge struct {
	FromNodeGUID string
	FromPortNum  int
	ToNodeGUID   string
	ToPortNum    int
	LinkWidth    int
	LinkSpeed    int
}

type Group struct {
	GroupName string
	NodeGUIDs []string
	Count     int
}

type NodeDetails struct {
	NodeID                 int64
	NodeGUID               string
	NodeDesc               string
	NodeType               string
	NumPorts               int
	ProductName            string
	SerialNumber           string
	PartNumber             string
	LID                    int
	LogID                  int64
	Endianness             int
	ReproducibilityDisable int
}

type NodePortsResponse struct {
	NodeID   int64
	NodeGUID string
	Ports    []NodePort
}

type NodePort struct {
	PortGUID      string
	PortNum       int
	LID           int
	LinkWidthActv int
	LinkSpeedActv int
	PortState     int
	PortStateDesc string
}

type LogInfo struct {
	LogID      int64
	Filename   string
	Status     string
	UploadDate time.Time
	NodesCount int
	PortsCount int
}

func NodeTypeToString(nodeType int) string {
	switch nodeType {
	case nodeTypeHost:
		return "host"
	case nodeTypeSwitch:
		return "switch"
	default:
		return fmt.Sprintf("Unknown(%d)", nodeType)
	}
}

func portStateToString(state int) string {
	switch state {
	case portStateNoChange:
		return "NoChange"
	case portStateDown:
		return "Down"
	case portStateInit:
		return "Initialize"
	case portStateArmed:
		return "Armed"
	case portStateActive:
		return "Active"
	default:
		return fmt.Sprintf("Unknown(%d)", state)
	}
}
