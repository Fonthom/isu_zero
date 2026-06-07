package pubsub

import (
	"context"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	StreamName = "ISU_ZERO"
	StreamDesc = "ISU-Zero event stream"
)

// Subjects published on this stream
const (
	SubjectNavGoalRequested  = "nav.goal.requested"
	SubjectNavGoalCompleted  = "nav.goal.completed"
	SubjectPhotoCaptured     = "photo.captured"
)

type Bus struct {
	nc *nats.Conn
	js jetstream.JetStream
}

func New(nc *nats.Conn) (*Bus, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}

	_, err = js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:        StreamName,
		Description: StreamDesc,
		Subjects: []string{
			SubjectNavGoalRequested,
			SubjectNavGoalCompleted,
			SubjectPhotoCaptured,
		},
	})
	if err != nil {
		return nil, err
	}

	return &Bus{nc: nc, js: js}, nil
}

func (b *Bus) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := b.js.Publish(ctx, subject, data)
	return err
}

func (b *Bus) Subscribe(ctx context.Context, subject, durableName string, handler func([]byte)) error {
	cons, err := b.js.CreateOrUpdateConsumer(ctx, StreamName, jetstream.ConsumerConfig{
		Durable:        durableName,
		FilterSubject:  subject,
		DeliverPolicy:  jetstream.DeliverAllPolicy,
		AckPolicy:      jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return err
	}

	_, err = cons.Consume(func(msg jetstream.Msg) {
		handler(msg.Data())
		if err := msg.Ack(); err != nil {
			log.Printf("failed to ack message on %s: %v", subject, err)
		}
	})
	return err
}