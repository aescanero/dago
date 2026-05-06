package schema

import (
	"regexp"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

var orgPathRegex = regexp.MustCompile(`^(/[^/]+)*/?$`)

// OrgUnit holds the schema definition for the OrgUnit entity.
type OrgUnit struct{ ent.Schema }

// Fields of the OrgUnit.
func (OrgUnit) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("name").MaxLen(255).NotEmpty(),
		field.String("path").MaxLen(1024).Match(orgPathRegex).Unique(),
		field.Strings("tags").Default([]string{}),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the OrgUnit.
func (OrgUnit) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("children", OrgUnit.Type).From("parent").Unique(),
		edge.To("users", User.Type),
	}
}

// Indexes of the OrgUnit.
func (OrgUnit) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("path").Unique(),
	}
}
