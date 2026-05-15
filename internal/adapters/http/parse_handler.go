package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/exerayy/yadro-netlog-parser/internal/core/parser"
)

// LogID id лога
type LogID struct {
	LogID int64 `json:"log_id"`
}

// ParseRequest путь до архива
type ParseRequest struct {
	Path string `json:"path" example:"data/log.zip"`
}

// NewParseHandler создает обработчик для parse
// @Summary Парсинг логов
// @Description Выполняет парсинг логов из zip-архива. Архив должен содержать файлы .db_csv и .sharp_an_info
// @Tags logs
// @Accept json
// @Produce json
// @Param path query string true "Путь" example("data/log.zip")
// @Success 200 {object} LogID "Лог успешно обработан"
// @Failure 400 {string} string "Неверный путь или path traversal"
// @Failure 404 {string} string "Архив не найден"
// @Failure 422 {string} string "Ошибка валидации: архив поврежден, неверный формат CSV, отсутствуют обязательные файлы, нет данных для сохранения"
// @Failure 499 {string} string "Клиент закрыл соединение"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /api/v1/parse [post]
func NewParseHandler(log *slog.Logger, service parser.IParser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ParseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		logID, err := service.Parse(r.Context(), req.Path)
		if err != nil {
			log.Warn("parse failed", "path", req.Path, "error", err)

			if errors.Is(err, context.Canceled) {
				http.Error(w, "context canceled", httpStatusClientClosedRequest)
				return
			}

			if errors.Is(err, parser.ErrPathTraversal) {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}

			if errors.Is(err, parser.ErrArchiveNotFound) {
				http.Error(w, "archive not found", http.StatusNotFound)
				return
			}

			if errors.Is(err, parser.ErrInvalidArchive) ||
				errors.Is(err, parser.ErrRequiredFilesNotFound) ||
				errors.Is(err, parser.ErrInvalidCSVFormat) ||
				errors.Is(err, parser.ErrInvalidNodeRecord) {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSONFormatted(w, log, LogID{
			LogID: logID,
		})
	}
}
