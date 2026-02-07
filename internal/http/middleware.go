package http

import (
	"jwtea/internal/core"
	"log"
	"net/http"
	"strings"
	"time"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.status = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	if rr.status == 0 {
		rr.status = http.StatusOK
	}
	n, err := rr.ResponseWriter.Write(b)
	rr.bytes += n
	return n, err
}

type LoggingMiddleware struct {
	logHub *core.LogHub
	chaos  *core.ChaosFlags
}

func NewLoggingMiddleware(logHub *core.LogHub, chaos *core.ChaosFlags) *LoggingMiddleware {
	return &LoggingMiddleware{
		logHub: logHub,
		chaos:  chaos,
	}
}

func (m *LoggingMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.chaos.IsSimulate500() {
			http.Error(w, "Chaos: Simulated 500 Internal Server Error", http.StatusInternalServerError)
			m.logHub.Append(core.LogEntry{
				Time:      time.Now(),
				Method:    r.Method,
				Path:      r.URL.RequestURI(),
				Status:    http.StatusInternalServerError,
				Duration:  0,
				RemoteIP:  clientIP(r),
				UserAgent: r.UserAgent(),
				Bytes:     0,
			})
			return
		}

		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w}
		defer func() {
			if rec := recover(); rec != nil {
				rr.status = http.StatusInternalServerError
				log.Printf("panic handling %s %s: %v", r.Method, r.URL.Path, rec)
			}
			if rr.status == 0 {
				rr.status = http.StatusOK
			}
			dur := time.Since(start)
			if m.logHub != nil {
				m.logHub.Append(core.LogEntry{
					Time:      start,
					Method:    r.Method,
					Path:      r.URL.RequestURI(),
					Status:    rr.status,
					Duration:  dur,
					RemoteIP:  clientIP(r),
					UserAgent: r.UserAgent(),
					Bytes:     rr.bytes,
				})
			}
		}()
		next.ServeHTTP(rr, r)
	})
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		if idx := strings.Index(xff, ","); idx >= 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return xff
	}
	if xr := strings.TrimSpace(r.Header.Get("X-Real-IP")); xr != "" {
		return xr
	}
	return r.RemoteAddr
}
