package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/exerayy/yadro-netlog-parser/internal/core/topology"
)

// TopologyResponse структура ответа
type TopologyResponse struct {
	LogID  int64           `json:"log_id"`
	Nodes  []TopologyNode  `json:"nodes"`
	Edges  []TopologyEdge  `json:"edges,omitempty"`
	Groups []TopologyGroup `json:"groups"`
}

// TopologyNode узел топологии
type TopologyNode struct {
	NodeID      int64  `json:"node_id"`
	NodeGUID    string `json:"node_guid"`
	NodeDesc    string `json:"node_desc"`
	NodeType    string `json:"node_type"`
	NumPorts    int    `json:"num_ports"`
	ProductName string `json:"product_name,omitempty"`
	LID         int    `json:"lid,omitempty"`
}

// TopologyEdge связь между узлами
type TopologyEdge struct {
	FromNodeGUID string `json:"from_node_guid"`
	FromPortNum  int    `json:"from_port_num"`
	ToNodeGUID   string `json:"to_node_guid"`
	ToPortNum    int    `json:"to_port_num"`
	LinkWidth    int    `json:"link_width"`
	LinkSpeed    int    `json:"link_speed"`
}

// TopologyGroup группа узлов
type TopologyGroup struct {
	GroupName string   `json:"group_name"`
	NodeGUIDs []string `json:"node_guids"`
	Count     int      `json:"count"`
}

// NewGetTopologyHandler возвращает топологию для указанного лога
// @Summary Получение топологии
// @Description Возвращает список узлов и групп топологии
// @Tags logs
// @Accept json
// @Produce json
// @Param log_id path int true "ID лога"
// @Success 200 {object} TopologyResponse "Топология"
// @Failure 400 {string} string "Неверный формат log_id"
// @Failure 404 {string} string "Топология не найдена"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /api/v1/topology/{log_id} [get]
func NewGetTopologyHandler(log *slog.Logger, service topology.ITopology) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logIDStr := r.PathValue("log_id")

		if logIDStr == "" {
			log.Error("log_id is required")
			http.Error(w, "log_id is required", http.StatusBadRequest)
			return
		}

		logID, err := strconv.ParseInt(logIDStr, 10, 64)
		if err != nil {
			log.Error("invalid log_id", "log_id", logIDStr, "error", err)
			http.Error(w, "invalid log_id format", http.StatusBadRequest)
			return
		}

		resp, err := service.GetTopology(r.Context(), logID)
		if err != nil {
			log.Error("failed to get topology", "log_id", logID, "error", err)

			if errors.Is(err, context.Canceled) {
				http.Error(w, "request cancelled", httpStatusClientClosedRequest)
				return
			}

			if errors.Is(err, topology.ErrNoNodesFound) {
				http.Error(w, "no nodes found for specified log_id", http.StatusNotFound)
				return
			}

			http.Error(w, "failed to build topology", http.StatusInternalServerError)
			return
		}

		topologyResp := TopologyResponse{
			LogID:  resp.LogID,
			Nodes:  make([]TopologyNode, 0, len(resp.Nodes)),
			Edges:  make([]TopologyEdge, 0, len(resp.Edges)),
			Groups: make([]TopologyGroup, 0, len(resp.Groups)),
		}

		for _, n := range resp.Nodes {
			topologyResp.Nodes = append(topologyResp.Nodes, TopologyNode{
				NodeID:      n.NodeID,
				NodeGUID:    n.NodeGUID,
				NodeDesc:    n.NodeDesc,
				NodeType:    n.NodeType,
				NumPorts:    n.NumPorts,
				ProductName: n.ProductName,
				LID:         n.LID,
			})
		}

		for _, e := range resp.Edges {
			topologyResp.Edges = append(topologyResp.Edges, TopologyEdge{
				FromNodeGUID: e.FromNodeGUID,
				FromPortNum:  e.FromPortNum,
				ToNodeGUID:   e.ToNodeGUID,
				ToPortNum:    e.ToPortNum,
				LinkWidth:    e.LinkWidth,
				LinkSpeed:    e.LinkSpeed,
			})
		}

		for _, g := range resp.Groups {
			topologyResp.Groups = append(topologyResp.Groups, TopologyGroup{
				GroupName: g.GroupName,
				NodeGUIDs: g.NodeGUIDs,
				Count:     g.Count,
			})
		}

		writeJSONFormatted(w, log, topologyResp)
	}
}
