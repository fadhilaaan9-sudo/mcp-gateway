package mcpserver

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"mcp-gateway/domain/gateway"
)

// New membuat server MCP dengan 3 tool:
//   - gateway_send: kirim satu pesan diskusi ke gateway.
//   - gateway_read: baca semua pesan dalam satu topik.
//   - gateway_list_topics: lihat daftar topik terbaru.
//
// Server ini dijalankan lewat HTTP (Streamable HTTP) di jalan /mcp,
// jadi akun Muse bisa memanggilnya dari jauh, sama seperti REST.
func New(repo gateway.MessageRepository) *server.MCPServer {
	s := server.NewMCPServer(
		"mcp-gateway",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	sendTool := mcp.NewTool("gateway_send",
		mcp.WithDescription("Kirim satu pesan diskusi ke gateway. Satu panggilan = satu pesan, satu putaran."),
		mcp.WithString("topic_id", mcp.Required(), mcp.Description("ID topik, 24 karakter hex.")),
		mcp.WithString("sender", mcp.Required(), mcp.Description("Nama pengirim, contoh muse-1 atau muse-2.")),
		mcp.WithNumber("round", mcp.Required(), mcp.Description("Nomor putaran, 1 sampai 3.")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Isi pesan, maksimal 500 kata.")),
		mcp.WithBoolean("is_final", mcp.Description("true kalau ini hasil akhir SELESAI. Hanya muse-2 yang boleh.")),
	)

	readTool := mcp.NewTool("gateway_read",
		mcp.WithDescription("Baca semua pesan dalam satu topik, urut dari yang paling lama."),
		mcp.WithString("topic_id", mcp.Required(), mcp.Description("ID topik, 24 karakter hex.")),
	)

	listTool := mcp.NewTool("gateway_list_topics",
		mcp.WithDescription("Lihat daftar topik terbaru, urut dari yang paling baru. Dipakai supaya akun bisa menemukan topik baru sendiri, tanpa harus diberi tahu dulu ID topiknya."),
	)

	s.AddTool(sendTool, sendHandler(repo))
	s.AddTool(readTool, readHandler(repo))
	s.AddTool(listTool, listHandler(repo))

	return s
}

func sendHandler(repo gateway.MessageRepository) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		topicID, err := req.RequireString("topic_id")
		if err != nil {
			return mcp.NewToolResultError("topic_id wajib diisi"), nil
		}
		sender, err := req.RequireString("sender")
		if err != nil {
			return mcp.NewToolResultError("sender wajib diisi"), nil
		}
		content, err := req.RequireString("content")
		if err != nil {
			return mcp.NewToolResultError("content wajib diisi"), nil
		}
		round := req.GetInt("round", 0)
		isFinal := req.GetBool("is_final", false)

		topicOID, err := primitive.ObjectIDFromHex(topicID)
		if err != nil {
			return mcp.NewToolResultError("topic_id tidak valid"), nil
		}

		msg, err := gateway.NewMessage(topicOID, sender, round, content)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if isFinal {
			msg.MarkFinal()
		}

		if err := repo.Save(ctx, msg); err != nil {
			return mcp.NewToolResultError("gagal simpan pesan"), nil
		}

		out, err := json.Marshal(msg)
		if err != nil {
			return mcp.NewToolResultError("gagal baca pesan tersimpan"), nil
		}
		return mcp.NewToolResultText(string(out)), nil
	}
}

func readHandler(repo gateway.MessageRepository) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		topicID, err := req.RequireString("topic_id")
		if err != nil {
			return mcp.NewToolResultError("topic_id wajib diisi"), nil
		}

		topicOID, err := primitive.ObjectIDFromHex(topicID)
		if err != nil {
			return mcp.NewToolResultError("topic_id tidak valid"), nil
		}

		msgs, err := repo.ListByTopic(ctx, topicOID)
		if err != nil {
			return mcp.NewToolResultError("gagal baca pesan"), nil
		}
		if msgs == nil {
			msgs = []*gateway.Message{}
		}

		out, err := json.Marshal(msgs)
		if err != nil {
			return mcp.NewToolResultError("gagal baca pesan"), nil
		}
		return mcp.NewToolResultText(string(out)), nil
	}
}

func listHandler(repo gateway.MessageRepository) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		topics, err := repo.ListTopics(ctx)
		if err != nil {
			return mcp.NewToolResultError("gagal baca topik"), nil
		}
		if topics == nil {
			topics = []gateway.Topic{}
		}

		out, err := json.Marshal(topics)
		if err != nil {
			return mcp.NewToolResultError("gagal baca topik"), nil
		}
		return mcp.NewToolResultText(string(out)), nil
	}
}
