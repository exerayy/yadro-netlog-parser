package topology

import (
	"context"
)

type DB interface {
	GetNodesByLogID(ctx context.Context, logID int64) ([]NodeData, error)
	GetPortsByLogID(ctx context.Context, logID int64) ([]PortData, error)
	GetNodeByID(ctx context.Context, nodeID int64) (*NodeDetails, error)
	GetPortsByNodeID(ctx context.Context, nodeID int64) ([]PortData, error)
	GetLogByID(ctx context.Context, logID int64) (*LogInfo, error)
}

type ITopology interface {
	GetTopology(ctx context.Context, logID int64) (*Topology, error)
	GetNodeDetails(ctx context.Context, nodeID int64) (*NodeDetails, error)
	GetNodePorts(ctx context.Context, nodeID int64) (*NodePortsResponse, error)
	GetLogInfo(ctx context.Context, logID int64) (*LogInfo, error)
}
