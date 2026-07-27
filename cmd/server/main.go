package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ai12fz/12fz-chat/internal/config"
	"github.com/ai12fz/12fz-chat/internal/db"
	"github.com/ai12fz/12fz-chat/internal/handler"
	"github.com/ai12fz/12fz-chat/internal/middleware"
	"github.com/ai12fz/12fz-chat/internal/ws"
	"github.com/gorilla/mux"
)

func main() {
	cfg := config.Load()
	log.Printf("[chat] starting 12fz-chat on :%d", cfg.Port)

	// Connect DB
	database, err := db.ConnectBoth(cfg.PGConnStr, cfg.PlatformDSN)
	if err != nil {
		log.Fatalf("[chat] db connect: %v", err)
	}
	defer database.Close()

	// Auto migrate
	ctx := context.Background()
	if err := database.AutoMigrate(ctx); err != nil {
		log.Fatalf("[chat] migrate: %v", err)
	}
	log.Println("[chat] db ready")

	// Init hub
	hub := ws.NewHub()

	// Init auth handler
	authHandler := handler.NewAuthHandler(cfg.JWTSecret, cfg.BotTokens, database)

	// Init handlers
	msgHandler := handler.NewMessageHandler(database, hub)
	httpHandler := handler.NewHTTPHandler(database, hub, authHandler)

	// Setup router
	r := mux.NewRouter()

	// Health check (public)
	r.HandleFunc("/health", httpHandler.Health).Methods("GET")
	r.HandleFunc("/api/connections", httpHandler.ListConnections).Methods("GET")
	
	r.HandleFunc("/api/whoami", httpHandler.WhoAmI).Methods("GET")
	r.HandleFunc("/api/devices", httpHandler.PublicListDevices).Methods("GET")
	r.HandleFunc("/api/devices/{id}", httpHandler.PublicDeleteDevice).Methods("DELETE")

	// REST API (authenticated)
	api := r.PathPrefix("/api").Subrouter()
	api.Use(httpHandler.AuthMiddleware)
	api.HandleFunc("/groups", httpHandler.CreateGroup).Methods("POST")
	api.HandleFunc("/groups", httpHandler.ListGroups).Methods("GET")
	api.HandleFunc("/groups/{id}/members", httpHandler.GetMembers).Methods("GET")
	api.HandleFunc("/groups/{id}/members", httpHandler.AddMember).Methods("POST")
	api.HandleFunc("/groups/my", httpHandler.GetMyGroups).Methods("GET")
	api.HandleFunc("/messages", httpHandler.GetMessages).Methods("GET")
	api.HandleFunc("/messages", httpHandler.SendMessage).Methods("POST")
	api.HandleFunc("/messages/unread", httpHandler.GetUnreadCount).Methods("GET")
		// Agent CRUD
	api.HandleFunc("/agents", httpHandler.ListAgents).Methods("GET")
	api.HandleFunc("/agents", httpHandler.CreateAgent).Methods("POST")
	api.HandleFunc("/agents/{bot_id}", httpHandler.GetAgent).Methods("GET")
	api.HandleFunc("/agents/{bot_id}", httpHandler.UpdateAgent).Methods("PUT")
	api.HandleFunc("/agents/{bot_id}", httpHandler.DeleteAgent).Methods("DELETE")
	api.HandleFunc("/agents/{bot_id}/groups", httpHandler.AgentGroups).Methods("GET")
	api.HandleFunc("/agents/{bot_id}/groups", httpHandler.SetAgentGroups).Methods("PUT")


	

	r.HandleFunc("/api/devices/agents", httpHandler.DeviceAgents).Methods("GET")
	// Proxy admin (public - auth handled internally or via nginx)
	r.HandleFunc("/admin/proxy/dashboard", httpHandler.ProxyDashboard).Methods("GET")
	r.HandleFunc("/admin/proxy/models", httpHandler.ProxyListModels).Methods("GET")
	r.HandleFunc("/admin/proxy/models", httpHandler.ProxyCreateModel).Methods("POST")
	r.HandleFunc("/admin/proxy/models/{id}", httpHandler.ProxyUpdateModel).Methods("PUT")
	r.HandleFunc("/admin/proxy/models/{id}", httpHandler.ProxyDeleteModel).Methods("DELETE")
	r.HandleFunc("/admin/proxy/pricing", httpHandler.ProxyGetPricing).Methods("GET")
	r.HandleFunc("/admin/proxy/pricing/{key}", httpHandler.ProxyUpdatePricingItem).Methods("PUT")
	r.HandleFunc("/admin/proxy/pricing/multiplier", httpHandler.ProxyUpdateMultiplier).Methods("PUT")
	r.HandleFunc("/admin/proxy/merchants", httpHandler.ProxyListMerchants).Methods("GET")
	r.HandleFunc("/admin/proxy/merchants/{org_id}/recharge", httpHandler.ProxyRechargeMerchant).Methods("POST")
	r.HandleFunc("/admin/proxy/merchants/{org_id}/ledger", httpHandler.ProxyMerchantLedger).Methods("GET")
	r.HandleFunc("/admin/proxy/keys", httpHandler.ProxyListKeys).Methods("GET")
	r.HandleFunc("/admin/proxy/keys", httpHandler.ProxyCreateKey).Methods("POST")
	r.HandleFunc("/admin/proxy/keys/{id}/revoke", httpHandler.ProxyRevokeKey).Methods("POST")


	api.HandleFunc("/messages/read", httpHandler.MarkRead).Methods("POST")
	api.HandleFunc("/friends", httpHandler.AddFriend).Methods("POST")
	api.HandleFunc("/friends/{user_id}", httpHandler.GetFriends).Methods("GET")
	api.HandleFunc("/agent-status", httpHandler.GetAgentStatus).Methods("GET")
	api.HandleFunc("/whoami", httpHandler.WhoAmI).Methods("GET")
	api.HandleFunc("/users/{id}", httpHandler.GetUserInfo).Methods("GET")
	api.HandleFunc("/whoami", httpHandler.WhoAmI).Methods("GET")
	api.HandleFunc("/users/{id}", httpHandler.GetUserInfo).Methods("GET")
	api.HandleFunc("/whoami", httpHandler.WhoAmI).Methods("GET")
	api.HandleFunc("/friends/action", httpHandler.HandleFriendRequest).Methods("POST")
	api.HandleFunc("/friend-messages", httpHandler.SendFriendMessage).Methods("POST")
	api.HandleFunc("/friend-messages", httpHandler.GetFriendMessages).Methods("GET")

	// LLM proxy to new-api
	r.HandleFunc("/v1/chat/completions", httpHandler.ProxyChat).Methods("POST", "OPTIONS")
	r.HandleFunc("/v1/models", httpHandler.ProxyModels).Methods("GET", "OPTIONS")

	// WebSocket - token-based auth
	api.HandleFunc("/device-reg-codes", httpHandler.GenerateRegCode).Methods("POST")
	api.HandleFunc("/device-reg-codes", httpHandler.ListRegCodes).Methods("GET")
	api.HandleFunc("/device-reg-codes/{code}", httpHandler.RevokeRegCode).Methods("DELETE")
	r.HandleFunc("/api/devices/register", httpHandler.RegisterDevice).Methods("POST")
	r.HandleFunc("/api/devices/setup", httpHandler.DeviceSetup).Methods("GET")
		r.HandleFunc("/api/devices/agents", httpHandler.DeviceAgents).Methods("GET")
	// Proxy admin (public - auth handled internally or via nginx)
	r.HandleFunc("/admin/proxy/dashboard", httpHandler.ProxyDashboard).Methods("GET")
	r.HandleFunc("/admin/proxy/models", httpHandler.ProxyListModels).Methods("GET")
	r.HandleFunc("/admin/proxy/models", httpHandler.ProxyCreateModel).Methods("POST")
	r.HandleFunc("/admin/proxy/models/{id}", httpHandler.ProxyUpdateModel).Methods("PUT")
	r.HandleFunc("/admin/proxy/models/{id}", httpHandler.ProxyDeleteModel).Methods("DELETE")
	r.HandleFunc("/admin/proxy/pricing", httpHandler.ProxyGetPricing).Methods("GET")
	r.HandleFunc("/admin/proxy/pricing/{key}", httpHandler.ProxyUpdatePricingItem).Methods("PUT")
	r.HandleFunc("/admin/proxy/pricing/multiplier", httpHandler.ProxyUpdateMultiplier).Methods("PUT")
	r.HandleFunc("/admin/proxy/merchants", httpHandler.ProxyListMerchants).Methods("GET")
	r.HandleFunc("/admin/proxy/merchants/{org_id}/recharge", httpHandler.ProxyRechargeMerchant).Methods("POST")
	r.HandleFunc("/admin/proxy/merchants/{org_id}/ledger", httpHandler.ProxyMerchantLedger).Methods("GET")
	r.HandleFunc("/admin/proxy/keys", httpHandler.ProxyListKeys).Methods("GET")
	r.HandleFunc("/admin/proxy/keys", httpHandler.ProxyCreateKey).Methods("POST")
	r.HandleFunc("/admin/proxy/keys/{id}/revoke", httpHandler.ProxyRevokeKey).Methods("POST")

	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", 401)
			return
		}
		botID, err := authHandler.ValidateToken(token)
		if err != nil {
			http.Error(w, "invalid token: "+err.Error(), 401)
			return
		}
		log.Printf("[chat] WS connect: bot_id=%s", botID)
		hub.ServeWS(w, r, botID, msgHandler)
	})

	// Serve static frontend
	r.PathPrefix("/").Handler(httpHandler.StaticHandler())

	// Apply CORS
		// WS-only server on port 8082 (no timeouts)
	wsMux := http.NewServeMux()
	wsMux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", 401)
			return
		}
		botID, err := authHandler.ValidateToken(token)
		if err != nil {
			http.Error(w, "invalid token: "+err.Error(), 401)
			return
		}
		log.Printf("[chat] WS2 connect: bot_id=%s", botID)
		hub.ServeWS(w, r, botID, msgHandler)
	})
	go func() {
		wsSrv := &http.Server{Addr: ":8082", Handler: middleware.CORS(wsMux)}
		log.Printf("WS2 server listening on :8082")
		wsSrv.ListenAndServe()
	}()

srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      middleware.CORS(r),
		ReadTimeout:  0,
		WriteTimeout: 0,
		IdleTimeout:  0,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh
		log.Println("[chat] shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("[chat] listening on :%d", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[chat] server error: %v", err)
	}
	log.Println("[chat] stopped")
}
