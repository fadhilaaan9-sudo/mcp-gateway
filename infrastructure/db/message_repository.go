package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mcp-gateway/domain/gateway"
)

// MongoMessageRepository menyimpan dan membaca Message
// dari collection "messages" di MongoDB.
type MongoMessageRepository struct {
	collection *mongo.Collection
}

// NewMongoMessageRepository membuat repository baru dari satu database.
// Nama collection sudah tetap "messages" sesuai desain.
func NewMongoMessageRepository(db *mongo.Database) *MongoMessageRepository {
	return &MongoMessageRepository{
		collection: db.Collection("messages"),
	}
}

// EnsureIndexes membuat index {topic_id: 1, created_at: 1} sesuai desain.
// Index ini bikin baca pesan satu topik jadi cepat dan selalu urut waktu.
func (r *MongoMessageRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "topic_id", Value: 1},
			{Key: "created_at", Value: 1},
		},
	})
	return err
}

// Save menyimpan satu pesan baru ke MongoDB.
// InsertOne tidak mengubah struct yang dikirim, jadi id yang dibuat
// MongoDB harus disalin sendiri dari InsertedID, supaya jawaban POST
// membawa id yang asli, bukan nol.
func (r *MongoMessageRepository) Save(ctx context.Context, m *gateway.Message) error {
	res, err := r.collection.InsertOne(ctx, m)
	if err != nil {
		return err
	}
	if id, ok := res.InsertedID.(primitive.ObjectID); ok {
		m.ID = id
	}
	return nil
}

// ListByTopic membaca semua pesan dalam satu topik,
// diurutkan dari yang paling lama ke yang paling baru.
func (r *MongoMessageRepository) ListByTopic(ctx context.Context, topicID primitive.ObjectID) ([]*gateway.Message, error) {
	filter := bson.M{"topic_id": topicID}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	messages := []*gateway.Message{}
	for cursor.Next(ctx) {
		var m gateway.Message
		if err := cursor.Decode(&m); err != nil {
			return nil, err
		}
		messages = append(messages, &m)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return messages, nil
}

// FindFinalByTopic membaca pesan yang ditandai SELESAI (is_final) dalam satu topik.
// Kalau belum ada, hasilnya nil dan error-nya juga nil.
func (r *MongoMessageRepository) FindFinalByTopic(ctx context.Context, topicID primitive.ObjectID) (*gateway.Message, error) {
	filter := bson.M{"topic_id": topicID, "is_final": true}
	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})

	var m gateway.Message
	err := r.collection.FindOne(ctx, filter, opts).Decode(&m)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// topicRow adalah bentuk mentah satu baris hasil agregasi ListTopics.
// _id di sini berisi topic_id, jadi harus dipindah ke gateway.Topic.
type topicRow struct {
	TopicID   primitive.ObjectID `bson:"_id"`
	Content   string             `bson:"content"`
	Sender    string             `bson:"sender"`
	CreatedAt time.Time          `bson:"created_at"`
}

// ListTopics membaca daftar topik terbaru: satu baris per topik,
// berisi pesan pertama topik itu, diurutkan dari yang paling baru.
// Maksimal 20 topik, supaya jawaban tidak terlalu besar.
func (r *MongoMessageRepository) ListTopics(ctx context.Context) ([]gateway.Topic, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: 1}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$topic_id"},
			{Key: "content", Value: bson.D{{Key: "$first", Value: "$content"}}},
			{Key: "sender", Value: bson.D{{Key: "$first", Value: "$sender"}}},
			{Key: "created_at", Value: bson.D{{Key: "$first", Value: "$created_at"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: -1}}}},
		{{Key: "$limit", Value: 20}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	topics := []gateway.Topic{}
	for cursor.Next(ctx) {
		var row topicRow
		if err := cursor.Decode(&row); err != nil {
			return nil, err
		}
		topics = append(topics, gateway.Topic{
			TopicID:   row.TopicID,
			Content:   row.Content,
			Sender:    row.Sender,
			CreatedAt: row.CreatedAt,
		})
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return topics, nil
}
