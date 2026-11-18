package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Config kecil khusus bot, diisi dari config utama.
type Config struct {
	BotToken    string // token bot Telegram dari @BotFather
	GatewayURL  string // contoh http://localhost:8080
	GatewayAuth string // token gateway untuk bot (GATEWAY_TOKEN_BOT)
}

// watchedTopic adalah topik yang sedang ditunggu hasil akhirnya.
type watchedTopic struct {
	Text   string
	ChatID int64
}

// gatewayMessage adalah bentuk pesan gateway yang dibaca bot.
type gatewayMessage struct {
	ID      string `json:"id"`
	TopicID string `json:"topic_id"`
	Sender  string `json:"sender"`
	Round   int    `json:"round"`
	Content string `json:"content"`
	IsFinal bool   `json:"is_final"`
}

// Bot adalah bot Telegram untuk gateway.
// Tugasnya 2: terima topik dari Farhan, dan kirim hasil akhir
// ke Farhan kalau diskusi sudah SELESAI.
type Bot struct {
	api     *tgbotapi.BotAPI
	cfg     Config
	client  *http.Client
	watched map[string]watchedTopic
}

// Run menjalankan bot sampai dihentikan (Ctrl+C).
// Bot mengecek pesan masuk dari Telegram, dan tiap 30 detik
// mengecek gateway apakah ada topik yang sudah SELESAI.
func Run(ctx context.Context, cfg Config) error {
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return fmt.Errorf("gagal hubungi Telegram, cek token dan internet")
	}

	b := &Bot{
		api:     api,
		cfg:     cfg,
		client:  &http.Client{Timeout: 10 * time.Second},
		watched: map[string]watchedTopic{},
	}
	slog.Info("bot Telegram jalan", "username", api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := api.GetUpdatesChan(u)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-updates:
			if update.Message == nil {
				continue
			}
			b.handleMessage(update.Message)
		case <-ticker.C:
			b.checkFinals()
		}
	}
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	switch {
	case msg.IsCommand() && msg.Command() == "topik":
		b.handleTopik(msg)
	case msg.IsCommand() && msg.Command() == "status":
		b.handleStatus(msg)
	default:
		b.reply(msg.Chat.ID, "Perintah yang saya kenal:\n"+
			"/topik <topik + kriteria> — mulai diskusi baru.\n"+
			"/status — lihat topik yang sedang ditunggu.\n\n"+
			"Contoh:\n/topik Buatkan fungsi tambah di Go, harus ada tes dan komentar Bahasa Indonesia.")
	}
}

// handleTopik membuat topik baru di gateway dari pesan Farhan.
// Topik ditulis sebagai pesan pertama, pengirim "farhan", putaran 1.
func (b *Bot) handleTopik(msg *tgbotapi.Message) {
	text := strings.TrimSpace(msg.CommandArguments())
	if text == "" {
		b.reply(msg.Chat.ID, "Tulis topiknya setelah /topik. Contoh:\n/topik Buatkan fungsi sorting di Go, harus ada tes.")
		return
	}

	topicID := primitive.NewObjectID().Hex()
	body, _ := json.Marshal(map[string]any{
		"topic_id": topicID,
		"sender":   "farhan",
		"round":    1,
		"content":  text,
	})

	req, err := http.NewRequest(http.MethodPost, b.cfg.GatewayURL+"/messages", bytes.NewReader(body))
	if err != nil {
		b.reply(msg.Chat.ID, "gagal buat topik, coba lagi.")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.cfg.GatewayAuth)

	resp, err := b.client.Do(req)
	if err != nil {
		b.reply(msg.Chat.ID, "gateway tidak bisa dihubungi, pastikan server gateway jalan.")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		b.reply(msg.Chat.ID, fmt.Sprintf("gagal simpan topik: %v", errBody))
		return
	}

	b.watched[topicID] = watchedTopic{Text: text, ChatID: msg.Chat.ID}
	b.reply(msg.Chat.ID, "Topik dicatat. Muse 1 dan Muse 2 akan diskusi, hasilnya saya kirim ke sini kalau sudah SELESAI.\n\nID topik: "+topicID)
}

// handleStatus menampilkan topik yang sedang ditunggu.
func (b *Bot) handleStatus(msg *tgbotapi.Message) {
	if len(b.watched) == 0 {
		b.reply(msg.Chat.ID, "Tidak ada topik yang sedang ditunggu.")
		return
	}
	var lines []string
	for id, t := range b.watched {
		msgs, _ := b.listMessages(id)
		lines = append(lines, fmt.Sprintf("- %s (%d pesan)", snippet(t.Text), len(msgs)))
	}
	b.reply(msg.Chat.ID, "Topik yang ditunggu:\n"+strings.Join(lines, "\n"))
}

// checkFinals mengecek semua topik yang ditunggu.
// Kalau ada pesan SELESAI (is_final), kirim isinya ke Farhan
// lalu berhenti menunggu topik itu.
// Kalau sudah putaran 3 tapi belum ada hasil akhir,
// kirim kabar "belum selesai", bukan hasil yang salah.
func (b *Bot) checkFinals() {
	for topicID, topic := range b.watched {
		msgs, err := b.listMessages(topicID)
		if err != nil || len(msgs) == 0 {
			continue
		}

		for _, m := range msgs {
			if m.IsFinal {
				b.reply(topic.ChatID, "Hasil akhir diskusi:\n\n"+m.Content)
				delete(b.watched, topicID)
				break
			}
		}
		if _, ok := b.watched[topicID]; !ok {
			continue
		}

		lastRound := 0
		for _, m := range msgs {
			if m.Round > lastRound {
				lastRound = m.Round
			}
		}
		if lastRound >= 3 {
			b.reply(topic.ChatID, "Belum selesai: sudah 3 putaran tapi belum ada hasil akhir untuk topik:\n"+snippet(topic.Text))
			delete(b.watched, topicID)
		}
	}
}

// listMessages membaca semua pesan satu topik dari gateway.
func (b *Bot) listMessages(topicID string) ([]gatewayMessage, error) {
	req, err := http.NewRequest(http.MethodGet, b.cfg.GatewayURL+"/messages?topic_id="+topicID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+b.cfg.GatewayAuth)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway jawab %d", resp.StatusCode)
	}

	var msgs []gatewayMessage
	if err := json.NewDecoder(resp.Body).Decode(&msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}

func (b *Bot) reply(chatID int64, text string) {
	m := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(m); err != nil {
		slog.Warn("gagal kirim pesan Telegram", "error", err)
	}
}

// snippet memotong teks jadi pendek untuk tampilan status.
func snippet(s string) string {
	if len(s) > 40 {
		return s[:40] + "..."
	}
	return s
}
