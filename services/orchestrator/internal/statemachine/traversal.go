package statemachine

import "github.com/aescanero/dago/libs/domain"

// NextNode returns the key of the next sequential node after currentNode,
// or ("", nil) if currentNode is terminal (no outgoing sequential edge).
func NextNode(g domain.GraphDefinition, currentNode string) (string, error) {
	for _, e := range g.Edges {
		if e.From == currentNode && e.Type == "sequential" {
			return e.To, nil
		}
	}
	return "", nil
}
