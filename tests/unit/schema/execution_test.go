package schema_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aescanero/dago/ent/enttest"
	"github.com/aescanero/dago/ent/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func TestExecutionCreate(t *testing.T) {
	client := enttest.Open(t, "sqlite3", sqliteDSN)
	defer client.Close()

	ctx := context.Background()
	def := json.RawMessage(`{}`)

	g, err := client.Graph.Create().
		SetName("exec-test-graph").
		SetVersion("1.0.0").
		SetEntryNode("start").
		SetDefinition(def).
		Save(ctx)
	require.NoError(t, err)

	e, err := client.Execution.Create().
		SetGraph(g).
		Save(ctx)

	require.NoError(t, err)
	assert.NotNil(t, e.ID)
	assert.Equal(t, execution.StatusPending, e.Status)
	assert.False(t, e.CreatedAt.IsZero())
	assert.False(t, e.UpdatedAt.IsZero())
}

func TestExecutionStatusDefault(t *testing.T) {
	client := enttest.Open(t, "sqlite3", sqliteDSN)
	defer client.Close()

	ctx := context.Background()
	def := json.RawMessage(`{}`)

	g, err := client.Graph.Create().
		SetName("exec-status-graph").
		SetVersion("1.0.0").
		SetEntryNode("start").
		SetDefinition(def).
		Save(ctx)
	require.NoError(t, err)

	e, err := client.Execution.Create().
		SetGraph(g).
		Save(ctx)

	require.NoError(t, err)
	assert.Equal(t, execution.StatusPending, e.Status)
}

func TestExecutionJSONDefaults(t *testing.T) {
	client := enttest.Open(t, "sqlite3", sqliteDSN)
	defer client.Close()

	ctx := context.Background()
	def := json.RawMessage(`{}`)

	g, err := client.Graph.Create().
		SetName("exec-json-graph").
		SetVersion("1.0.0").
		SetEntryNode("start").
		SetDefinition(def).
		Save(ctx)
	require.NoError(t, err)

	e, err := client.Execution.Create().
		SetGraph(g).
		Save(ctx)

	require.NoError(t, err)
	assert.Equal(t, json.RawMessage(`{}`), e.Variables)
	assert.Equal(t, json.RawMessage(`[]`), e.Messages)
	assert.Equal(t, json.RawMessage(`{}`), e.NodeResults)
}

func TestExecutionOptionalFields(t *testing.T) {
	client := enttest.Open(t, "sqlite3", sqliteDSN)
	defer client.Close()

	ctx := context.Background()
	def := json.RawMessage(`{}`)

	g, err := client.Graph.Create().
		SetName("exec-optional-graph").
		SetVersion("1.0.0").
		SetEntryNode("start").
		SetDefinition(def).
		Save(ctx)
	require.NoError(t, err)

	e, err := client.Execution.Create().
		SetGraph(g).
		Save(ctx)

	require.NoError(t, err)
	assert.Nil(t, e.StartedAt)
	assert.Nil(t, e.CompletedAt)
	assert.Nil(t, e.CurrentNode)
	assert.Nil(t, e.Error)
}
