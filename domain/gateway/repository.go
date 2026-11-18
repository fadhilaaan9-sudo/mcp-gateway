package gateway

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MessageRepository adalah janji: apapun yang menyimpan Message,
// harus bisa 5 hal ini. MongoMessageRepository di infrastructure/db
// sudah memenuhi janji ini, jadi handler tidak perlu tahu MongoDB.
type MessageRepository interface {
	EnsureIndexes(ctx context.Context) error
	Save(ctx context.Context, m *Message) error
	ListByTopic(ctx context.Context, topicID primitive.ObjectID) ([]*Message, error)
	FindFinalByTopic(ctx context.Context, topicID primitive.ObjectID) (*Message, error)
	ListTopics(ctx context.Context) ([]Topic, error)
}
