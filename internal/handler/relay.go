package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
)

var (
	relayFile = "/tmp/12fz-relay.json"
	relayMu   sync.RWMutex
)

type RelayStatus struct {
	Enabled bool `json:"enabled"`
}

func getRelayStatus() RelayStatus {
	relayMu.RLock()
	defer relayMu.RUnlock()
	data, err := os.ReadFile(relayFile)
	if err != nil {
		return RelayStatus{Enabled: true}
	}
	var s RelayStatus
	json.Unmarshal(data, &s)
	return s
}

func setRelayStatus(enabled bool) {
	relayMu.Lock()
	defer relayMu.Unlock()
	s := RelayStatus{Enabled: enabled}
	data, _ := json.Marshal(s)
	os.WriteFile(relayFile, data, 0644)
}

func (h *HTTPHandler) GetRelayStatus(w http.ResponseWriter, r *http.Request) {
	s := getRelayStatus()
	jsonResp(w, s, 200)
}

func (h *HTTPHandler) ToggleRelay(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	if req.Enabled == nil {
		current := getRelayStatus()
		setRelayStatus(!current.Enabled)
	} else {
		setRelayStatus(*req.Enabled)
	}
	s := getRelayStatus()
	jsonResp(w, s, 200)
}
