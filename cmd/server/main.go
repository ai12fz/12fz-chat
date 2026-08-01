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
	if err := database.EnsureAnalyticsTable(ctx); err != nil {
		log.Printf("[chat] analytics table: %v", err)
	}
	if err := database.AutoMigrate(ctx); err != nil {
		log.Fatalf("[chat] migrate: %v", err)
	}
	log.Println("[chat] db ready")

	// Reset all device statuses to offline — only WebSocket connection sets online
	if err := database.ResetDeviceStatus(ctx); err != nil {
		log.Printf("[chat] reset device status: %v", err)
	}

	// Init hub
	hub := ws.NewHubWithDB(database)

	// Init auth handler
	authHandler := handler.NewAuthHandler(cfg.JWTSecret, cfg.BotTokens, database)

	// Init handlers
	msgHandler := handler.NewMessageHandler(database, hub)
	httpHandler := handler.NewHTTPHandler(database, hub, authHandler, cfg.DocsDir)

	// Setup router
	r := mux.NewRouter()

	// Health check (public)
	r.HandleFunc("/health", httpHandler.Health).Methods("GET")
	r.HandleFunc("/api/connections", httpHandler.ListConnections).Methods("GET")
	
	r.HandleFunc("/api/whoami", httpHandler.WhoAmI).Methods("GET")
	r.HandleFunc("/api/v1/track", httpHandler.TrackEvent).Methods("POST")
	r.HandleFunc("/api/v1/analytics/overview", httpHandler.AnalyticsOverview).Methods("GET")
	r.HandleFunc("/analytics", httpHandler.AnalyticsPage).Methods("GET")
	r.HandleFunc("/api/sso/exchange", httpHandler.SsoExchange).Methods("POST")
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
	api.HandleFunc("/agents/{bot_id}/heartbeat", httpHandler.AgentHeartbeat).Methods("POST")


	



	api.HandleFunc("/messages/read", httpHandler.MarkRead).Methods("POST")
	api.HandleFunc("/friends/{id}/category", httpHandler.UpdateFriendCategory).Methods("PATCH")
	api.HandleFunc("/friends", httpHandler.AddFriend).Methods("POST")
	api.HandleFunc("/friends/{user_id}", httpHandler.GetFriends).Methods("GET")
	api.HandleFunc("/friends/{id}/grant", httpHandler.GrantFriend).Methods("POST")
	api.HandleFunc("/org/staff", httpHandler.ListOrgStaff).Methods("GET")
	api.HandleFunc("/agent-status", httpHandler.GetAgentStatus).Methods("GET")

	api.HandleFunc("/whoami", httpHandler.WhoAmI).Methods("GET")
	api.HandleFunc("/users/{id}", httpHandler.GetUserInfo).Methods("GET")
	api.HandleFunc("/friends/action", httpHandler.HandleFriendRequest).Methods("POST")
	api.HandleFunc("/friend-messages", httpHandler.SendFriendMessage).Methods("POST")
	api.HandleFunc("/friend-messages", httpHandler.GetFriendMessages).Methods("GET")
	// Documents (merchant-scoped files produced by agents/bots)
	api.HandleFunc("/documents", httpHandler.UploadDocument).Methods("POST")
	api.HandleFunc("/documents", httpHandler.ListDocuments).Methods("GET")
	api.HandleFunc("/documents/{id}/download", httpHandler.DownloadDocument).Methods("GET")
	api.HandleFunc("/documents/{id}/preview", httpHandler.PreviewDocument).Methods("GET")
	api.HandleFunc("/device-command", httpHandler.DeviceCommand).Methods("POST")

	// Admin API: /api/admin/* (AuthMiddleware + AdminOnly) — migrated from public /admin/proxy/*
	admin := api.PathPrefix("/admin").Subrouter()
	admin.Use(httpHandler.AdminOnly)
	admin.HandleFunc("/proxy/dashboard", httpHandler.ProxyDashboard).Methods("GET")
	admin.HandleFunc("/proxy/models", httpHandler.ProxyListModels).Methods("GET")
	admin.HandleFunc("/proxy/models", httpHandler.ProxyCreateModel).Methods("POST")
	admin.HandleFunc("/proxy/models/{id}", httpHandler.ProxyUpdateModel).Methods("PUT")
	admin.HandleFunc("/proxy/models/{id}", httpHandler.ProxyDeleteModel).Methods("DELETE")
	admin.HandleFunc("/proxy/pricing", httpHandler.ProxyGetPricing).Methods("GET")
	admin.HandleFunc("/proxy/pricing/{key}", httpHandler.ProxyUpdatePricingItem).Methods("PUT")
	admin.HandleFunc("/proxy/pricing/multiplier", httpHandler.ProxyUpdateMultiplier).Methods("PUT")
	admin.HandleFunc("/proxy/merchants", httpHandler.ProxyListMerchants).Methods("GET")
	admin.HandleFunc("/proxy/merchants/{org_id}/recharge", httpHandler.ProxyRechargeMerchant).Methods("POST")
	admin.HandleFunc("/proxy/merchants/{org_id}/ledger", httpHandler.ProxyMerchantLedger).Methods("GET")
	admin.HandleFunc("/proxy/keys", httpHandler.ProxyListKeys).Methods("GET")
	admin.HandleFunc("/proxy/keys", httpHandler.ProxyCreateKey).Methods("POST")
	admin.HandleFunc("/proxy/keys/{id}/revoke", httpHandler.ProxyRevokeKey).Methods("POST")
	admin.HandleFunc("/doc-quota", httpHandler.AdminDocQuota).Methods("GET", "PUT")

	// LLM proxy to new-api
	r.HandleFunc("/v1/chat/completions", httpHandler.ProxyChat).Methods("POST", "OPTIONS")
	r.HandleFunc("/v1/models", httpHandler.ProxyModels).Methods("GET", "OPTIONS")

	// WebSocket - token-based auth
	api.HandleFunc("/device-reg-codes", httpHandler.GenerateRegCode).Methods("POST")
	api.HandleFunc("/device-reg-codes", httpHandler.ListRegCodes).Methods("GET")
	api.HandleFunc("/device-reg-codes/{code}", httpHandler.RevokeRegCode).Methods("DELETE")
	r.HandleFunc("/api/devices/register", httpHandler.RegisterDevice).Methods("POST")
	r.HandleFunc("/api/devices/setup", httpHandler.DeviceSetup).Methods("GET")
	api.HandleFunc("/devices/heartbeat", httpHandler.DeviceHeartbeat).Methods("POST")
	r.HandleFunc("/api/devices/{id}/toggle-skills", httpHandler.ToggleDeviceSkills).Methods("POST")
	r.HandleFunc("/api/devices/{id}/toggle-software", httpHandler.ToggleDeviceSoftware).Methods("POST")
	r.HandleFunc("/api/devices/{id}/model", httpHandler.GetDeviceModel).Methods("GET")
	r.HandleFunc("/api/devices/{id}/model", httpHandler.SetDeviceModel).Methods("PUT")
	r.HandleFunc("/api/devices/{id}/activity", httpHandler.GetDeviceActivity).Methods("GET")
	api.HandleFunc("/devices/activity", httpHandler.PostDeviceActivity).Methods("POST")

	r.HandleFunc("/api/skills", httpHandler.ListSkills).Methods("GET")
	r.HandleFunc("/api/capabilities", httpHandler.ListCapabilities).Methods("GET")
	r.HandleFunc("/api/skills/{name}", httpHandler.UpdateSkill).Methods("PUT")
	r.HandleFunc("/api/skills/{name}", httpHandler.DeleteSkill).Methods("DELETE")
		r.HandleFunc("/api/devices/agents", httpHandler.DeviceAgents).Methods("GET")

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
		hub.ServeWSGorilla(w, r, botID, msgHandler)
	})

	// Serve static frontend
		r.HandleFunc("/api/test-log", func(w http.ResponseWriter, r *http.Request) {
		_, err := database.LogProxyUsage(r.Context(), "00000000-0000-0000-0000-000000000000", 0, "test-model", 999)
		if err != nil {
			w.Write([]byte("err: " + err.Error()))
		} else {
			w.Write([]byte("ok"))
		}
	}).Methods("GET")

	// Hermes bridge script (public download)
	r.HandleFunc("/hermes-bridge.py", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.ServeFile(w, r, "static/hermes-bridge.py")
	}).Methods("GET")

	r.PathPrefix("/").Handler(httpHandler.StaticHandler())

		// Hourly cleanup: keep only 50 messages per conversation
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			n, err := database.CleanupOldMessages(context.Background())
			if err != nil {
				log.Printf("[cleanup] err: %v", err)
			} else if n > 0 {
				log.Printf("[cleanup] deleted %d old messages", n)
			}
		}
	}()


srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/ws" || req.URL.Path == "/ws/" {
				token := req.URL.Query().Get("token")
				if token == "" { http.Error(w, "missing token", 401); return }
				botID, err := authHandler.ValidateToken(token)
				if err != nil { http.Error(w, "invalid token", 401); return }
				log.Printf("[chat] WS connect: bot_id=%s", botID)
				hub.ServeWSGorilla(w, req, botID, msgHandler)
				return
			}
			r.ServeHTTP(w, req)
		}),
		ReadTimeout:  0,
		WriteTimeout: 0,
		IdleTimeout:  0,
	}

		// Offline detection: mark devices offline if no heartbeat for 90s
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err := database.Pool().Exec(ctx,
				"UPDATE chat.devices SET status='offline' WHERE status='online' AND last_seen < NOW() - INTERVAL '90 seconds'")
			cancel()
			if err != nil {
				log.Printf("[heartbeat] offline detector error: %v", err)
			}
		}
	}()

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
