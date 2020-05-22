package server

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"

	"net/http"
)

// ClientIDURLParam is the url param used to "authenticate" the client.
const ClientIDURLParam = "clientId"

type handler struct {
	logger *zap.Logger
	db     *accessDB
}

func (h *handler) foo(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Start handling foo request.")

	keys, ok := r.URL.Query()[ClientIDURLParam]
	if !ok || len(keys) != 1 {
		h.logger.Debug(fmt.Sprintf("URL Param '%s' is missing", ClientIDURLParam),
			zap.Strings("keys", keys))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	clientID:=keys[0]
	if h.db.handleSingleAccess(clientID) {
		h.logger.Debug("Valid request", zap.String("clientID", clientID))
		w.WriteHeader(http.StatusOK)
		return
	}

	// too many requests by the client
	h.logger.Debug("Too many requests", zap.String("clientID", clientID))
	w.WriteHeader(http.StatusServiceUnavailable)
}

func (h *handler) updates(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Start handling updates request.")

	var update Update
	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.db.handleMultipleAccess(update)
	w.WriteHeader(http.StatusOK)
}
