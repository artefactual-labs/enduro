package objectevent

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-logr/logr"
)

const maxRequestBodyBytes = 1 << 20

func HTTPServer(logger logr.Logger, config *Config, publisher Publisher) *http.Server {
	mux := http.NewServeMux()
	h := &handler{
		logger:    logger.WithName("objectevent"),
		config:    config,
		publisher: publisher,
	}
	mux.HandleFunc("POST /seaweedfs/events", h.seaweedFSEvents)

	return &http.Server{
		Addr:         config.Listen,
		Handler:      mux,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		IdleTimeout:  time.Second * 120,
	}
}

type handler struct {
	logger    logr.Logger
	config    *Config
	publisher Publisher
}

func (h *handler) seaweedFSEvents(w http.ResponseWriter, r *http.Request) {
	var event seaweedFSEvent
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)).Decode(&event); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	normalized, ok, err := enduroEventFromSeaweedFS(event, h.config.BucketsPath)
	if err != nil {
		http.Error(w, "invalid SeaweedFS event", http.StatusBadRequest)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.publisher.Publish(r.Context(), normalized); err != nil {
		if errors.Is(err, r.Context().Err()) {
			return
		}
		h.logger.Error(err, "Error publishing object event.")
		http.Error(w, "error publishing object event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
