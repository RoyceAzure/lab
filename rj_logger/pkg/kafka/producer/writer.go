package producer

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type Writer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}
