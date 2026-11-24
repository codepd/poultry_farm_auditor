package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

const traceIDContextKey contextKey = "trace_id"
const correlationIDContextKey contextKey = "correlation_id"

var appLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: ParseLogLevel(os.Getenv("LOG_LEVEL")),
}))

// ParseLogLevel maps LOG_LEVEL env var (DEBUG, INFO, WARN, ERROR) to slog.Level.
func ParseLogLevel(raw string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Logger returns the application's structured JSON logger.
func Logger() *slog.Logger {
	return appLogger
}

// GetTraceIDFromContext extracts the W3C trace_id from context if present.
func GetTraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(traceIDContextKey).(string); ok {
		return v
	}
	return ""
}

// LogWithContext returns a logger enriched with trace_id and correlation_id from ctx.
func LogWithContext(ctx context.Context) *slog.Logger {
	traceID := GetTraceIDFromContext(ctx)
	corrID, _ := ctx.Value(correlationIDContextKey).(string)
	if traceID == "" && corrID == "" {
		return appLogger
	}
	return appLogger.With("trace_id", traceID, "correlation_id", corrID)
}

func randomHex(numBytes int) string {
	b := make([]byte, numBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func isValidHexLen(s string, expectedLen int) bool {
	if len(s) != expectedLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// ParseOrCreateTraceContext extracts W3C traceparent (00-<32hex>-<16hex>-<2hex>) or generates a fresh trace context.
func ParseOrCreateTraceContext(traceparentHeader, correlationHeader string) (traceID, spanID, responseTraceparent, correlationID string) {
	parts := strings.Split(strings.TrimSpace(traceparentHeader), "-")
	if len(parts) == 4 && isValidHexLen(parts[1], 32) && isValidHexLen(parts[2], 16) {
		traceID = strings.ToLower(parts[1])
	} else {
		traceID = randomHex(16) // 32 hex chars
	}
	spanID = randomHex(8) // 16 hex chars
	responseTraceparent = fmt.Sprintf("00-%s-%s-01", traceID, spanID)

	correlationID = strings.TrimSpace(correlationHeader)
	if correlationID == "" {
		correlationID = traceID
	}
	return traceID, spanID, responseTraceparent, correlationID
}

type statusCaptureWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (w *statusCaptureWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusCaptureWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Logging propagates W3C traceparent / X-Correlation-ID and logs structured JSON per request.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		traceID, spanID, outTraceparent, corrID := ParseOrCreateTraceContext(
			r.Header.Get("traceparent"),
			r.Header.Get("X-Correlation-ID"),
		)

		w.Header().Set("traceparent", outTraceparent)
		w.Header().Set("X-Correlation-ID", corrID)

		ctx := context.WithValue(r.Context(), traceIDContextKey, traceID)
		ctx = context.WithValue(ctx, correlationIDContextKey, corrID)
		r = r.WithContext(ctx)

		wrapped := &statusCaptureWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		durationMs := float64(time.Since(start).Microseconds()) / 1000.0
		level := slog.LevelInfo
		if wrapped.statusCode >= 500 {
			level = slog.LevelError
		} else if wrapped.statusCode >= 400 {
			level = slog.LevelWarn
		} else if r.URL.Path == "/health" || r.URL.Path == "/api/health" {
			level = slog.LevelDebug
		}

		attrs := []slog.Attr{
			slog.String("trace_id", traceID),
			slog.String("span_id", spanID),
			slog.String("correlation_id", corrID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", wrapped.statusCode),
			slog.Float64("duration_ms", durationMs),
			slog.Int("bytes", wrapped.bytes),
			slog.String("remote_ip", r.RemoteAddr),
		}
		if claims := GetUserFromContext(r); claims != nil {
			attrs = append(attrs,
				slog.Int("user_id", claims.UserID),
				slog.String("tenant_id", claims.TenantID),
				slog.String("role", claims.Role),
			)
		}

		appLogger.LogAttrs(ctx, level, "http_request", attrs...)
	})
}
