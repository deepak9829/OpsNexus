package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/opsnexus/document-service/internal/domain"
	"github.com/opsnexus/document-service/internal/ports"
	"go.uber.org/zap"
)

const maxFileSizeBytes = 50 * 1024 * 1024 // 50MB

var allowedMimeTypes = map[string]bool{
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"text/plain": true,
	"text/csv":   true,
}

type documentService struct {
	docRepo   ports.DocumentRepository
	uploadDir string
	logger    *zap.Logger
}

func NewDocumentService(
	docRepo ports.DocumentRepository,
	uploadDir string,
	logger *zap.Logger,
) ports.DocumentService {
	return &documentService{
		docRepo:   docRepo,
		uploadDir: uploadDir,
		logger:    logger,
	}
}

func (s *documentService) UploadDocument(ctx context.Context, tenantID, userID string, req ports.UploadRequest) (*domain.Document, error) {
	if req.SizeBytes > maxFileSizeBytes {
		return nil, domain.ErrFileTooLarge
	}

	if !allowedMimeTypes[req.MimeType] {
		return nil, domain.ErrUnsupportedFileType
	}

	docID := uuid.New().String()
	storageKey := fmt.Sprintf("%s/%s/%s", tenantID, docID, req.Filename)
	fullPath := filepath.Join(s.uploadDir, storageKey)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	if err := os.WriteFile(fullPath, req.Content, 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	now := time.Now().UTC()
	doc := &domain.Document{
		ID:             docID,
		TenantID:       tenantID,
		Filename:       req.Filename,
		OriginalName:   req.Filename,
		MimeType:       req.MimeType,
		SizeBytes:      req.SizeBytes,
		StorageKey:     storageKey,
		UploadedBy:     userID,
		CaseID:         req.CaseID,
		VersionCount:   1,
		CurrentVersion: 1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.docRepo.Create(ctx, doc); err != nil {
		_ = os.Remove(fullPath)
		return nil, err
	}

	version := &domain.DocumentVersion{
		ID:         uuid.New().String(),
		DocumentID: docID,
		Version:    1,
		StorageKey: storageKey,
		UploadedBy: userID,
		CreatedAt:  now,
	}
	if err := s.docRepo.AddVersion(ctx, version); err != nil {
		s.logger.Warn("failed to record document version", zap.Error(err))
	}

	s.logger.Info("document uploaded", zap.String("id", docID), zap.String("tenant_id", tenantID))
	return doc, nil
}

func (s *documentService) GetDocument(ctx context.Context, id string) (*domain.Document, error) {
	return s.docRepo.FindByID(ctx, id)
}

func (s *documentService) DeleteDocument(ctx context.Context, id string) error {
	doc, err := s.docRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(s.uploadDir, doc.StorageKey)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		s.logger.Warn("failed to delete file from storage", zap.Error(err))
	}

	return s.docRepo.Delete(ctx, id)
}

func (s *documentService) ListDocuments(ctx context.Context, tenantID string, page, limit int) ([]*domain.Document, int64, error) {
	return s.docRepo.ListByTenant(ctx, tenantID, page, limit)
}

func (s *documentService) GetVersions(ctx context.Context, documentID string) ([]*domain.DocumentVersion, error) {
	return s.docRepo.ListVersions(ctx, documentID)
}
