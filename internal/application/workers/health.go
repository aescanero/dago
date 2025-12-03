package workers

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// HealthMonitor monitors worker health
type HealthMonitor struct {
	pool     *Pool
	interval time.Duration
	logger   *zap.Logger

	mu      sync.RWMutex
	running bool
	stopCh  chan struct{}
}

// HealthStatus represents the health status of the worker pool
type HealthStatus struct {
	TotalWorkers  int
	IdleWorkers   int
	BusyWorkers   int
	StoppedWorkers int
	Healthy       bool
	Timestamp     time.Time
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(pool *Pool, interval time.Duration, logger *zap.Logger) *HealthMonitor {
	return &HealthMonitor{
		pool:     pool,
		interval: interval,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}
}

// Start starts the health monitor
func (h *HealthMonitor) Start() {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.mu.Unlock()

	go h.run()
}

// Stop stops the health monitor
func (h *HealthMonitor) Stop() {
	h.mu.Lock()
	if !h.running {
		h.mu.Unlock()
		return
	}
	h.running = false
	h.mu.Unlock()

	close(h.stopCh)
}

// run is the main health monitoring loop
func (h *HealthMonitor) run() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.checkHealth()
		}
	}
}

// checkHealth checks worker health and logs status
func (h *HealthMonitor) checkHealth() {
	status := h.GetStatus()

	h.logger.Info("worker pool health check",
		zap.Int("total", status.TotalWorkers),
		zap.Int("idle", status.IdleWorkers),
		zap.Int("busy", status.BusyWorkers),
		zap.Int("stopped", status.StoppedWorkers),
		zap.Bool("healthy", status.Healthy))

	// Record metrics
	h.pool.metrics.RecordWorkerPoolStatus(
		status.IdleWorkers,
		status.BusyWorkers,
		status.StoppedWorkers,
	)

	// Warn if pool is unhealthy
	if !status.Healthy {
		h.logger.Warn("worker pool is unhealthy",
			zap.Int("idle", status.IdleWorkers),
			zap.Int("total", status.TotalWorkers))
	}

	// Warn if all workers are busy
	if status.BusyWorkers == status.TotalWorkers {
		h.logger.Warn("all workers are busy - consider scaling up",
			zap.Int("total", status.TotalWorkers))
	}
}

// GetStatus returns the current health status
func (h *HealthMonitor) GetStatus() *HealthStatus {
	workerStatuses := h.pool.GetStatus()

	var idle, busy, stopped int
	for _, status := range workerStatuses {
		switch status {
		case WorkerStatusIdle:
			idle++
		case WorkerStatusBusy:
			busy++
		case WorkerStatusStopped:
			stopped++
		}
	}

	total := len(workerStatuses)
	healthy := idle > 0 && stopped == 0

	return &HealthStatus{
		TotalWorkers:   total,
		IdleWorkers:    idle,
		BusyWorkers:    busy,
		StoppedWorkers: stopped,
		Healthy:        healthy,
		Timestamp:      time.Now(),
	}
}

// IsHealthy returns true if the worker pool is healthy
func (h *HealthMonitor) IsHealthy() bool {
	status := h.GetStatus()
	return status.Healthy
}
