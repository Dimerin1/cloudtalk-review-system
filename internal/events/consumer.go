package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/dimerin1/cloudtalk-review-system/internal/model"
)

// Consumer reads review events from Kafka and logs them.
type Consumer struct {
	reader *kafka.Reader
	logger *slog.Logger
}

func NewConsumer(brokers []string, groupID string, logger *slog.Logger) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          TopicReviewEvents,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			StartOffset:    kafka.FirstOffset,
			CommitInterval: time.Second,
		}),
		logger: logger,
	}
}

// Start begins consuming messages. Blocks until the context is cancelled.
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("consumer started, waiting for review events")

	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return c.reader.Close()
			}
			c.logger.Error("failed to read message", "error", err)
			continue
		}

		var event model.ReviewEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.Error("failed to unmarshal event", "error", err)
			continue
		}

		c.logger.Info("review event received",
			"type", event.Type,
			"review_id", event.ReviewID,
			"product_id", event.ProductID,
			"rating", event.Rating,
			"timestamp", event.Timestamp,
		)
	}
}
