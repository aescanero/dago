package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// GraphStatus represents the lifecycle status of a graph.
type GraphStatus string

// GraphStatus enum values.
const (
	GraphStatusDraft    GraphStatus = "draft"
	GraphStatusActive   GraphStatus = "active"
	GraphStatusArchived GraphStatus = "archived"
)

// Graph is the definition of an execution graph (not an instance).
type Graph struct {
	ID           uuid.UUID
	Name         string
	Version      string
	Description  string
	EntryNode    string
	Definition   json.RawMessage
	MemoryConfig json.RawMessage
	Status       GraphStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsDraft returns true if the graph can be modified.
func (g *Graph) IsDraft() bool { return g.Status == GraphStatusDraft }

// IsActive returns true if the graph can start executions.
func (g *Graph) IsActive() bool { return g.Status == GraphStatusActive }
