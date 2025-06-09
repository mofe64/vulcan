package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type EventBus struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func NewEventBus(url string) (*EventBus, error) {
	// connect; “nats://user:pass@host:4222” works too if auth enabled
	nc, err := nats.Connect(url,
		nats.Name("vulkan-api"),
		nats.MaxReconnects(-1), // infinite reconnect
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, err
	}

	// get JetStream context (creates it if not enabled)
	js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		return nil, err
	}

	return &EventBus{js: js, nc: nc}, nil
}

func (e *EventBus) OrgCreated(ctx context.Context, orgID uuid.UUID) {
	payload, _ := json.Marshal(map[string]any{
		"event":  "org.created",
		"org_id": orgID.String(),
		"ts":     time.Now().UTC().Format(time.RFC3339),
	})
	// fire-and-forget: async publish, drop if buffer full
	_, _ = e.js.PublishAsync("org.created", payload)
}

func (e *EventBus) Close() {
	e.nc.Drain() // flush buffered async publishes
}

//Consuming the event (optional audit sink example)
// nc, _ := nats.Connect("nats://nats.cp.svc:4222")
// js, _ := nc.JetStream()

// js.Subscribe("org.created", func(msg *nats.Msg) {
//     // write to Loki or send e-mail
// }, nats.Durable("audit-sink"))
