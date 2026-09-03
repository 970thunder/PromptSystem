package api

import (
	"context"
	"net/http"
	"time"
)

// handleHealthLive reports that the process is alive and serving requests.
func (s *server) handleHealthLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[map[string]any]{
		Code:    200,
		Message: "Success",
		Data: map[string]any{
			"status":  "ok",
			"service": "promptos-backend",
		},
	})
}

// handleHealthReady reports whether dependencies (MySQL) are reachable and the
// store mode. In development, memory fallback is reported as degraded rather
// than failing the probe.
func (s *server) handleHealthReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	degraded := s.storageMode != "mysql"
	dependencies := map[string]bool{
		"mysql":  s.storageMode == "mysql",
		"redis":  s.cache != nil,
		"upload": s.imageStorage != nil,
	}
	if s.readyCheck != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		for name, healthy := range s.readyCheck(ctx) {
			dependencies[name] = healthy
		}
		if !dependencies["mysql"] || !dependencies["redis"] {
			degraded = true
		}
	}
	if s.metrics != nil {
		for name, healthy := range dependencies {
			s.metrics.setDependency(name, healthy)
		}
	}
	data := map[string]any{
		"status":       "ready",
		"service":      "promptos-backend",
		"environment":  s.config.AppEnv,
		"storageMode":  s.storageMode,
		"degraded":     degraded,
		"dependencies": dependencies,
	}
	if degraded {
		data["degradedReason"] = "storage is not backed by MySQL"
	}

	code := http.StatusOK
	message := "Success"
	if degraded && s.config.AppEnv != "development" {
		code = http.StatusServiceUnavailable
		message = "Degraded"
	}

	writeJSON(w, code, apiResponse[map[string]any]{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
