package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Author holds the schema definition for the Author entity.
type Author struct {
	ent.Schema
}

// Fields of the Author.
func (Author) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id"),
		field.String("first_name"),
		field.String("middle_name").Optional(),
		field.String("last_name"),
		field.Time("created_at").Optional().StructTag(`json:"-"`),
		field.Time("updated_at").Optional().StructTag(`json:"-"`),
		field.Time("deleted_at").Optional().Nillable().StructTag(`json:"-"`),
	}
}

// Edges of the Author.
func (Author) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("books", Book.Type).Ref("authors"),
	}
}
