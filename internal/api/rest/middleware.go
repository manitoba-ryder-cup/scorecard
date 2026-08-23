package rest

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"slices"

	"github.com/google/uuid"
	"github.com/travisbale/knowhere/identity"
	"github.com/travisbale/knowhere/jwt"
)

// optionalAuth guards a public read route. With an Authorization header it delegates
// to full JWT authentication (tenant + actor from the token; 401 on a bad token).
// Without one, it falls back to the configured public tenant so anonymous spectators
// can read a single-tenant site; if no public tenant is configured, it is 401 (a
// multi-tenant deployment requires login even to read).
func optionalAuth(m *jwt.HTTPMiddleware, publicTenantID *uuid.UUID, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != "" {
			m.Authenticate(next)(w, req)
			return
		}
		if publicTenantID == nil {
			respondError(req.Context(), w, http.StatusUnauthorized, "authentication required", nil)
			return
		}
		ctx := identity.WithTenant(req.Context(), *publicTenantID)
		next(w, req.WithContext(ctx))
	}
}

// recoverMiddleware turns a panic in a downstream handler into a 500 instead of
// crashing the server. knowhere provides no recoverer; this mirrors heimdall.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log with a stack so a recovered panic leaves a diagnosable trail.
				slog.ErrorContext(req.Context(), "panic recovered", "error", err, "stack", string(debug.Stack()))
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, req)
	})
}

// hasScope reports whether the request's token carries a scope beyond the one its route
// required. Absent claims read as no scopes, so an unauthenticated caller never qualifies.
func hasScope(req *http.Request, scope string) bool {
	claims, err := jwt.GetJWTClaims(req)
	if err != nil {
		return false
	}
	return slices.Contains(claims.Scopes, jwt.Scope(scope))
}
