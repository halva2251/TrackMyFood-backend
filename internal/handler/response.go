package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
)

type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	var buf bytes.Buffer
	resp := Response{Success: true, Data: data}
	if err := json.NewEncoder(&buf).Encode(resp); err != nil {
		slog.Error("failed to encode response", "error", err)
		http.Error(w, `{"success":false,"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

func WriteError(w http.ResponseWriter, status int, msg string) {
	var buf bytes.Buffer
	resp := Response{Success: false, Error: msg}
	if err := json.NewEncoder(&buf).Encode(resp); err != nil {
		slog.Error("failed to encode error response", "error", err)
		http.Error(w, `{"success":false,"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}
