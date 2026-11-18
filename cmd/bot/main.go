package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"mcp-gateway/config"
	"mcp-gateway/telegram"
)

// Pintu masuk bot Telegram. Jalankan di terminal kedua,
// sementara server gateway jalan di terminal pertama:
// go run ./cmd/bot
func main() {
	cfg := config.Load()

	if cfg.TelegramBotToken == "" {
		slog.Error("TELEGRAM_BOT_TOKEN kosong di .env, buat bot dulu di @BotFather")
		os.Exit(1)
	}
	if cfg.BotToken == "" {
		slog.Error("GATEWAY_TOKEN_BOT kosong di .env")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := telegram.Run(ctx, telegram.Config{
		BotToken:    cfg.TelegramBotToken,
		GatewayURL:  cfg.GatewayURL,
		GatewayAuth: cfg.BotToken,
	}); err != nil {
		slog.Error("bot berhenti, cek token bot dan internet")
		os.Exit(1)
	}
}
