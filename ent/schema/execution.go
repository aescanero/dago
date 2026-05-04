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

// jsonbField returns a JSON field with postgres jsonb schema type.
func jsonbField(name string, def json.RawMessage) ent.Field {
	return field.JSON(name, json.RawMessage{}).
		Default(def).
		SchemaType(map[string]string{"postgres": "jsonb"})
}

// Fields of the Execution.
func (Execution) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.Enum("status").Values(
			"pending", "running", "completed",
			"failed", "cancelled", "interrupted",
		).Default("pending"),
		field.String("current_node").MaxLen(255).Optional().Nillable(),
		jsonbField("variables", json.RawMessage("{}")),
		jsonbField("messages", json.RawMessage("[]")),
		jsonbField("node_results", json.RawMessage("{}")),
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
