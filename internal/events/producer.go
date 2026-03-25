package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/dimerin1/cloudtalk-review-system/internal/model"
)

const TopicReviewEvents = "review-events"

// Producer publishes review events to Kafka.
type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  TopicReviewEvents,
			Balancer:               &kafka.LeastBytes{},
			BatchTimeout:           10 * time.Millisecond,
			AllowAutoTopicCreation: true,
		},
	}
}

// PublishReviewEvent sends a review lifecycle event to Kafka.
func (p *Producer) PublishReviewEvent(ctx context.Context, eventType string, reviewID, productID uuid.UUID, rating int) error {
	event := model.ReviewEvent{
		Type:      eventType,
		ReviewID:  reviewID,
		ProductID: productID,
		Rating:    rating,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(productID.String()),
		Value: data,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
