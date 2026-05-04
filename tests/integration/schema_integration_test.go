//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/aescanero/dago/ent"
	"github.com/aescanero/dago/ent/execution"
	"github.com/aescanero/dago/ent/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq"
)

func openTestClient(t *testing.T) *ent.Client {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://dago:dago@localhost:5432/dago?sslmode=disable"
	}
	client, err := ent.Open(dialect.Postgres, dsn)
	require.NoError(t, err, "opening ent client")
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestGraphNodeExecutionPersistence(t *testing.T) {
	client := openTestClient(t)
	ctx := context.Background()

	def := json.RawMessage(`{"nodes":{"start":{},"end":{}},"edges":[]}`)

	g, err := client.Graph.Create().
		SetName("integration-graph").
		SetVersion("1.0.0").
		SetEntryNode("start").
		SetDefinition(def).
		Save(ctx)
	require.NoError(t, err)
	assert.Equal(t, "integration-graph", g.Name)
	assert.Equal(t, graph.StatusDraft, g.Status)

	cfg1 := json.RawMessage(`{"model":"claude-3-opus"}`)
	n1, err := client.Node.Create().
		SetNodeKey("start").
		SetPattern("llm_call").
		SetConfig(cfg1).
		SetGraph(g).
		Save(ctx)
	require.NoError(t, err)
	assert.Equal(t, "start", n1.NodeKey)

	cfg2 := json.RawMessage(`{"condition":"true"}`)
	n2, err := client.Node.Create().
		SetNodeKey("end").
		SetPattern("guardrail").
		SetConfig(cfg2).
		SetGraph(g).
		Save(ctx)
	require.NoError(t, err)
	assert.Equal(t, "end", n2.NodeKey)

	e, err := client.Execution.Create().
		SetGraph(g).
		Save(ctx)
	require.NoError(t, err)
	assert.Equal(t, execution.StatusPending, e.Status)

	loaded, err := client.Graph.Query().
		WithNodes().
		WithExecutions().
		Where(graph.ID(g.ID)).
		Only(ctx)
	require.NoError(t, err)
	assert.Len(t, loaded.Edges.Nodes, 2)
	assert.Len(t, loaded.Edges.Executions, 1)

	assert.Equal(t, json.RawMessage(`{}`), e.Variables)
	assert.Equal(t, json.RawMessage(`[]`), e.Messages)
	assert.Equal(t, json.RawMessage(`{}`), e.NodeResults)

	t.Cleanup(func() {
		client.Execution.DeleteOneID(e.ID).Exec(ctx) //nolint:errcheck
		client.Node.DeleteOneID(n1.ID).Exec(ctx)     //nolint:errcheck
		client.Node.DeleteOneID(n2.ID).Exec(ctx)     //nolint:errcheck
		client.Graph.DeleteOneID(g.ID).Exec(ctx)     //nolint:errcheck
	})
}

func TestExecutionStatusTransitions(t *testing.T) {
	client := openTestClient(t)
	ctx := context.Background()

	def := json.RawMessage(`{}`)
	g, err := client.Graph.Create().
		SetName("status-transition-graph").
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

	running, err := client.Execution.UpdateOneID(e.ID).
		SetStatus(execution.StatusRunning).
		Save(ctx)
	require.NoError(t, err)
	assert.Equal(t, execution.StatusRunning, running.Status)

	completed, err := client.Execution.UpdateOneID(e.ID).
		SetStatus(execution.StatusCompleted).
		Save(ctx)
	require.NoError(t, err)
	assert.Equal(t, execution.StatusCompleted, completed.Status)

	t.Cleanup(func() {
		client.Execution.DeleteOneID(e.ID).Exec(ctx)
		client.Graph.DeleteOneID(g.ID).Exec(ctx)
	})
}
