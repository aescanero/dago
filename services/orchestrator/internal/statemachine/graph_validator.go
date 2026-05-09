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

	for _, e := range g.Edges {
		if e.Type != "sequential" {
			return fmt.Errorf("%w: unsupported edge type: %s", domain.ErrGraphValidation, e.Type)
		}
	}

	// Build a DAG using dominikbraun/graph to verify reachability.
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

	// BFS/DFS from entry_node; every node must be reachable.
	adjacency, err := dag.AdjacencyMap()
	if err != nil {
		return fmt.Errorf("%w: build adjacency map: %s", domain.ErrGraphValidation, err.Error())
	}
	reachable := make(map[string]bool)
	var visit func(n string)
	visit = func(n string) {
		if reachable[n] {
			return
		}
		reachable[n] = true
		for neighbor := range adjacency[n] {
			visit(neighbor)
		}
	}
	visit(g.EntryNode)

	for key := range g.Nodes {
		if !reachable[key] {
			return fmt.Errorf("%w: node %q is not reachable from entry_node %q",
				domain.ErrGraphValidation, key, g.EntryNode)
		}
	}
	return nil
}
