package api //nolint:revive // api is a clear and conventional package name

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// writeJSON writes a JSON body with the given status code.
func writeJSON(logger *logrus.Logger, w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil && logger != nil {
		logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// respond writes a successful Response envelope.
func respond(logger *logrus.Logger, w http.ResponseWriter, status int, data any) {
	writeJSON(logger, w, status, Response{Success: true, Data: data, Timestamp: time.Now()})
}
