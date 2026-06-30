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
		r.Route("/forms", func(r chi.Router) {
			r.Get("/", handler.ListFormTemplates)
			r.Post("/", handler.CreateFormTemplate)
			r.Get("/{id}", handler.GetFormTemplate)
			r.Put("/{id}", handler.UpdateFormTemplate)
			r.Post("/{id}/publish", handler.PublishFormTemplate)
			r.Post("/{id}/archive", handler.ArchiveFormTemplate)
		})

		r.Route("/submissions", func(r chi.Router) {
			r.Get("/", handler.ListSubmissions)
			r.Post("/", handler.SubmitForm)
			r.Get("/{id}", handler.GetSubmission)
		})

		r.Route("/documents", func(r chi.Router) {
			r.Get("/", handler.ListDocuments)
			r.Post("/upload", handler.UploadDocument)
			r.Get("/{id}", handler.GetDocument)
			r.Delete("/{id}", handler.DeleteDocument)
			r.Get("/{id}/versions", handler.GetDocumentVersions)
		})
	})

	return r
}
