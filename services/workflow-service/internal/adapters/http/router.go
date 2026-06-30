package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func NewRouter(h *Handler, logger *zap.Logger) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RealIP)
	r.Use(RequestID)
	r.Use(CORS)
	r.Use(RequestLogger(logger))
	r.Use(Recovery(logger))

	// Health check (no auth)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "workflow-service"})
	})

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(TenantContext)
		r.Use(UserContext)

		// Cases
		r.Route("/cases", func(r chi.Router) {
			r.Get("/", h.ListCases)
			r.Post("/", h.CreateCase)

			r.Route("/{caseId}", func(r chi.Router) {
				r.Get("/", h.GetCase)
				r.Put("/", h.UpdateCase)
				r.Post("/transitions", h.TransitionCase)
				r.Post("/assign", h.AssignCase)

				// Tasks nested under case
				r.Get("/tasks", h.ListTasks)
				r.Post("/tasks", h.CreateTask)

				// Comments nested under case
				r.Get("/comments", h.ListComments)
				r.Post("/comments", h.AddComment)
			})
		})

		// Tasks (standalone)
		r.Route("/tasks/{taskId}", func(r chi.Router) {
			r.Put("/", h.UpdateTask)
			r.Post("/complete", h.CompleteTask)
		})

		// Workflow templates
		r.Route("/workflows", func(r chi.Router) {
			r.Get("/", h.ListWorkflows)
			r.Post("/", h.CreateWorkflow)
		})
	})

	return r
}
