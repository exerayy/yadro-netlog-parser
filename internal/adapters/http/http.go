package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

const httpStatusClientClosedRequest int = 499

func writeJSONFormatted(w http.ResponseWriter, log *slog.Logger, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	formattedJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Error("failed to marshal JSON", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err = w.Write(formattedJSON); err != nil {
		log.Error("failed to write JSON response", "error", err)
	}
}
