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
	database, err := db.Connect(cfg)
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
	authHandler := handler.NewAuthHandler(cfg.JWTSecret, cfg.AdminBotID, cfg.AdminPass, cfg.BotTokens)

	// Init handlers
	msgHandler := handler.NewMessageHandler(database, hub)
	httpHandler := handler.NewHTTPHandler(database, hub, authHandler)

	// Setup router
	r := mux.NewRouter()

	// Health check (public)
	r.HandleFunc("/health", httpHandler.Health).Methods("GET")

	// Login (public)
	r.HandleFunc("/api/login", authHandler.Login).Methods("POST")

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
	api.HandleFunc("/messages/read", httpHandler.MarkRead).Methods("POST")
	api.HandleFunc("/friends", httpHandler.AddFriend).Methods("POST")
	api.HandleFunc("/friends/{user_id}", httpHandler.GetFriends).Methods("GET")
	api.HandleFunc("/groups/dm", httpHandler.CreateDMGroup).Methods("POST")

	// WebSocket - token-based auth
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
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      middleware.CORS(r),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
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
