package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Execution holds the schema definition for the Execution entity.
type Execution struct{ ent.Schema }

// Fields of the Execution.
func (Execution) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.Enum("status").Values(
			"pending", "running", "completed",
			"failed", "cancelled", "interrupted",
		).Default("pending"),
		field.String("current_node").MaxLen(255).Optional().Nillable(),
		field.JSON("variables", json.RawMessage{}).
			Default(json.RawMessage("{}")).
			SchemaType(map[string]string{"postgres": "jsonb"}),
		field.JSON("messages", json.RawMessage{}).
			Default(json.RawMessage("[]")).
			SchemaType(map[string]string{"postgres": "jsonb"}),
		field.JSON("node_results", json.RawMessage{}).
			Default(json.RawMessage("{}")).
			SchemaType(map[string]string{"postgres": "jsonb"}),
		field.String("error").Optional().Nillable(),
		field.Time("started_at").Optional().Nillable(),
		field.Time("completed_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Execution.
func (Execution) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("graph", Graph.Type).Ref("executions").Unique().Required(),
	}
}

// Indexes of the Execution.
func (Execution) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status").Edges("graph"),
		index.Fields("status"),
		index.Fields("created_at"),
	}
}
