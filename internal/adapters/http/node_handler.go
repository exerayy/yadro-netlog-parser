package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/exerayy/yadro-netlog-parser/internal/core/topology"
)

// NodeResponse ответ с деталями узла
type NodeResponse struct {
	NodeID                 int64  `json:"node_id"`
	NodeGUID               string `json:"node_guid"`
	NodeDesc               string `json:"node_desc"`
	NodeType               string `json:"node_type"`
	NumPorts               int    `json:"num_ports"`
	ProductName            string `json:"product_name,omitempty"`
	SerialNumber           string `json:"serial_number,omitempty"`
	PartNumber             string `json:"part_number,omitempty"`
	LID                    int    `json:"lid"`
	LogID                  int64  `json:"log_id"`
	Endianness             int    `json:"endianness,omitempty"`
	ReproducibilityDisable int    `json:"reproducibility_disable,omitempty"`
}

// NewGetNodeHandler возвращает детали узла системы из лога
// @Summary Получение информации об узле
// @Description Возвращает детальную информацию об узле
// @Tags logs
// @Accept json
// @Produce json
// @Param node_id path int true "ID узла"
// @Success 200 {object} NodeResponse "Информация об узле"
// @Failure 400 {string} string "Неверный формат node_id"
// @Failure 404 {string} string "Узел не найден"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /api/v1/node/{node_id} [get]
func NewGetNodeHandler(log *slog.Logger, service topology.ITopology) http.HandlerFunc {
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

		resp, err := service.GetNodeDetails(r.Context(), nodeID)
		if err != nil {
			log.Error("failed to get node details", "node_id", nodeID, "error", err)

			if errors.Is(err, context.Canceled) {
				http.Error(w, "request cancelled", httpStatusClientClosedRequest)
				return
			}

			if errors.Is(err, topology.ErrNodeNotFound) {
				http.Error(w, "node not found", http.StatusNotFound)
				return
			}

			http.Error(w, "failed to get node details", http.StatusInternalServerError)
			return
		}

		nodeResp := NodeResponse{
			NodeID:                 resp.NodeID,
			NodeGUID:               resp.NodeGUID,
			NodeDesc:               resp.NodeDesc,
			NodeType:               resp.NodeType,
			NumPorts:               resp.NumPorts,
			ProductName:            resp.ProductName,
			SerialNumber:           resp.SerialNumber,
			PartNumber:             resp.PartNumber,
			LID:                    resp.LID,
			LogID:                  resp.LogID,
			Endianness:             resp.Endianness,
			ReproducibilityDisable: resp.ReproducibilityDisable,
		}

		writeJSONFormatted(w, log, nodeResp)
	}
}
