package statemachine

import "github.com/aescanero/dago/libs/domain"

// ValidateGraph checks structural validity of a GraphDefinition for execution.
// Only "sequential" edges are supported in this sprint.
// Returns an error wrapping domain.ErrGraphValidation on any structural problem.
func ValidateGraph(g domain.GraphDefinition) error {
	panic("not implemented")
}
