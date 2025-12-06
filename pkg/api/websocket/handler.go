package websocket

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aescanero/dago-libs/pkg/domain"
	"github.com/aescanero/dago-libs/pkg/ports"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for MVP
	},
}

// Handler handles WebSocket connections
type Handler struct {
	eventBus ports.EventBus
	logger   *zap.Logger
}

// NewHandler creates a new WebSocket handler
func NewHandler(eventBus ports.EventBus, logger *zap.Logger) *Handler {
	return &Handler{
		eventBus: eventBus,
		logger:   logger,
	}
}

// HandleGraphStream handles WebSocket streaming for a specific graph
func (h *Handler) HandleGraphStream(c *gin.Context) {
	graphID := c.Param("id")

	// Upgrade connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("failed to upgrade connection", zap.Error(err))
		return
	}
	defer func() { _ = conn.Close() }()

	h.logger.Info("WebSocket connection established",
		zap.String("graph_id", graphID),
		zap.String("client", c.ClientIP()))

	// Subscribe to events for this graph
	eventChan := make(chan *domain.Event, 10)
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Subscribe to all event types (simplified for MVP)
	go h.subscribeToEvents(ctx, eventChan)

	// Send events to client
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-eventChan:
			if event == nil {
				continue
			}

			// Only send events for this graph
			if event.GraphID != graphID {
				continue
			}

			// Send event to client
			data, err := json.Marshal(event)
			if err != nil {
				h.logger.Error("failed to marshal event", zap.Error(err))
				continue
			}

			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				h.logger.Error("failed to write message", zap.Error(err))
				return
			}
		}
	}
}

// subscribeToEvents subscribes to all event types
func (h *Handler) subscribeToEvents(ctx context.Context, ch chan<- *domain.Event) {
	// Create event handler that converts ports.Event to domain.Event
	eventHandler := func(ctx context.Context, event ports.Event) error {
		// Convert ports.Event to domain.Event
		domainEvent := &domain.Event{
			ID:        event.ID,
			Type:      domain.EventType(event.Type),
			GraphID:   event.ExecutionID,
			Timestamp: event.Timestamp,
			Data:      event.Data,
		}

		// Send to channel (non-blocking)
		select {
		case ch <- domainEvent:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Channel full, skip event
			h.logger.Warn("event channel full, dropping event",
				zap.String("event_id", event.ID),
				zap.String("event_type", string(event.Type)))
		}
		return nil
	}

	// Subscribe to graph and node events
	topics := []string{"graph.events", "node.events"}
	for _, topic := range topics {
		if err := h.eventBus.Subscribe(ctx, topic, eventHandler); err != nil {
			h.logger.Error("failed to subscribe to events",
				zap.String("topic", topic),
				zap.Error(err))
		}
	}
}
