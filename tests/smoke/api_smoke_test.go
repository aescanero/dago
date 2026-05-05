//go:build smoke

package smoke_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/aescanero/dago/tests/testutil/fakes"
	orchutil "github.com/aescanero/dago/services/orchestrator/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSmokeServer(t *testing.T) string {
	t.Helper()
	gr := fakes.NewInMemoryGraphRepository()
	er := fakes.NewInMemoryExecutionRepository()
	srv := orchutil.NewTestServer(gr, er)
	t.Cleanup(srv.Close)
	return srv.URL
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	return resp
}

func TestAPICRUDSmoke(t *testing.T) {
	base := newSmokeServer(t)

	// POST → create graph
	createResp := postJSON(t, base+"/api/v1/graphs", map[string]any{
		"name": "smoke-graph", "version": "1.0.0",
		"entry_node": "start", "definition": map[string]any{},
	})
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var created map[string]any
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	id := fmt.Sprint(created["id"])
	_, parseErr := uuid.Parse(id)
	require.NoError(t, parseErr)

	// GET list
	listResp, err := http.Get(base + "/api/v1/graphs")
	require.NoError(t, err)
	defer listResp.Body.Close()
	assert.Equal(t, http.StatusOK, listResp.StatusCode)
	var list map[string]any
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&list))
	items := list["items"].([]any)
	assert.Len(t, items, 1)

	// GET by id
	getResp, err := http.Get(base + "/api/v1/graphs/" + id)
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusOK, getResp.StatusCode)

	// PUT update
	updateReq, _ := http.NewRequest(http.MethodPut, base+"/api/v1/graphs/"+id, bytes.NewReader(func() []byte {
		b, _ := json.Marshal(map[string]any{
			"name": "smoke-graph", "version": "2.0.0",
			"entry_node": "start", "definition": map[string]any{},
		})
		return b
	}()))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, err := http.DefaultClient.Do(updateReq)
	require.NoError(t, err)
	defer updateResp.Body.Close()
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)

	// DELETE (archive)
	delReq, _ := http.NewRequest(http.MethodDelete, base+"/api/v1/graphs/"+id, nil)
	delResp, err := http.DefaultClient.Do(delReq)
	require.NoError(t, err)
	defer delResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)

	// GET by id → 404 after archive actually returns 200 (archived graph still exists)
	// but a fresh graph lookup returns the archived graph, not 404.
	// Verify it's archived.
	getAfterResp, err := http.Get(base + "/api/v1/graphs/" + id)
	require.NoError(t, err)
	defer getAfterResp.Body.Close()
	assert.Equal(t, http.StatusOK, getAfterResp.StatusCode)
	var afterBody map[string]any
	require.NoError(t, json.NewDecoder(getAfterResp.Body).Decode(&afterBody))
	assert.Equal(t, "archived", afterBody["status"])
}

func TestAPIStartExecutionSmoke(t *testing.T) {
	base := newSmokeServer(t)

	// Create a graph first.
	createResp := postJSON(t, base+"/api/v1/graphs", map[string]any{
		"name": "exec-graph", "version": "1.0.0",
		"entry_node": "start", "definition": map[string]any{},
	})
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var graph map[string]any
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&graph))

	// Start execution.
	execResp := postJSON(t, base+"/api/v1/executions", map[string]any{
		"graph_id": graph["id"],
		"variables": map[string]any{"input": "hello"},
	})
	defer execResp.Body.Close()
	assert.Equal(t, http.StatusCreated, execResp.StatusCode)
	var exec map[string]any
	require.NoError(t, json.NewDecoder(execResp.Body).Decode(&exec))
	assert.Equal(t, "pending", exec["status"])
	assert.Equal(t, graph["id"], exec["graph_id"])
}
