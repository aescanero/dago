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

// Node holds the schema definition for the Node entity.
type Node struct{ ent.Schema }

// Fields of the Node.
func (Node) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("node_key").MaxLen(255).NotEmpty(),
		field.Enum("pattern").Values(
			"llm_call", "tool_use", "react", "reflection",
			"router", "guardrail", "subgraph",
		),
		field.JSON("config", json.RawMessage{}).
			SchemaType(map[string]string{"postgres": "jsonb"}),
		field.JSON("position", json.RawMessage{}).Optional().
			SchemaType(map[string]string{"postgres": "jsonb"}),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Node.
func (Node) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("graph", Graph.Type).Ref("nodes").Unique().Required(),
	}
}

// Indexes of the Node.
func (Node) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("node_key").Edges("graph").Unique(),
		index.Fields("pattern").Edges("graph"),
	}
}
