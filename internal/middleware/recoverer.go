package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/mochammadbagas/go-crud-api/internal/utils"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// Recoverer returns a structured panic recovery middleware using slog.
// It catches unhandled runtime panics, logs full structured details (request_id, panic error, stack trace),
// and returns a standard HTTP 500 JSON response, preventing process aborts and information disclosure (CWE-209).
func Recoverer() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					if rvr == http.ErrAbortHandler {
						panic(rvr)
					}

					reqID := chimiddleware.GetReqID(r.Context())
					stack := string(debug.Stack())

					slog.ErrorContext(r.Context(), "HTTP request panic recovered",
						slog.String("request_id", reqID),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
						slog.Any("panic", fmt.Sprintf("%v", rvr)),
						slog.String("stack", stack),
					)

					w.Header().Set("Connection", "close")
					utils.ErrorCode(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "an internal server error occurred")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
