package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// metrics is a deliberately small in-process registry. It avoids adding a
// resident metrics server or a large observability dependency to the
// no-Swap production host. The endpoint is reachable only through the
// loopback-bound backend; nginx does not proxy it publicly.
type metrics struct {
	requests       atomic.Uint64
	errors         atomic.Uint64
	durationNanos  atomic.Uint64
	requestSamples atomic.Uint64

	dependencyMu sync.RWMutex
	dependencies map[string]bool
	uploads      atomic.Uint64
	uploadErrors atomic.Uint64
	taskRuns     atomic.Uint64
	taskErrors   atomic.Uint64
}

func newMetrics() *metrics {
	return &metrics{dependencies: map[string]bool{
		"mysql":  false,
		"redis":  false,
		"upload": false,
	}}
}

func (m *metrics) observeRequest(status int, duration time.Duration) {
	if m == nil {
		return
	}
	m.requests.Add(1)
	m.requestSamples.Add(1)
	m.durationNanos.Add(uint64(maxDuration(duration)))
	if status >= http.StatusBadRequest {
		m.errors.Add(1)
	}
}

func (m *metrics) setDependency(name string, healthy bool) {
	if m == nil {
		return
	}
	m.dependencyMu.Lock()
	m.dependencies[name] = healthy
	m.dependencyMu.Unlock()
}

func (m *metrics) observeUpload(success bool) {
	if m == nil {
		return
	}
	if success {
		m.uploads.Add(1)
		m.setDependency("upload", true)
		return
	}
	m.uploadErrors.Add(1)
	m.setDependency("upload", false)
}

func (m *metrics) observeTask(success bool) {
	if m == nil {
		return
	}
	m.taskRuns.Add(1)
	if !success {
		m.taskErrors.Add(1)
	}
}

func maxDuration(value time.Duration) int64 {
	if value < 0 {
		return 0
	}
	return int64(value)
}

func (m *metrics) writePrometheus(w http.ResponseWriter) {
	m.dependencyMu.RLock()
	mysqlHealthy := m.dependencies["mysql"]
	redisHealthy := m.dependencies["redis"]
	uploadHealthy := m.dependencies["upload"]
	m.dependencyMu.RUnlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	requests := m.requests.Load()
	samples := m.requestSamples.Load()
	avgSeconds := 0.0
	if samples > 0 {
		avgSeconds = float64(m.durationNanos.Load()) / float64(samples) / float64(time.Second)
	}
	fmt.Fprintf(w, "# HELP promptos_http_requests_total Total HTTP requests.\n# TYPE promptos_http_requests_total counter\npromptos_http_requests_total %d\n", requests)
	fmt.Fprintf(w, "# HELP promptos_http_errors_total HTTP responses with status 4xx or 5xx.\n# TYPE promptos_http_errors_total counter\npromptos_http_errors_total %d\n", m.errors.Load())
	fmt.Fprintf(w, "# HELP promptos_http_request_duration_seconds_avg Average HTTP request duration.\n# TYPE promptos_http_request_duration_seconds_avg gauge\npromptos_http_request_duration_seconds_avg %.9f\n", avgSeconds)
	fmt.Fprintf(w, "# HELP promptos_uploads_total Successful image uploads.\n# TYPE promptos_uploads_total counter\npromptos_uploads_total %d\n", m.uploads.Load())
	fmt.Fprintf(w, "# HELP promptos_upload_errors_total Failed image uploads.\n# TYPE promptos_upload_errors_total counter\npromptos_upload_errors_total %d\n", m.uploadErrors.Load())
	fmt.Fprintf(w, "# HELP promptos_task_runs_total One-shot maintenance task runs.\n# TYPE promptos_task_runs_total counter\npromptos_task_runs_total %d\n", m.taskRuns.Load())
	fmt.Fprintf(w, "# HELP promptos_task_errors_total Failed one-shot maintenance task runs.\n# TYPE promptos_task_errors_total counter\npromptos_task_errors_total %d\n", m.taskErrors.Load())
	fmt.Fprintf(w, "# HELP promptos_dependency_healthy Dependency health (1 healthy, 0 unhealthy).\n# TYPE promptos_dependency_healthy gauge\npromptos_dependency_healthy{dependency=\"mysql\"} %s\npromptos_dependency_healthy{dependency=\"redis\"} %s\npromptos_dependency_healthy{dependency=\"upload\"} %s\n", boolMetric(mysqlHealthy), boolMetric(redisHealthy), boolMetric(uploadHealthy))
}

func boolMetric(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func (s *server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	if s.metrics == nil {
		s.metrics = newMetrics()
	}
	s.metrics.writePrometheus(w)
}

var numericPathSegment = regexp.MustCompile(`^[0-9]+$`)

// metricRoute removes high-cardinality IDs from route labels. It is kept
// available for future per-route counters without ever putting prompt titles,
// emails, or arbitrary query strings into metric labels.
func metricRoute(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if numericPathSegment.MatchString(part) {
			parts[i] = ":id"
		}
	}
	return strings.Join(parts, "/")
}

func statusString(status int) string { return strconv.Itoa(status) }
