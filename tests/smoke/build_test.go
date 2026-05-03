//go:build smoke

// Package smoke_test contains smoke tests that verify the monorepo build pipeline.
package smoke_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err, "git rev-parse --show-toplevel must succeed")
	return strings.TrimSpace(string(out))
}

func TestServicesBuildWithoutErrors(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	assert.NoError(t, err, "go build ./... must exit 0:\n%s", string(out))
}

func TestGoVetReportsNoProblems(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	assert.NoError(t, err, "go vet ./... must exit 0:\n%s", string(out))
}

func TestModulePathIsCorrect(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	cmd := exec.Command("go", "list", "-m")
	cmd.Dir = root
	out, err := cmd.Output()
	require.NoError(t, err, "go list -m must succeed")
	module := strings.TrimSpace(string(out))
	assert.Equal(t, "github.com/aescanero/dago", module)
}

func TestAllServicesHaveMainPackage(t *testing.T) {
	t.Parallel()
	services := []string{
		"orchestrator",
		"executor",
		"router",
		"planner",
		"auth-server",
		"catalog",
		"mcp-registry",
		"agent-registry",
	}
	root := repoRoot(t)
	for _, svc := range services {
		svc := svc
		t.Run(svc, func(t *testing.T) {
			t.Parallel()
			mainFile := filepath.Join(root, "services", svc, "cmd", "main.go")
			data, err := os.ReadFile(mainFile)
			require.NoError(t, err, "cmd/main.go must exist for service %s", svc)
			assert.Contains(t, string(data), "package main",
				"cmd/main.go for %s must declare package main", svc)
		})
	}
}

func TestDockerComposeIsValid(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "docker-compose.yml"))
	require.NoError(t, err, "docker-compose.yml must exist")

	var compose map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &compose), "docker-compose.yml must be valid YAML")

	svcs, ok := compose["services"].(map[string]interface{})
	require.True(t, ok, "docker-compose.yml must have a services key")
	assert.Contains(t, svcs, "postgres", "docker-compose must define postgres service")
	assert.Contains(t, svcs, "valkey", "docker-compose must define valkey service")
}
