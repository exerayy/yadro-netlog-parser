package http

import (
	"log/slog"
	"net/http"
)

// HealthResponse ответ healthcheck
type HealthResponse struct {
	Status string `json:"status"`
}

// NewHealthHandler возвращает healthcheck
// @Summary Healthcheck
// @Description Проверка работоспособности сервиса
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse "Сервис работает"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /api/v1/health [get]
func NewHealthHandler(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSONFormatted(w, log, HealthResponse{
			Status: "ok",
		})
	}
}
