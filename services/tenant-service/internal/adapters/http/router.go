package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// NewRouter wires all HTTP routes and returns a ready-to-use chi.Mux.
func NewRouter(h *Handler, logger *zap.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware stack.
	r.Use(Recovery(logger))
	r.Use(RequestID())
	r.Use(RequestLogger(logger))
	r.Use(CORS())
	r.Use(TenantContext())
	r.Use(chimiddleware.Compress(5))

	// Health check.
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// API v1 routes.
	r.Route("/api/v1", func(r chi.Router) {
		// Tenants.
		r.Route("/tenants", func(r chi.Router) {
			r.Post("/", h.CreateTenant)
			r.Get("/", h.ListTenants)

			r.Route("/{tenantId}", func(r chi.Router) {
				r.Get("/", h.GetTenant)
				r.Put("/", h.UpdateTenant)
				r.Delete("/", h.DeactivateTenant)
				r.Get("/settings", h.GetSettings)
				r.Put("/settings", h.UpdateSettings)
				r.Get("/members", h.ListMembers)
			})
		})

		// Profiles.
		r.Route("/profiles", func(r chi.Router) {
			r.Get("/{userId}", h.GetProfile)
			r.Put("/{userId}", h.UpdateProfile)
		})

		// Organizations.
		r.Route("/organizations", func(r chi.Router) {
			r.Get("/", h.ListOrganizations)
			r.Post("/", h.CreateOrganization)
		})
	})

	return r
}
