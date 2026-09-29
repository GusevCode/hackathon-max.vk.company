package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/usecase/inspection"
	"github.com/nats-io/nats.go"
)

const (
	notificationSubject = "max.notifications.v1"
	inspectionSubject   = "max.inspections.requested.v1"
	inspectionStream    = "MAX_INSPECTIONS"
	inspectionConsumer  = "evidence-analyzer"
)

type NATS struct {
	conn      *nats.Conn
	jetStream nats.JetStreamContext
}

func Connect(url string) (*NATS, error) {
	conn, err := nats.Connect(url,
		nats.Name("max-hackathon-control"),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(nats.DefaultReconnectWait),
		// A notification that falls back to direct delivery must not remain in
		// the reconnect buffer and be delivered a second time later.
		nats.ReconnectBufSize(0),
	)
	if err != nil {
		return nil, fmt.Errorf("connect NATS: %w", err)
	}
	jetStream, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("create NATS JetStream context: %w", err)
	}
	if _, err := jetStream.StreamInfo(inspectionStream); err != nil {
		if !errors.Is(err, nats.ErrStreamNotFound) {
			conn.Close()
			return nil, fmt.Errorf("inspect NATS stream: %w", err)
		}
		_, err = jetStream.AddStream(&nats.StreamConfig{
			Name: inspectionStream, Subjects: []string{inspectionSubject},
			Retention: nats.WorkQueuePolicy, Storage: nats.FileStorage, MaxAge: 24 * time.Hour,
		})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("create NATS inspection stream: %w", err)
		}
	}
	return &NATS{conn: conn, jetStream: jetStream}, nil
}

func (b *NATS) Publish(ctx context.Context, notification domain.Notification) error {
	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("encode notification: %w", err)
	}
	if err := b.conn.Publish(notificationSubject, payload); err != nil {
		return fmt.Errorf("publish notification: %w", err)
	}
	flushCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := b.conn.FlushWithContext(flushCtx); err != nil {
		return fmt.Errorf("flush notification: %w", err)
	}
	return nil
}

func (b *NATS) Subscribe(ctx context.Context) (<-chan domain.Notification, error) {
	out := make(chan domain.Notification, 32)
	subscription, err := b.conn.Subscribe(notificationSubject, func(message *nats.Msg) {
		var notification domain.Notification
		if decodeErr := json.Unmarshal(message.Data, &notification); decodeErr != nil {
			return
		}
		select {
		case out <- notification:
		case <-ctx.Done():
		}
	})
	if err != nil {
		close(out)
		return nil, fmt.Errorf("subscribe notifications: %w", err)
	}
	go func() {
		<-ctx.Done()
		_ = subscription.Unsubscribe()
		close(out)
	}()
	return out, nil
}

func (b *NATS) PublishInspection(ctx context.Context, request domain.InspectionRequested) error {
	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode inspection request: %w", err)
	}
	if _, err := b.jetStream.Publish(inspectionSubject, payload, nats.Context(ctx)); err != nil {
		return fmt.Errorf("publish durable inspection request: %w", err)
	}
	return nil
}

func (b *NATS) SubscribeInspections(ctx context.Context) (<-chan inspection.Delivery, error) {
	out := make(chan inspection.Delivery, 32)
	subscription, err := b.jetStream.Subscribe(inspectionSubject, func(message *nats.Msg) {
		var request domain.InspectionRequested
		if decodeErr := json.Unmarshal(message.Data, &request); decodeErr != nil {
			_ = message.Term()
			return
		}
		select {
		case out <- inspection.Delivery{Request: request, Ack: func() error { return message.Ack() }}:
		case <-ctx.Done():
		}
	}, nats.Durable(inspectionConsumer), nats.BindStream(inspectionStream), nats.ManualAck(), nats.AckExplicit(), nats.AckWait(15*time.Minute), nats.MaxDeliver(3))
	if err != nil {
		return nil, fmt.Errorf("subscribe inspections: %w", err)
	}
	go func() {
		<-ctx.Done()
		_ = subscription.Drain()
	}()
	return out, nil
}

func (b *NATS) Close() error {
	if b == nil || b.conn == nil {
		return nil
	}
	if err := b.conn.Drain(); err != nil {
		b.conn.Close()
		return err
	}
	b.conn.Close()
	return nil
}
