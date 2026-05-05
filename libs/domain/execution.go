package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ExecutionStatus represents the status of an execution.
type ExecutionStatus string

// ExecutionStatus enum values.
const (
	ExecutionStatusPending     ExecutionStatus = "pending"
	ExecutionStatusRunning     ExecutionStatus = "running"
	ExecutionStatusCompleted   ExecutionStatus = "completed"
	ExecutionStatusFailed      ExecutionStatus = "failed"
	ExecutionStatusCancelled   ExecutionStatus = "cancelled"
	ExecutionStatusInterrupted ExecutionStatus = "interrupted"
)

// Execution is an execution instance of a graph.
type Execution struct {
	ID          uuid.UUID
	GraphID     uuid.UUID
	Status      ExecutionStatus
	CurrentNode string
	Variables   json.RawMessage
	Messages    json.RawMessage
	NodeResults json.RawMessage
	Error       string
	StartedAt   *time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
