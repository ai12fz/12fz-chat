package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// ── Dashboard ──

func (h *HTTPHandler) ProxyDashboard(w http.ResponseWriter, r *http.Request) {
	keyID := r.URL.Query().Get("key_id")
	today, month, daily, topModels := h.db.ProxyDashboard(r.Context(), keyID)
	jsonResp(w, map[string]interface{}{
		"today":      today,
		"month":      month,
		"daily":      daily,
		"top_models": topModels,
	}, 200)
}

// ── Models CRUD ──

func (h *HTTPHandler) ProxyListModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.db.ProxyListModels(r.Context())
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, models, 200)
}

func (h *HTTPHandler) ProxyCreateModel(w http.ResponseWriter, r *http.Request) {
	var m map[string]interface{}
	json.NewDecoder(r.Body).Decode(&m)
	if err := h.db.ProxyCreateModel(r.Context(), m); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 201)
}

func (h *HTTPHandler) ProxyUpdateModel(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var m map[string]interface{}
	json.NewDecoder(r.Body).Decode(&m)
	if err := h.db.ProxyUpdateModel(r.Context(), id, m); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

func (h *HTTPHandler) ProxyDeleteModel(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.db.ProxyDeleteModel(r.Context(), id); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

// ── Pricing ──

func (h *HTTPHandler) ProxyGetPricing(w http.ResponseWriter, r *http.Request) {
	items, multiplier, err := h.db.ProxyGetPricing(r.Context())
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]interface{}{"items": items, "multiplier": multiplier}, 200)
}

func (h *HTTPHandler) ProxyUpdatePricingItem(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	var body struct {
		Amount *float64 `json:"amount"`
		Active *bool    `json:"active"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if err := h.db.ProxyUpdatePricingItem(r.Context(), key, body.Amount, body.Active); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

func (h *HTTPHandler) ProxyUpdateMultiplier(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Multiplier float64 `json:"multiplier"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if err := h.db.ProxyUpdateMultiplier(r.Context(), body.Multiplier); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

// ── Merchants (deduplicated by org_id) ──

func (h *HTTPHandler) ProxyListMerchants(w http.ResponseWriter, r *http.Request) {
	merchants, err := h.db.ProxyListMerchants(r.Context())
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, merchants, 200)
}

func (h *HTTPHandler) ProxyRechargeMerchant(w http.ResponseWriter, r *http.Request) {
	orgID := mux.Vars(r)["org_id"]
	var body struct {
		Amount float64 `json:"amount"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if err := h.db.ProxyRechargeMerchant(r.Context(), orgID, body.Amount); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}

func (h *HTTPHandler) ProxyMerchantLedger(w http.ResponseWriter, r *http.Request) {
	orgID := mux.Vars(r)["org_id"]
	ledger, err := h.db.ProxyMerchantLedger(r.Context(), orgID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, ledger, 200)
}

// ── Keys ──

func (h *HTTPHandler) ProxyListKeys(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("org_id")
	keys, err := h.db.ProxyListKeys(r.Context(), orgID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, keys, 200)
}

func (h *HTTPHandler) ProxyCreateKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OrgID string `json:"org_id"`
		Name  string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	key, err := h.db.ProxyCreateKey(r.Context(), body.OrgID, body.Name)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, key, 201)
}

func (h *HTTPHandler) ProxyRevokeKey(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.db.ProxyRevokeKey(r.Context(), id); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"}, 200)
}
