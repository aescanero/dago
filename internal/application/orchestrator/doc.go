// Package orchestrator implements the core orchestration logic for graph execution.
//
// The orchestrator manager coordinates graph execution by:
//   - Validating graph structure and dependencies
//   - Managing execution lifecycle (submit, monitor, cancel)
//   - Publishing events to the event bus
//   - Tracking execution state via state storage
//
// The validator ensures graphs are well-formed with no cycles and valid dependencies.
package orchestrator
