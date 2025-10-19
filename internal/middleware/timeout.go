package middleware

import (
	"context"
	"net/http"
	"time"
)

// Timeout wraps a handler with a timeout to prevent slow requests.
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create a channel to signal completion
			done := make(chan struct{})

			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				// Request completed successfully
				return
			case <-ctx.Done():
				// Request timed out
				http.Error(w, "Request timeout", http.StatusRequestTimeout)
				return
			}
		})
	}
}
