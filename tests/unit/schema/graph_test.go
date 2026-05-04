package schema_test

import (
	"context"
	"encoding/json"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aescanero/dago/ent/enttest"
	"github.com/aescanero/dago/ent/graph"
)

const sqliteDSN = "file:ent?mode=memory&cache=shared&_fk=1"

func TestGraphCreate(t *testing.T) {
	client := enttest.Open(t, "sqlite3", sqliteDSN)
	defer client.Close()

	ctx := context.Background()
	def := json.RawMessage(`{"nodes":[],"edges":[]}`)

	g, err := client.Graph.Create().
		SetName("test-graph").
		SetVersion("1.0.0").
		SetEntryNode("start").
		SetDefinition(def).
		Save(ctx)

	require.NoError(t, err)
	assert.NotNil(t, g.ID)
	assert.Equal(t, "test-graph", g.Name)
	assert.Equal(t, "1.0.0", g.Version)
	assert.Equal(t, "start", g.EntryNode)
	assert.Equal(t, graph.StatusDraft, g.Status)
	assert.False(t, g.CreatedAt.IsZero())
	assert.False(t, g.UpdatedAt.IsZero())
}

func TestGraphVersionValidation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", sqliteDSN)
	defer client.Close()

	ctx := context.Background()
	def := json.RawMessage(`{}`)

	_, err := client.Graph.Create().
		SetName("bad-version-graph").
		SetVersion("abc").
		SetEntryNode("start").
		SetDefinition(def).
		Save(ctx)

	require.Error(t, err, "expected validation error for non-semver version")
	assert.Contains(t, err.Error(), "version")
}

func TestGraphStatusDefault(t *testing.T) {
	client := enttest.Open(t, "sqlite3", sqliteDSN)
	defer client.Close()

	ctx := context.Background()
	def := json.RawMessage(`{}`)

	g, err := client.Graph.Create().
		SetName("default-status-graph").
		SetVersion("0.1.0").
		SetEntryNode("node1").
		SetDefinition(def).
		Save(ctx)

	require.NoError(t, err)
	assert.Equal(t, graph.StatusDraft, g.Status)
}

func TestGraphUniqueNameVersion(t *testing.T) {
	client := enttest.Open(t, "sqlite3", sqliteDSN)
	defer client.Close()

	ctx := context.Background()
	def := json.RawMessage(`{}`)

	_, err := client.Graph.Create().
		SetName("unique-graph").
		SetVersion("1.0.0").
		SetEntryNode("start").
		SetDefinition(def).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Graph.Create().
		SetName("unique-graph").
		SetVersion("1.0.0").
		SetEntryNode("start").
		SetDefinition(def).
		Save(ctx)

	require.Error(t, err, "expected unique constraint error on (name, version)")
}
