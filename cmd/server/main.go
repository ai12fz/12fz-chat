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

	// Init handlers
	msgHandler := handler.NewMessageHandler(database, hub)
	httpHandler := handler.NewHTTPHandler(database)

	// Setup router
	r := mux.NewRouter()

	// Health check
	r.HandleFunc("/health", httpHandler.Health).Methods("GET")

	// REST API
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/groups", httpHandler.CreateGroup).Methods("POST")
	api.HandleFunc("/groups", httpHandler.ListGroups).Methods("GET")
	api.HandleFunc("/groups/{id}/members", httpHandler.GetMembers).Methods("GET")
	api.HandleFunc("/groups/{id}/members", httpHandler.AddMember).Methods("POST")
	api.HandleFunc("/groups/my", httpHandler.GetMyGroups).Methods("GET")
	api.HandleFunc("/messages", httpHandler.GetMessages).Methods("GET")
	api.HandleFunc("/messages", httpHandler.SendMessage).Methods("POST")
	api.HandleFunc("/friends", httpHandler.AddFriend).Methods("POST")
	api.HandleFunc("/friends/{user_id}", httpHandler.GetFriends).Methods("GET")

	// WebSocket - bot connects here
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		botID := r.URL.Query().Get("bot_id")
		if botID == "" {
			http.Error(w, "missing bot_id", 400)
			return
		}
		hub.ServeWS(w, r, botID, msgHandler)
	})

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
