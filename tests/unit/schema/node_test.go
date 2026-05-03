package schema_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aescanero/dago/ent/enttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func TestNodeCreate(t *testing.T) {
	client := enttest.Open(t, "sqlite3", sqliteDSN)
	defer client.Close()

	ctx := context.Background()
	def := json.RawMessage(`{}`)

	g, err := client.Graph.Create().
		SetName("node-test-graph").
		SetVersion("1.0.0").
		SetEntryNode("classifier").
		SetDefinition(def).
		Save(ctx)
	require.NoError(t, err)

	cfg := json.RawMessage(`{"model":"claude-3-opus"}`)
	n, err := client.Node.Create().
		SetNodeKey("classifier").
		SetPattern("llm_call").
		SetConfig(cfg).
		SetGraph(g).
		Save(ctx)

	require.NoError(t, err)
	assert.NotNil(t, n.ID)
	assert.Equal(t, "classifier", n.NodeKey)
	assert.Equal(t, "llm_call", string(n.Pattern))
	assert.False(t, n.CreatedAt.IsZero())
}

func TestNodePatternValidation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", sqliteDSN)
	defer client.Close()

	ctx := context.Background()
	def := json.RawMessage(`{}`)

	g, err := client.Graph.Create().
		SetName("node-pattern-graph").
		SetVersion("1.0.0").
		SetEntryNode("start").
		SetDefinition(def).
		Save(ctx)
	require.NoError(t, err)

	cfg := json.RawMessage(`{}`)
	_, err = client.Node.Create().
		SetNodeKey("bad-node").
		SetPattern("invalid_pattern").
		SetConfig(cfg).
		SetGraph(g).
		Save(ctx)

	require.Error(t, err, "expected validation error for invalid pattern")
}

func TestNodeUniqueKeyPerGraph(t *testing.T) {
	client := enttest.Open(t, "sqlite3", sqliteDSN)
	defer client.Close()

	ctx := context.Background()
	def := json.RawMessage(`{}`)
	cfg := json.RawMessage(`{}`)

	g, err := client.Graph.Create().
		SetName("unique-node-graph").
		SetVersion("1.0.0").
		SetEntryNode("step1").
		SetDefinition(def).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Node.Create().
		SetNodeKey("step1").
		SetPattern("llm_call").
		SetConfig(cfg).
		SetGraph(g).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Node.Create().
		SetNodeKey("step1").
		SetPattern("tool_use").
		SetConfig(cfg).
		SetGraph(g).
		Save(ctx)

	require.Error(t, err, "expected unique constraint on (graph_id, node_key)")
}

func TestNodeBelongsToGraph(t *testing.T) {
	client := enttest.Open(t, "sqlite3", sqliteDSN)
	defer client.Close()

	ctx := context.Background()
	def := json.RawMessage(`{}`)
	cfg := json.RawMessage(`{}`)

	g, err := client.Graph.Create().
		SetName("belongs-to-graph").
		SetVersion("1.0.0").
		SetEntryNode("analyzer").
		SetDefinition(def).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Node.Create().
		SetNodeKey("analyzer").
		SetPattern("react").
		SetConfig(cfg).
		SetGraph(g).
		Save(ctx)
	require.NoError(t, err)

	loaded, err := client.Node.Query().
		WithGraph().
		First(ctx)
	require.NoError(t, err)

	require.NotNil(t, loaded.Edges.Graph)
	assert.Equal(t, g.ID, loaded.Edges.Graph.ID)
	assert.Equal(t, "belongs-to-graph", loaded.Edges.Graph.Name)
}
