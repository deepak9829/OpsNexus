package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/opsnexus/auth-service/internal/ports"
	"go.uber.org/zap"
)

func NewRouter(handler *Handler, authSvc ports.AuthService, logger *zap.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(RequestID())
	r.Use(RequestLogger(logger))
	r.Use(Recovery(logger))
	r.Use(CORS())

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "auth-service"})
	})

	r.Route("/api/v1/auth", func(r chi.Router) {
		// Public routes
		r.Post("/register", handler.Register)
		r.Post("/login", handler.Login)
		r.Post("/refresh", handler.Refresh)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(JWTAuth(authSvc))
			r.Post("/logout", handler.Logout)
			r.Get("/me", handler.Me)
			r.Put("/me", handler.UpdateMe)

			// Admin only
			r.Group(func(r chi.Router) {
				r.Use(RequireRole("admin", "super_admin"))
				r.Get("/roles", handler.ListRoles)
				r.Post("/roles", handler.CreateRole)
				r.Get("/users", handler.ListUsers)
				r.Patch("/users/{userId}", handler.UpdateUserStatus)
				r.Post("/users/{userId}/roles", handler.AssignRole)
			})
		})
	})

	return r
}
