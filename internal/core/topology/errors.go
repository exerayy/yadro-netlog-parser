package topology

import "errors"

var (
	ErrLogNotFound  = errors.New("log not found")
	ErrNoNodesFound = errors.New("no nodes found")
	ErrNoPortsFound = errors.New("no ports found")
	ErrNodeNotFound = errors.New("node not found")
)
