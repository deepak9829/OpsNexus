package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func NewRouter(handler *Handler, logger *zap.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(CORSMiddleware)
	r.Use(LoggingMiddleware(logger))

	r.Get("/health", handler.HealthCheck)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/notifications", func(r chi.Router) {
			r.Get("/", handler.ListNotifications)
			r.Post("/", handler.SendNotification)
			r.Post("/bulk", handler.SendBulkNotifications)
			r.Post("/mark-all-read", handler.MarkAllRead)
			r.Get("/{id}", handler.GetNotification)
			r.Post("/{id}/read", handler.MarkRead)
			r.Delete("/{id}", handler.DismissNotification)
		})

		r.Route("/audit", func(r chi.Router) {
			r.Get("/", handler.QueryAuditEvents)
			r.Post("/", handler.RecordAuditEvent)
			r.Get("/{id}", handler.GetAuditEvent)
		})
		r.Route("/audit-events", func(r chi.Router) {
			r.Get("/", handler.QueryAuditEvents)
			r.Post("/", handler.RecordAuditEvent)
			r.Get("/{id}", handler.GetAuditEvent)
		})
	})

	return r
}
