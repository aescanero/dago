//go:build contract

package contract_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	orchutil "github.com/aescanero/dago/services/orchestrator/testutil"
	"github.com/aescanero/dago/tests/testutil/fakes"
)

func newContractServer(t *testing.T) string {
	t.Helper()
	gr := fakes.NewInMemoryGraphRepository()
	er := fakes.NewInMemoryExecutionRepository()
	srv := orchutil.NewTestServer(gr, er)
	t.Cleanup(srv.Close)
	return srv.URL
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func assertRequiredGraphFields(t *testing.T, body map[string]any) {
	t.Helper()
	required := []string{"id", "name", "version", "entry_node", "definition", "status", "created_at", "updated_at"}
	for _, f := range required {
		assert.Contains(t, body, f, "missing required field %q in GraphResponse", f)
	}
}

func assertErrorResponseFields(t *testing.T, body map[string]any) {
	t.Helper()
	assert.Contains(t, body, "code", "missing 'code' in ErrorResponse")
	assert.Contains(t, body, "message", "missing 'message' in ErrorResponse")
}

func validGraphPayload() map[string]any {
	return map[string]any{
		"name": "contract-graph", "version": "1.0.0",
		"entry_node": "start", "definition": map[string]any{"nodes": []any{}, "edges": []any{}},
	}
}

func TestCreateGraphContract(t *testing.T) {
	base := newContractServer(t)
	resp, err := http.Post(base+"/api/v1/graphs", "application/json", bytes.NewReader(mustJSON(validGraphPayload())))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assertRequiredGraphFields(t, body)
	assert.Equal(t, "draft", body["status"])
	_, err = uuid.Parse(fmt.Sprint(body["id"]))
	assert.NoError(t, err, "id must be a valid UUID")
}

func TestCreateGraphValidationContract(t *testing.T) {
	base := newContractServer(t)
	payload := validGraphPayload()
	payload["version"] = "v1-invalid"
	resp, err := http.Post(base+"/api/v1/graphs", "application/json", bytes.NewReader(mustJSON(payload)))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assertErrorResponseFields(t, body)
	assert.Equal(t, "VALIDATION_ERROR", body["code"])
}

func TestListGraphsContract(t *testing.T) {
	base := newContractServer(t)
	resp, err := http.Get(base + "/api/v1/graphs")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Contains(t, body, "items", "missing 'items' in GraphListResponse")
	assert.Contains(t, body, "pagination", "missing 'pagination' in GraphListResponse")
}

func TestGetGraphContract(t *testing.T) {
	base := newContractServer(t)

	postResp, err := http.Post(base+"/api/v1/graphs", "application/json", bytes.NewReader(mustJSON(validGraphPayload())))
	require.NoError(t, err)
	defer postResp.Body.Close()
	require.Equal(t, http.StatusCreated, postResp.StatusCode)
	var created map[string]any
	require.NoError(t, json.NewDecoder(postResp.Body).Decode(&created))

	// GET existing
	resp, err := http.Get(base + "/api/v1/graphs/" + fmt.Sprint(created["id"]))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assertRequiredGraphFields(t, body)

	// GET non-existing
	resp404, err := http.Get(base + "/api/v1/graphs/" + uuid.New().String())
	require.NoError(t, err)
	defer resp404.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp404.StatusCode)
	var err404 map[string]any
	require.NoError(t, json.NewDecoder(resp404.Body).Decode(&err404))
	assertErrorResponseFields(t, err404)
}

func TestUpdateGraphContract(t *testing.T) {
	base := newContractServer(t)

	postResp, err := http.Post(base+"/api/v1/graphs", "application/json", bytes.NewReader(mustJSON(validGraphPayload())))
	require.NoError(t, err)
	defer postResp.Body.Close()
	var created map[string]any
	require.NoError(t, json.NewDecoder(postResp.Body).Decode(&created))

	update := validGraphPayload()
	update["name"] = "updated-graph"
	update["version"] = "2.0.0"
	req, _ := http.NewRequest(http.MethodPut, base+"/api/v1/graphs/"+fmt.Sprint(created["id"]), bytes.NewReader(mustJSON(update)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assertRequiredGraphFields(t, body)
}

func TestDeleteGraphContract(t *testing.T) {
	base := newContractServer(t)

	postResp, err := http.Post(base+"/api/v1/graphs", "application/json", bytes.NewReader(mustJSON(validGraphPayload())))
	require.NoError(t, err)
	defer postResp.Body.Close()
	var created map[string]any
	require.NoError(t, json.NewDecoder(postResp.Body).Decode(&created))

	req, _ := http.NewRequest(http.MethodDelete, base+"/api/v1/graphs/"+fmt.Sprint(created["id"]), nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestStartExecutionContract(t *testing.T) {
	base := newContractServer(t)

	postResp, err := http.Post(base+"/api/v1/graphs", "application/json", bytes.NewReader(mustJSON(validGraphPayload())))
	require.NoError(t, err)
	defer postResp.Body.Close()
	var graph map[string]any
	require.NoError(t, json.NewDecoder(postResp.Body).Decode(&graph))

	payload := map[string]any{"graph_id": graph["id"]}
	resp, err := http.Post(base+"/api/v1/executions", "application/json", bytes.NewReader(mustJSON(payload)))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	required := []string{"id", "graph_id", "status", "variables", "messages", "node_results", "created_at", "updated_at"}
	for _, f := range required {
		assert.Contains(t, body, f, "missing required field %q in ExecutionResponse", f)
	}
	assert.Equal(t, "pending", body["status"])
}

func TestStartExecutionGraphNotFoundContract(t *testing.T) {
	base := newContractServer(t)
	payload := map[string]any{"graph_id": uuid.New().String()}
	resp, err := http.Post(base+"/api/v1/executions", "application/json", bytes.NewReader(mustJSON(payload)))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assertErrorResponseFields(t, body)
}
