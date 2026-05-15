package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/exerayy/yadro-netlog-parser/internal/core/topology"
)

// NodePortsResponse ответ с портами узла
type NodePortsResponse struct {
	NodeID   int64          `json:"node_id"`
	NodeGUID string         `json:"node_guid"`
	Ports    []NodePortData `json:"ports"`
}

// NodePortData данные порта
type NodePortData struct {
	PortGUID      string `json:"port_guid"`
	PortNum       int    `json:"port_num"`
	LID           int    `json:"lid"`
	LinkWidthActv int    `json:"link_width_actv"`
	LinkSpeedActv int    `json:"link_speed_actv"`
	PortState     int    `json:"port_state"`
	PortStateDesc string `json:"port_state_desc"`
}

// NewGetNodePortsHandler возвращает порты узла системы из лога
// @Summary Получение портов узла
// @Description Возвращает список портов указанного узла
// @Tags logs
// @Accept json
// @Produce json
// @Param node_id path int true "ID узла"
// @Success 200 {object} NodePortsResponse "Порты узла"
// @Failure 400 {string} string "Неверный формат node_id"
// @Failure 404 {string} string "Порты не найдены"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /api/v1/port/{node_id} [get]
func NewGetNodePortsHandler(log *slog.Logger, service topology.ITopology) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeIDStr := r.PathValue("node_id")
		if nodeIDStr == "" {
			log.Error("node_id is required")
			http.Error(w, "node_id is required", http.StatusBadRequest)
			return
		}

		nodeID, err := strconv.ParseInt(nodeIDStr, 10, 64)
		if err != nil {
			log.Error("invalid node_id", "node_id", nodeIDStr, "error", err)
			http.Error(w, "invalid node_id format", http.StatusBadRequest)
			return
		}

		resp, err := service.GetNodePorts(r.Context(), nodeID)
		if err != nil {
			log.Error("failed to get node ports", "node_id", nodeID, "error", err)

			if errors.Is(err, context.Canceled) {
				http.Error(w, "request cancelled", httpStatusClientClosedRequest)
				return
			}

			if errors.Is(err, topology.ErrNodeNotFound) {
				http.Error(w, "node not found", http.StatusNotFound)
				return
			}

			if errors.Is(err, topology.ErrNoPortsFound) {
				http.Error(w, "no ports found for specified node_id", http.StatusNotFound)
				return
			}

			http.Error(w, "failed to get node ports", http.StatusInternalServerError)
			return
		}

		portsResp := NodePortsResponse{
			NodeID:   resp.NodeID,
			NodeGUID: resp.NodeGUID,
			Ports:    make([]NodePortData, 0, len(resp.Ports)),
		}

		for _, p := range resp.Ports {
			portsResp.Ports = append(portsResp.Ports, NodePortData{
				PortGUID:      p.PortGUID,
				PortNum:       p.PortNum,
				LID:           p.LID,
				LinkWidthActv: p.LinkWidthActv,
				LinkSpeedActv: p.LinkSpeedActv,
				PortState:     p.PortState,
				PortStateDesc: p.PortStateDesc,
			})
		}

		writeJSONFormatted(w, log, portsResp)
	}
}
