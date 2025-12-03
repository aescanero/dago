// Package workers implements the worker pool for executing graph nodes.
//
// The worker pool manages a fixed number of goroutines that:
//   - Subscribe to node execution events from the event bus
//   - Execute nodes using the appropriate adapters (LLM, etc.)
//   - Update execution state in state storage
//   - Publish completion/failure events
//
// The health monitor tracks worker status and logs metrics.
package workers
