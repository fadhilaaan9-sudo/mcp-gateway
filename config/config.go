package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config adalah semua setting gateway yang dibaca dari environment.
// Nilainya nanti diisi dari file .env di laptop, bukan ditulis di kode.
type Config struct {
	MongoURI string
	MongoDB  string
	Port     string
	// Tokens adalah daftar token yang boleh masuk, kunci: nama pengirim.
	// Contoh: "muse-1" -> token dari GATEWAY_TOKEN_MUSE_1.
	// Kalau nanti tambah akun, cukup tambah 1 baris di bawah (MUSE_3)
	// dan 1 baris di .env, tanpa ubah kode lain.
	Tokens map[string]string
	// TelegramBotToken adalah token bot Telegram dari @BotFather.
	TelegramBotToken string
	// GatewayURL adalah alamat gateway yang dipakai bot,
	// contoh http://localhost:8080.
	GatewayURL string
	// BotToken adalah token gateway khusus bot (GATEWAY_TOKEN_BOT),
	// supaya bot bisa tulis dan baca pesan di gateway.
	BotToken string
}

// Load membaca setting dari file .env kalau ada,
// lalu dari environment. Kalau tidak ada, pakai nilai default
// untuk jalan di laptop.
func Load() Config {
	// godotenv.Load hanya mengembalikan 1 nilai (error),
	// jadi pakai _ = untuk mengabaikannya kalau .env tidak ada.
	_ = godotenv.Load()

	tokens := map[string]string{}
	if t := os.Getenv("GATEWAY_TOKEN_MUSE_1"); t != "" {
		tokens["muse-1"] = t
	}
	if t := os.Getenv("GATEWAY_TOKEN_MUSE_2"); t != "" {
		tokens["muse-2"] = t
	}
	if t := os.Getenv("GATEWAY_TOKEN_MUSE_3"); t != "" {
		tokens["muse-3"] = t
	}
	if t := os.Getenv("GATEWAY_TOKEN_BOT"); t != "" {
		tokens["bot"] = t
	}

	return Config{
		MongoURI:         getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDB:          getEnv("MONGODB_DB", "gateway"),
		Port:             getEnv("PORT", "8080"),
		Tokens:           tokens,
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		GatewayURL:       getEnv("GATEWAY_URL", "http://localhost:8080"),
		BotToken:         getEnv("GATEWAY_TOKEN_BOT", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
