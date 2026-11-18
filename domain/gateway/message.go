package gateway

import (
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message adalah satu pesan di gateway.
// Satu pesan = satu dokumen di collection "messages" MongoDB.
// Tidak ada relasi ke tabel lain, jadi bentuknya tetap sama.
type Message struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TopicID   primitive.ObjectID `bson:"topic_id" json:"topic_id"`
	Sender    string             `bson:"sender" json:"sender"`
	Round     int                `bson:"round" json:"round"`
	Content   string             `bson:"content" json:"content"`
	IsFinal   bool               `bson:"is_final" json:"is_final"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// NewMessage membuat satu pesan baru.
// Aturan yang dicek di sini sama seperti desain:
// round 1 sampai 3, content tidak kosong, maksimal 500 kata.
func NewMessage(topicID primitive.ObjectID, sender string, round int, content string) (*Message, error) {
	m := &Message{
		TopicID:   topicID,
		Sender:    strings.TrimSpace(sender),
		Round:     round,
		Content:   strings.TrimSpace(content),
		CreatedAt: time.Now(),
	}
	if err := m.validate(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Message) validate() error {
	if m.TopicID.IsZero() {
		return errors.New("topic_id tidak boleh kosong")
	}
	if m.Sender == "" {
		return errors.New("sender tidak boleh kosong")
	}
	if m.Round < 1 || m.Round > 3 {
		return errors.New("round harus 1 sampai 3")
	}
	if m.Content == "" {
		return errors.New("content tidak boleh kosong")
	}
	if len(strings.Fields(m.Content)) > 500 {
		return errors.New("content maksimal 500 kata")
	}
	return nil
}

// MarkFinal menandai pesan ini sebagai hasil akhir (SELESAI).
// Hanya Muse 2 (pemeriksa) yang boleh memanggil ini.
func (m *Message) MarkFinal() {
	m.IsFinal = true
}

// Topic adalah satu baris di daftar topik: pesan pertama dari satu topik.
// Dipakai gateway_list_topics supaya akun bisa menemukan topik baru
// sendiri, tanpa harus diberi tahu dulu ID topiknya.
type Topic struct {
	TopicID   primitive.ObjectID `bson:"topic_id" json:"topic_id"`
	Content   string             `bson:"content" json:"content"`
	Sender    string             `bson:"sender" json:"sender"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
