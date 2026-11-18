package gateway

import (
	"encoding/json"
	"net/http"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Handler adalah pintu masuk HTTP untuk kirim dan baca pesan.
// Ia hanya bicara ke MessageRepository, tidak langsung ke MongoDB.
type Handler struct {
	repo MessageRepository
}

// NewHandler membuat handler baru dari satu repository.
func NewHandler(repo MessageRepository) *Handler {
	return &Handler{repo: repo}
}

// sendRequest adalah bentuk JSON yang dikirim saat kirim pesan.
type sendRequest struct {
	TopicID string `json:"topic_id"`
	Sender  string `json:"sender"`
	Round   int    `json:"round"`
	Content string `json:"content"`
	IsFinal bool   `json:"is_final"`
}

// SendMessage menerima satu pesan baru (POST) dan menyimpannya.
// Kalau is_final true, pesan ini ditandai SELESAI.
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body json tidak valid")
		return
	}

	topicID, err := primitive.ObjectIDFromHex(req.TopicID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "topic_id tidak valid")
		return
	}

	msg, err := NewMessage(topicID, req.Sender, req.Round, req.Content)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.IsFinal {
		msg.MarkFinal()
	}

	if err := h.repo.Save(r.Context(), msg); err != nil {
		writeError(w, http.StatusInternalServerError, "gagal simpan pesan")
		return
	}

	writeJSON(w, http.StatusCreated, msg)
}

// ListMessages membaca semua pesan dalam satu topik (GET).
// Contoh: GET /messages?topic_id=...
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	topicID, err := primitive.ObjectIDFromHex(r.URL.Query().Get("topic_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "topic_id tidak valid")
		return
	}

	messages, err := h.repo.ListByTopic(r.Context(), topicID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal baca pesan")
		return
	}

	writeJSON(w, http.StatusOK, messages)
}

// ListTopics membaca daftar topik terbaru (GET /topics).
// Dipakai akun Muse supaya bisa menemukan topik baru sendiri,
// tanpa harus diberi tahu dulu ID topiknya.
func (h *Handler) ListTopics(w http.ResponseWriter, r *http.Request) {
	topics, err := h.repo.ListTopics(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal baca topik")
		return
	}

	writeJSON(w, http.StatusOK, topics)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
