package middleware

import (
	"net/http"

	"github.com/launchdarkly/ld-relay/v8/internal/metrics"

	"github.com/gorilla/mux"
)

func withCount(handler http.Handler, measure metrics.Measure) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := GetEnvContextInfo(req.Context()).Env
		userAgent := getUserAgent(req)
		metrics.WithCount(ctx.GetMetricsContext(), userAgent, func() {
			handler.ServeHTTP(w, req)
		}, measure)
	})
}

func withGauge(handler http.Handler, measure metrics.Measure) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := GetEnvContextInfo(req.Context())
		userAgent := getUserAgent(req)
		metrics.WithGauge(ctx.Env.GetMetricsContext(), userAgent, func() {
			handler.ServeHTTP(w, req)
		}, measure)
	})
}

// CountMobileConns is a middleware function that increments the total number of mobile connections,
// and also increments the number of active mobile connections until the handler ends.
func CountMobileConns(handler http.Handler) http.Handler {
	return withCount(withGauge(handler, metrics.MobileConns), metrics.NewMobileConns)
}

// CountBrowserConns is a middleware function that increments the total number of browser connections,
// and also increments the number of active browser connections until the handler ends.
func CountBrowserConns(handler http.Handler) http.Handler {
	return withCount(withGauge(handler, metrics.BrowserConns), metrics.NewBrowserConns)
}

// CountServerConns is a middleware function that increments the total number of server-side connections,
// and also increments the number of active server-side connections until the handler ends.
func CountServerConns(handler http.Handler) http.Handler {
	return withCount(withGauge(handler, metrics.ServerConns), metrics.NewServerConns)
}

// PollingRequestCount is a middleware function that increments the total number of server-side polling requests.
func PollingRequestCount(handler http.Handler) http.Handler {
	return withCount(handler, metrics.PollingRequests)
}

// RequestCount is a middleware function that increments the specified metric for each request.
func RequestCount(measure metrics.Measure) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := GetEnvContextInfo(req.Context())
			userAgent := getUserAgent(req)
			// Ignoring internal routing error that would have been ignored anyway
			route, _ := mux.CurrentRoute(req).GetPathTemplate()
			metrics.WithRouteCount(ctx.Env.GetMetricsContext(), userAgent, route, req.Method, func() {
				next.ServeHTTP(w, req)
			}, measure)
		})
	}
}

// RequestLatency is a middleware function that records the latency of requests.
func RequestLatency(measure metrics.Float64Measure) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := GetEnvContextInfo(req.Context())
			userAgent := getUserAgent(req)
			// Ignoring internal routing error that would have been ignored anyway
			route, _ := mux.CurrentRoute(req).GetPathTemplate()
			metrics.WithRouteLatency(ctx.Env.GetMetricsContext(), userAgent, route, req.Method, func() {
				next.ServeHTTP(w, req)
			}, measure)
		})
	}
}

// RequestErrors is a middleware function that records errors based on HTTP status codes.
func RequestErrors(measure metrics.Measure) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := GetEnvContextInfo(req.Context())
			userAgent := getUserAgent(req)
			
			// Create a response writer wrapper to capture the status code
			wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: 200}
			
			next.ServeHTTP(wrappedWriter, req)
			
			// Record error if status code indicates an error (4xx or 5xx)
			if wrappedWriter.statusCode >= 400 {
				metrics.RecordError(ctx.Env.GetMetricsContext(), userAgent, measure)
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
