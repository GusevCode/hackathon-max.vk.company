package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	"github.com/nats-io/nats.go"
)

const notificationSubject = "max.notifications.v1"

type NATS struct {
	conn *nats.Conn
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
	return &NATS{conn: conn}, nil
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
