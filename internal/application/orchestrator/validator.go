package orchestrator

import (
	"fmt"

	"github.com/aescanero/dago-libs/pkg/domain"
	"github.com/aescanero/dago-libs/pkg/domain/graph"
)

// Validator validates graph structures
type Validator struct{}

// NewValidator creates a new graph validator
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates a graph structure
func (v *Validator) Validate(g *domain.Graph) error {
	if g == nil {
		return fmt.Errorf("graph is nil")
	}

	// Check basic fields
	if g.ID == "" {
		return fmt.Errorf("graph ID is required")
	}

	if g.Version == "" {
		return fmt.Errorf("graph version is required")
	}

	if len(g.Nodes) == 0 {
		return fmt.Errorf("graph must have at least one node")
	}

	// Validate nodes
	nodeIDs := make(map[string]bool)
	for nodeID, node := range g.Nodes {
		if err := v.validateNode(nodeID, node); err != nil {
			return fmt.Errorf("invalid node %s: %w", nodeID, err)
		}

		// Check for duplicate node IDs
		if nodeIDs[nodeID] {
			return fmt.Errorf("duplicate node ID: %s", nodeID)
		}
		nodeIDs[nodeID] = true
	}

	// Validate entry node exists
	if g.EntryNode != "" {
		if _, exists := g.Nodes[g.EntryNode]; !exists {
			return fmt.Errorf("entry node %s not found in graph", g.EntryNode)
		}
	}

	// Validate edges
	for _, edge := range g.Edges {
		if _, exists := g.Nodes[edge.From]; !exists {
			return fmt.Errorf("edge references non-existent source node: %s", edge.From)
		}
		if _, exists := g.Nodes[edge.To]; !exists {
			return fmt.Errorf("edge references non-existent target node: %s", edge.To)
		}
	}

	return nil
}

// validateNode validates a single node
func (v *Validator) validateNode(nodeID string, node graph.Node) error {
	if nodeID == "" {
		return fmt.Errorf("node ID is required")
	}

	if node == nil {
		return fmt.Errorf("node is nil")
	}

	// Validate using the node's own Validate method
	if err := node.Validate(); err != nil {
		return err
	}

	return nil
}
