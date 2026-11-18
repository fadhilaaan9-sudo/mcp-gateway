package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"mcp-gateway/config"
	"mcp-gateway/domain/gateway"
	"mcp-gateway/infrastructure/db"
	"mcp-gateway/mcpserver"
	"mcp-gateway/middlewares"
)

func main() {
	cfg := config.Load()

	if len(cfg.Tokens) == 0 {
		slog.Warn("tidak ada token di .env, semua /messages akan ditolak")
	}

	// Sambung ke MongoDB, maksimal 10 detik.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := db.Connect(ctx, cfg.MongoURI)
	if err != nil {
		slog.Error("gagal sambung MongoDB", "error", err)
		os.Exit(1)
	}
	defer client.Disconnect(context.Background())

	// Rangkai semua bagian: database -> repository -> handler -> router.
	repo := db.NewMongoMessageRepository(client.Database(cfg.MongoDB))
	if err := repo.EnsureIndexes(context.Background()); err != nil {
		slog.Error("gagal buat index", "error", err)
		os.Exit(1)
	}
	handler := gateway.NewHandler(repo)

	// Server MCP (gateway_send + gateway_read + gateway_list_topics) lewat HTTP di jalan /mcp.
	// Stateless: setiap request berdiri sendiri, tanpa session.
	mcpSrv := mcpserver.New(repo)
	httpMCP := server.NewStreamableHTTPServer(mcpSrv, server.WithStateLess(true))

	mux := http.NewServeMux()
	mux.HandleFunc("POST /messages", handler.SendMessage)
	mux.HandleFunc("GET /messages", handler.ListMessages)
	mux.HandleFunc("GET /topics", handler.ListTopics)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("/mcp", httpMCP)

	// Bungkus semua jalan dengan penjaga token.
	// /health tetap terbuka tanpa token, /messages, /topics, dan /mcp wajib bawa token.
	protected := middlewares.Auth(cfg.Tokens)(mux)

	slog.Info("gateway jalan", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, protected); err != nil {
		slog.Error("server berhenti", "error", err)
		os.Exit(1)
	}
}
