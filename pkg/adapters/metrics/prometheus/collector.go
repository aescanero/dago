package prometheus

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector implements MetricsCollector using Prometheus
type Collector struct {
	graphsSubmitted   *prometheus.CounterVec
	graphsCompleted   *prometheus.CounterVec
	nodesExecuted     *prometheus.CounterVec
	nodeExecutionTime prometheus.Histogram
	workerPoolIdle    prometheus.Gauge
	workerPoolBusy    prometheus.Gauge
	workerPoolStopped prometheus.Gauge

	// Additional metrics
	graphsFailed      *prometheus.CounterVec
	nodesFailed       *prometheus.CounterVec
	toolExecutions    *prometheus.CounterVec
	toolFailures      *prometheus.CounterVec
	llmCalls          *prometheus.CounterVec
	llmTokens         *prometheus.CounterVec
	workerCount       *prometheus.GaugeVec
	queueDepth        *prometheus.GaugeVec
	activeExecutions  prometheus.Gauge
	graphDuration     *prometheus.HistogramVec
	nodeDuration      *prometheus.HistogramVec
	toolDuration      *prometheus.HistogramVec
	llmLatency        *prometheus.HistogramVec
	queueWaitTime     *prometheus.HistogramVec
}

// NewCollector creates a new Prometheus metrics collector
func NewCollector() *Collector {
	return &Collector{
		graphsSubmitted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dago_graphs_submitted_total",
				Help: "Total number of graphs submitted",
			},
			[]string{"status"},
		),
		graphsCompleted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dago_graphs_completed_total",
				Help: "Total number of graphs completed",
			},
			[]string{"status"},
		),
		graphsFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dago_graphs_failed_total",
				Help: "Total number of graphs failed",
			},
			[]string{},
		),
		nodesExecuted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dago_nodes_executed_total",
				Help: "Total number of nodes executed",
			},
			[]string{"node_type", "status"},
		),
		nodesFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dago_nodes_failed_total",
				Help: "Total number of nodes failed",
			},
			[]string{"node_type"},
		),
		nodeExecutionTime: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "dago_node_execution_duration_seconds",
				Help:    "Node execution duration in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
			},
		),
		toolExecutions: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dago_tool_executions_total",
				Help: "Total number of tool executions",
			},
			[]string{"tool"},
		),
		toolFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dago_tool_failures_total",
				Help: "Total number of tool failures",
			},
			[]string{"tool"},
		),
		llmCalls: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dago_llm_calls_total",
				Help: "Total number of LLM API calls",
			},
			[]string{"model"},
		),
		llmTokens: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dago_llm_tokens_total",
				Help: "Total number of LLM tokens used",
			},
			[]string{"model", "type"},
		),
		workerCount: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "dago_worker_count",
				Help: "Current number of workers by type",
			},
			[]string{"node_type"},
		),
		queueDepth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "dago_queue_depth",
				Help: "Current depth of execution queues",
			},
			[]string{"queue"},
		),
		activeExecutions: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "dago_active_executions",
				Help: "Number of currently active executions",
			},
		),
		workerPoolIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "dago_worker_pool_idle",
				Help: "Number of idle workers",
			},
		),
		workerPoolBusy: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "dago_worker_pool_busy",
				Help: "Number of busy workers",
			},
		),
		workerPoolStopped: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "dago_worker_pool_stopped",
				Help: "Number of stopped workers",
			},
		),
		graphDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "dago_graph_duration_seconds",
				Help:    "Graph execution duration in seconds",
				Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
			},
			[]string{},
		),
		nodeDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "dago_node_duration_seconds",
				Help:    "Node execution duration in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
			},
			[]string{"node_type"},
		),
		toolDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "dago_tool_duration_seconds",
				Help:    "Tool execution duration in seconds",
				Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
			},
			[]string{"tool"},
		),
		llmLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "dago_llm_latency_seconds",
				Help:    "LLM API call latency in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 20},
			},
			[]string{"model"},
		),
		queueWaitTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "dago_queue_wait_time_seconds",
				Help:    "Time spent waiting in queue",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
			},
			[]string{},
		),
	}
}

// IncGraphsSubmitted increments the count of submitted graphs
func (c *Collector) IncGraphsSubmitted(labels map[string]string) {
	c.graphsSubmitted.With(prometheus.Labels(labels)).Inc()
}

// IncGraphsCompleted increments the count of completed graphs
func (c *Collector) IncGraphsCompleted(labels map[string]string) {
	c.graphsCompleted.With(prometheus.Labels(labels)).Inc()
}

// IncGraphsFailed increments the count of failed graphs
func (c *Collector) IncGraphsFailed(labels map[string]string) {
	c.graphsFailed.With(prometheus.Labels{}).Inc()
}

// IncNodesExecuted increments the count of executed nodes
func (c *Collector) IncNodesExecuted(nodeType string, labels map[string]string) {
	mergedLabels := prometheus.Labels{"node_type": nodeType}
	for k, v := range labels {
		mergedLabels[k] = v
	}
	c.nodesExecuted.With(mergedLabels).Inc()
}

// IncNodesFailed increments the count of failed nodes
func (c *Collector) IncNodesFailed(nodeType string, labels map[string]string) {
	c.nodesFailed.WithLabelValues(nodeType).Inc()
}

// IncToolExecutions increments the count of tool executions
func (c *Collector) IncToolExecutions(toolName string, labels map[string]string) {
	c.toolExecutions.WithLabelValues(toolName).Inc()
}

// IncToolFailures increments the count of tool failures
func (c *Collector) IncToolFailures(toolName string, labels map[string]string) {
	c.toolFailures.WithLabelValues(toolName).Inc()
}

// IncLLMCalls increments the count of LLM API calls
func (c *Collector) IncLLMCalls(model string, labels map[string]string) {
	c.llmCalls.WithLabelValues(model).Inc()
}

// IncLLMTokens increments the count of LLM tokens used
func (c *Collector) IncLLMTokens(model string, tokenType string, count int, labels map[string]string) {
	c.llmTokens.WithLabelValues(model, tokenType).Add(float64(count))
}

// SetWorkerCount sets the current number of workers for a node type
func (c *Collector) SetWorkerCount(nodeType string, count int) {
	c.workerCount.WithLabelValues(nodeType).Set(float64(count))
}

// SetQueueDepth sets the current depth of the execution queue
func (c *Collector) SetQueueDepth(queueName string, depth int) {
	c.queueDepth.WithLabelValues(queueName).Set(float64(depth))
}

// SetActiveExecutions sets the number of currently active executions
func (c *Collector) SetActiveExecutions(count int) {
	c.activeExecutions.Set(float64(count))
}

// ObserveGraphDuration records the duration of a graph execution
func (c *Collector) ObserveGraphDuration(duration time.Duration, labels map[string]string) {
	c.graphDuration.With(prometheus.Labels{}).Observe(duration.Seconds())
}

// ObserveNodeDuration records the duration of a node execution
func (c *Collector) ObserveNodeDuration(nodeType string, duration time.Duration, labels map[string]string) {
	c.nodeDuration.WithLabelValues(nodeType).Observe(duration.Seconds())
}

// ObserveToolDuration records the duration of a tool execution
func (c *Collector) ObserveToolDuration(toolName string, duration time.Duration, labels map[string]string) {
	c.toolDuration.WithLabelValues(toolName).Observe(duration.Seconds())
}

// ObserveLLMLatency records the latency of an LLM API call
func (c *Collector) ObserveLLMLatency(model string, duration time.Duration, labels map[string]string) {
	c.llmLatency.WithLabelValues(model).Observe(duration.Seconds())
}

// ObserveQueueWaitTime records how long an execution waited in the queue
func (c *Collector) ObserveQueueWaitTime(duration time.Duration, labels map[string]string) {
	c.queueWaitTime.With(prometheus.Labels{}).Observe(duration.Seconds())
}

// RecordGraphSubmitted records a graph submission (compatibility method)
func (c *Collector) RecordGraphSubmitted(status string) {
	c.graphsSubmitted.WithLabelValues(status).Inc()
}

// RecordGraphCompleted records a graph completion (compatibility method)
func (c *Collector) RecordGraphCompleted(status string, duration time.Duration) {
	c.graphsCompleted.WithLabelValues(status).Inc()
}

// RecordNodeExecuted records a node execution (compatibility method)
func (c *Collector) RecordNodeExecuted(status string, duration time.Duration) {
	c.nodesExecuted.WithLabelValues("unknown", status).Inc()
	c.nodeExecutionTime.Observe(duration.Seconds())
}

// RecordWorkerPoolStatus records worker pool status (compatibility method)
func (c *Collector) RecordWorkerPoolStatus(idle, busy, stopped int) {
	c.workerPoolIdle.Set(float64(idle))
	c.workerPoolBusy.Set(float64(busy))
	c.workerPoolStopped.Set(float64(stopped))
}
