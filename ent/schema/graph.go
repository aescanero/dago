package schema

import (
	"encoding/json"
	"regexp"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Graph holds the schema definition for the Graph entity.
type Graph struct{ ent.Schema }

// Fields of the Graph.
func (Graph) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("name").MaxLen(255).NotEmpty(),
		field.String("version").MaxLen(20).
			Match(regexp.MustCompile(`^\d+\.\d+\.\d+$`)),
		field.String("description").Optional().Nillable(),
		field.String("entry_node").MaxLen(255).NotEmpty(),
		field.JSON("definition", json.RawMessage{}).
			SchemaType(map[string]string{"postgres": "jsonb"}),
		field.JSON("memory_config", json.RawMessage{}).Optional().
			SchemaType(map[string]string{"postgres": "jsonb"}),
		field.Enum("status").Values("draft", "active", "archived").Default("draft"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Graph.
func (Graph) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("nodes", Node.Type),
		edge.To("executions", Execution.Type),
	}
}

// Indexes of the Graph.
func (Graph) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name", "version").Unique(),
		index.Fields("status"),
	}
}
