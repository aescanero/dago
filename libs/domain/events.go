package domain

import (
	"encoding/json"
	"time"
)

// EventAuth carries the propagated auth context extracted from the originating JWT (ADR-012).
type EventAuth struct {
	Sub     string   `json:"sub"`
	Scope   string   `json:"scope"`
	Tags    []string `json:"tags"`
	OrgUnit string   `json:"org_unit"`
}

// Event is a CloudEvents 1.0 envelope used for all Valkey Streams messages (ADR-011).
type Event struct {
	ID              string          `json:"id"`
	Type            string          `json:"type"`
	Source          string          `json:"source"`
	SpecVersion     string          `json:"specversion"`
	Time            time.Time       `json:"time"`
	DataContentType string          `json:"datacontenttype"`
	Auth            *EventAuth      `json:"auth,omitempty"`
	Data            json.RawMessage `json:"data"`
}

// Orchestration event type constants matching the AsyncAPI channel addresses.
const (
	EventTypeExecutionRequested = "dago.graph.execution.requested"
	EventTypeNodeStarted        = "dago.node.execution.started"
	EventTypeNodeCompleted      = "dago.node.execution.completed"
	EventTypeNodeFailed         = "dago.node.execution.failed"
	EventTypeExecutionCompleted = "dago.graph.execution.completed"
	EventTypeExecutionFailed    = "dago.graph.execution.failed"
	EventTypeDLQ                = "dago.dlq"

	// Executor node-level event streams (SPRINT-009).
	StreamNodeExecuteRequested = "node.execute.requested"
	StreamNodeExecuted         = "node.executed"
	StreamNodeExecuteFailed    = "node.execute.failed"

	EventTypeNodeExecuteRequested = "node.execute.requested"
	EventTypeNodeExecuted         = "node.executed"
	EventTypeNodeExecuteFailed    = "node.execute.failed"
)
