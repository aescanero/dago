package statemachine

import (
	"fmt"

	"github.com/dominikbraun/graph"

	"github.com/aescanero/dago/libs/domain"
)

// ValidateGraph checks structural validity of a GraphDefinition for execution.
// Only "sequential" edges are supported; conditional/parallel/loop/interrupt → ErrGraphValidation.
func ValidateGraph(g domain.GraphDefinition) error {
	if _, ok := g.Nodes[g.EntryNode]; !ok {
		return fmt.Errorf("%w: entry_node %q not found in nodes", domain.ErrGraphValidation, g.EntryNode)
	}
	if err := validateEdgeTypes(g.Edges); err != nil {
		return err
	}
	return validateReachability(g)
}

func validateEdgeTypes(edges []domain.EdgeDefinition) error {
	for _, e := range edges {
		if e.Type != "sequential" {
			return fmt.Errorf("%w: unsupported edge type: %s", domain.ErrGraphValidation, e.Type)
		}
	}
	return nil
}

func validateReachability(g domain.GraphDefinition) error {
	dag := graph.New(graph.StringHash, graph.Directed())
	for key := range g.Nodes {
		if err := dag.AddVertex(key); err != nil {
			return fmt.Errorf("%w: add vertex %q: %s", domain.ErrGraphValidation, key, err.Error())
		}
	}
	for _, e := range g.Edges {
		if err := dag.AddEdge(e.From, e.To); err != nil {
			return fmt.Errorf("%w: add edge %s→%s: %s", domain.ErrGraphValidation, e.From, e.To, err.Error())
		}
	}

	adjacency, err := dag.AdjacencyMap()
	if err != nil {
		return fmt.Errorf("%w: build adjacency map: %s", domain.ErrGraphValidation, err.Error())
	}
	reachable := reachableFrom(g.EntryNode, adjacency)
	for key := range g.Nodes {
		if !reachable[key] {
			return fmt.Errorf("%w: node %q is not reachable from entry_node %q",
				domain.ErrGraphValidation, key, g.EntryNode)
		}
	}
	return nil
}

// reachableFrom returns the set of all nodes reachable from start via DFS.
func reachableFrom(start string, adjacency map[string]map[string]graph.Edge[string]) map[string]bool {
	visited := make(map[string]bool)
	var dfs func(n string)
	dfs = func(n string) {
		if visited[n] {
			return
		}
		visited[n] = true
		for neighbor := range adjacency[n] {
			dfs(neighbor)
		}
	}
	dfs(start)
	return visited
}
