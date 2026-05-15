package topology

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sort"
)

type Service struct {
	log *slog.Logger
	db  DB
}

func New(log *slog.Logger, db DB) *Service {
	return &Service{
		log: log,
		db:  db,
	}
}

func (s *Service) GetTopology(ctx context.Context, logID int64) (*Topology, error) {
	const op = "topology.GetTopology"

	log := s.log.With(slog.String("op", op), slog.Int64("log_id", logID))

	dbNodes, err := s.db.GetNodesByLogID(ctx, logID)
	if err != nil {
		log.Error("failed to get nodes", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if len(dbNodes) == 0 {
		return nil, fmt.Errorf("%s: %w. log_id=%d", op, ErrNoNodesFound, logID)
	}

	dbPorts, err := s.db.GetPortsByLogID(ctx, logID)
	if err != nil {
		log.Error("failed to get ports", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var mgmtPorts []PortData
	var dataPorts []PortData

	for _, p := range dbPorts {
		if p.PortNum == portStateNoChange {
			mgmtPorts = append(mgmtPorts, p)
		} else if p.PortState == portStateActive {
			dataPorts = append(dataPorts, p)
		}
	}

	nodes := s.buildNodes(dbNodes)
	edges := s.buildEdges(dataPorts, mgmtPorts, dbNodes)
	groups := s.buildGroups(dbNodes)

	log.Info("topology built successfully",
		slog.Int("nodes_count", len(nodes)),
		slog.Int("edges_count", len(edges)),
		slog.Int("groups_count", len(groups)))

	return &Topology{
		LogID:  logID,
		Nodes:  nodes,
		Edges:  edges,
		Groups: groups,
	}, nil
}

// buildNodes преобразует данные из БД в узлы топологии
func (s *Service) buildNodes(dbNodes []NodeData) []Node {
	nodes := make([]Node, 0, len(dbNodes))

	for _, n := range dbNodes {
		nodes = append(nodes, Node{
			NodeID:      n.NodeID,
			NodeGUID:    n.NodeGUID,
			NodeDesc:    n.NodeDesc,
			NodeType:    NodeTypeToString(n.NodeType),
			NumPorts:    n.NumPorts,
			ProductName: n.ProductName,
			LID:         n.LID,
		})
	}

	return nodes
}

// buildEdges строит связи между узлами на основе портов
func (s *Service) buildEdges(dataPorts []PortData, mgmtPorts []PortData, dbNodes []NodeData) []Edge {
	lidToNode := make(map[int]string)
	nodeToLID := make(map[string]int)

	for _, p := range mgmtPorts {
		if p.LID > 0 {
			lidToNode[p.LID] = p.NodeGUID
			nodeToLID[p.NodeGUID] = p.LID
		}
	}

	for _, p := range dataPorts {
		if p.LID > 0 {
			if _, exists := nodeToLID[p.NodeGUID]; !exists {
				nodeToLID[p.NodeGUID] = p.LID
				lidToNode[p.LID] = p.NodeGUID
			}
		}
	}

	nodeTypes := make(map[string]int)
	for _, n := range dbNodes {
		nodeTypes[n.NodeGUID] = n.NodeType
	}

	hostLIDs := make(map[string]int)
	switchLIDs := make(map[string]int)

	for guid, lid := range nodeToLID {
		if nodeTypes[guid] == nodeTypeHost {
			hostLIDs[guid] = lid
		}
		if nodeTypes[guid] == nodeTypeSwitch {
			switchLIDs[guid] = lid
		}
	}

	nodePorts := make(map[string][]PortData)
	for _, p := range dataPorts {
		nodePorts[p.NodeGUID] = append(nodePorts[p.NodeGUID], p)
	}

	var edges []Edge
	seen := make(map[string]bool)

	for hostGUID, hostLID := range hostLIDs {
		var nearestSwitch string
		minDiff := -1

		for swGUID, swLID := range switchLIDs {
			diff := abs(hostLID - swLID)
			if minDiff == -1 || diff < minDiff {
				minDiff = diff
				nearestSwitch = swGUID
			}
		}

		if nearestSwitch != "" {
			hostPorts := nodePorts[hostGUID]
			switchPorts := nodePorts[nearestSwitch]

			if len(hostPorts) > 0 && len(switchPorts) > 0 {
				hp := hostPorts[0]
				sp := switchPorts[0]

				key := s.connectionKey(hp, sp)
				if !seen[key] {
					edges = append(edges, Edge{
						FromNodeGUID: hostGUID,
						FromPortNum:  hp.PortNum,
						ToNodeGUID:   nearestSwitch,
						ToPortNum:    sp.PortNum,
						LinkWidth:    min(hp.LinkWidthActv, sp.LinkWidthActv),
						LinkSpeed:    min(hp.LinkSpeedActv, sp.LinkSpeedActv),
					})
					seen[key] = true
				}
			}
		}
	}

	switchGUIDs := make([]string, 0, len(switchLIDs))
	for guid := range switchLIDs {
		switchGUIDs = append(switchGUIDs, guid)
	}

	sort.Strings(switchGUIDs)

	for i := 0; i < len(switchGUIDs); i++ {
		for j := i + 1; j < len(switchGUIDs); j++ {
			sw1 := switchGUIDs[i]
			sw2 := switchGUIDs[j]

			ports1 := nodePorts[sw1]
			ports2 := nodePorts[sw2]

			connected := false
			for _, p1 := range ports1 {
				if connected {
					break
				}
				for _, p2 := range ports2 {
					if p1.LID > 0 && p1.LID == p2.LID {
						key := s.connectionKey(p1, p2)
						if !seen[key] {
							edges = append(edges, Edge{
								FromNodeGUID: sw1,
								FromPortNum:  p1.PortNum,
								ToNodeGUID:   sw2,
								ToPortNum:    p2.PortNum,
								LinkWidth:    min(p1.LinkWidthActv, p2.LinkWidthActv),
								LinkSpeed:    min(p1.LinkSpeedActv, p2.LinkSpeedActv),
							})
							seen[key] = true
						}
						connected = true
						break
					}
				}
			}

			if !connected && len(ports1) > 0 && len(ports2) > 0 {
				lid1 := switchLIDs[sw1]
				lid2 := switchLIDs[sw2]

				if lid1 > 0 && lid2 > 0 && abs(lid1-lid2) <= 2 {
					p1 := ports1[0]
					p2 := ports2[0]

					key := s.connectionKey(p1, p2)
					if !seen[key] {
						edges = append(edges, Edge{
							FromNodeGUID: sw1,
							FromPortNum:  p1.PortNum,
							ToNodeGUID:   sw2,
							ToPortNum:    p2.PortNum,
							LinkWidth:    min(p1.LinkWidthActv, p2.LinkWidthActv),
							LinkSpeed:    min(p1.LinkSpeedActv, p2.LinkSpeedActv),
						})
						seen[key] = true
					}
				}
			}
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].FromNodeGUID != edges[j].FromNodeGUID {
			return edges[i].FromNodeGUID < edges[j].FromNodeGUID
		}
		return edges[i].FromPortNum < edges[j].FromPortNum
	})

	return edges
}

// connectionKey создает уникальный ключ для связи
func (s *Service) connectionKey(p1, p2 PortData) string {
	if p1.NodeGUID < p2.NodeGUID {
		return fmt.Sprintf("%s:%d-%s:%d", p1.NodeGUID, p1.PortNum, p2.NodeGUID, p2.PortNum)
	}
	return fmt.Sprintf("%s:%d-%s:%d", p2.NodeGUID, p2.PortNum, p1.NodeGUID, p1.PortNum)
}

// buildGroups создает группы узлов
func (s *Service) buildGroups(dbNodes []NodeData) []Group {
	groups := []Group{{
		GroupName: "All Nodes",
		NodeGUIDs: make([]string, 0, len(dbNodes)),
		Count:     len(dbNodes),
	}}

	switchGroup := Group{
		GroupName: "Switches",
		NodeGUIDs: make([]string, 0),
	}
	hostGroup := Group{
		GroupName: "Hosts",
		NodeGUIDs: make([]string, 0),
	}
	productGroups := make(map[string]*Group)

	for _, n := range dbNodes {
		groups[0].NodeGUIDs = append(groups[0].NodeGUIDs, n.NodeGUID)

		if n.NodeType == nodeTypeSwitch {
			switchGroup.NodeGUIDs = append(switchGroup.NodeGUIDs, n.NodeGUID)
		} else {
			hostGroup.NodeGUIDs = append(hostGroup.NodeGUIDs, n.NodeGUID)
		}

		if n.ProductName != "" {
			if g, ok := productGroups[n.ProductName]; ok {
				g.NodeGUIDs = append(g.NodeGUIDs, n.NodeGUID)
				g.Count++
			} else {
				productGroups[n.ProductName] = &Group{
					GroupName: fmt.Sprintf("Product: %s", n.ProductName),
					NodeGUIDs: []string{n.NodeGUID},
					Count:     1,
				}
			}
		}
	}

	switchGroup.Count = len(switchGroup.NodeGUIDs)
	hostGroup.Count = len(hostGroup.NodeGUIDs)

	groups = append(groups, switchGroup, hostGroup)

	for _, g := range productGroups {
		if g.Count > 0 {
			groups = append(groups, *g)
		}
	}

	return groups
}

// GetNodeDetails возвращает детальную информацию об узле
func (s *Service) GetNodeDetails(ctx context.Context, nodeID int64) (*NodeDetails, error) {
	const op = "topology.GetNodeDetails"

	log := s.log.With(slog.String("op", op), slog.Int64("node_id", nodeID))

	node, err := s.db.GetNodeByID(ctx, nodeID)
	if err != nil {
		log.Error("failed to get node", slog.String("error", err.Error()))
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, ErrNodeNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return node, nil
}

// GetNodePorts возвращает порты указанного узла
func (s *Service) GetNodePorts(ctx context.Context, nodeID int64) (*NodePortsResponse, error) {
	const op = "topology.GetNodePorts"

	log := s.log.With(slog.String("op", op), slog.Int64("node_id", nodeID))

	ports, err := s.db.GetPortsByNodeID(ctx, nodeID)
	if err != nil {
		log.Error("failed to get ports", slog.String("error", err.Error()))
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, ErrNodeNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if len(ports) == 0 {
		return nil, fmt.Errorf("%s: %w for node_id=%d", op, ErrNoPortsFound, nodeID)
	}

	nodePorts := make([]NodePort, 0, len(ports))
	for _, p := range ports {
		nodePorts = append(nodePorts, NodePort{
			PortGUID:      p.PortGUID,
			PortNum:       p.PortNum,
			LID:           p.LID,
			LinkWidthActv: p.LinkWidthActv,
			LinkSpeedActv: p.LinkSpeedActv,
			PortState:     p.PortState,
			PortStateDesc: portStateToString(p.PortState),
		})
	}

	return &NodePortsResponse{
		NodeID:   nodeID,
		NodeGUID: ports[0].NodeGUID,
		Ports:    nodePorts,
	}, nil
}

// GetLogInfo возвращает мета-информацию о логе
func (s *Service) GetLogInfo(ctx context.Context, logID int64) (*LogInfo, error) {
	const op = "topology.GetLogInfo"

	log := s.log.With(slog.String("op", op), slog.Int64("log_id", logID))

	logInfo, err := s.db.GetLogByID(ctx, logID)
	if err != nil {
		log.Error("failed to get log info", slog.String("error", err.Error()))
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, ErrLogNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return logInfo, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
