package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/exerayy/yadro-netlog-parser/internal/core/topology"
)

// LogResponse ответ с мета-информацией о логе
type LogResponse struct {
	LogID      int64  `json:"log_id"`
	Filename   string `json:"filename"`
	Status     string `json:"status"`
	UploadDate string `json:"upload_date"`
	NodesCount int    `json:"nodes_count"`
	PortsCount int    `json:"ports_count"`
}

// NewGetLogHandler возвращает мета-информацию о логе
// @Summary Получение информации о логе
// @Description Возвращает мета-информацию о логе (status, количество узлов/портов, дата загрузки)
// @Tags logs
// @Accept json
// @Produce json
// @Param log_id path int true "ID лога"
// @Success 200 {object} LogResponse "Мета-информация о логе"
// @Failure 400 {string} string "Неверный формат log_id"
// @Failure 404 {string} string "Лог не найден"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /api/v1/log/{log_id} [get]
func NewGetLogHandler(log *slog.Logger, service topology.ITopology) http.HandlerFunc {
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

		resp, err := service.GetLogInfo(r.Context(), logID)
		if err != nil {
			log.Error("failed to get log info", "log_id", logID, "error", err)

			if errors.Is(err, context.Canceled) {
				http.Error(w, "request cancelled", httpStatusClientClosedRequest)
				return
			}

			if errors.Is(err, topology.ErrLogNotFound) {
				http.Error(w, "log not found", http.StatusNotFound)
				return
			}

			http.Error(w, "failed to get log info", http.StatusInternalServerError)
			return
		}

		logResp := LogResponse{
			LogID:      resp.LogID,
			Filename:   resp.Filename,
			Status:     resp.Status,
			UploadDate: resp.UploadDate.Format(time.RFC3339),
			NodesCount: resp.NodesCount,
			PortsCount: resp.PortsCount,
		}

		writeJSONFormatted(w, log, logResp)
	}
}
