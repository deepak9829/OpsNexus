package application_test

import (
	"context"
	"testing"

	"github.com/opsnexus/document-service/internal/application"
	"github.com/opsnexus/document-service/internal/domain"
	"github.com/opsnexus/document-service/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Mock repositories

type mockFormRepo struct {
	forms map[string]*domain.FormTemplate
}

func newMockFormRepo() *mockFormRepo {
	return &mockFormRepo{forms: make(map[string]*domain.FormTemplate)}
}

func (m *mockFormRepo) Create(ctx context.Context, form *domain.FormTemplate) error {
	m.forms[form.ID] = form
	return nil
}

func (m *mockFormRepo) FindByID(ctx context.Context, id string) (*domain.FormTemplate, error) {
	f, ok := m.forms[id]
	if !ok {
		return nil, domain.ErrFormNotFound
	}
	return f, nil
}

func (m *mockFormRepo) ListByTenant(ctx context.Context, tenantID string, page, limit int) ([]*domain.FormTemplate, int64, error) {
	var result []*domain.FormTemplate
	for _, f := range m.forms {
		if f.TenantID == tenantID {
			result = append(result, f)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockFormRepo) Update(ctx context.Context, form *domain.FormTemplate) error {
	m.forms[form.ID] = form
	return nil
}

func (m *mockFormRepo) UpdateStatus(ctx context.Context, id string, status domain.FormStatus) error {
	f, ok := m.forms[id]
	if !ok {
		return domain.ErrFormNotFound
	}
	f.Status = status
	return nil
}

type mockSubmissionRepo struct {
	submissions map[string]*domain.FormSubmission
}

func newMockSubmissionRepo() *mockSubmissionRepo {
	return &mockSubmissionRepo{submissions: make(map[string]*domain.FormSubmission)}
}

func (m *mockSubmissionRepo) Create(ctx context.Context, sub *domain.FormSubmission) error {
	m.submissions[sub.ID] = sub
	return nil
}

func (m *mockSubmissionRepo) FindByID(ctx context.Context, id string) (*domain.FormSubmission, error) {
	s, ok := m.submissions[id]
	if !ok {
		return nil, domain.ErrSubmissionNotFound
	}
	return s, nil
}

func (m *mockSubmissionRepo) ListByTenant(ctx context.Context, tenantID string, page, limit int) ([]*domain.FormSubmission, int64, error) {
	var result []*domain.FormSubmission
	for _, s := range m.submissions {
		if s.TenantID == tenantID {
			result = append(result, s)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockSubmissionRepo) ListByForm(ctx context.Context, formID string, page, limit int) ([]*domain.FormSubmission, int64, error) {
	var result []*domain.FormSubmission
	for _, s := range m.submissions {
		if s.FormID == formID {
			result = append(result, s)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockSubmissionRepo) UpdateStatus(ctx context.Context, id string, status domain.SubmissionStatus) error {
	s, ok := m.submissions[id]
	if !ok {
		return domain.ErrSubmissionNotFound
	}
	s.Status = status
	return nil
}

type mockDocRepo struct {
	docs     map[string]*domain.Document
	versions map[string][]*domain.DocumentVersion
}

func newMockDocRepo() *mockDocRepo {
	return &mockDocRepo{
		docs:     make(map[string]*domain.Document),
		versions: make(map[string][]*domain.DocumentVersion),
	}
}

func (m *mockDocRepo) Create(ctx context.Context, doc *domain.Document) error {
	m.docs[doc.ID] = doc
	return nil
}

func (m *mockDocRepo) FindByID(ctx context.Context, id string) (*domain.Document, error) {
	d, ok := m.docs[id]
	if !ok {
		return nil, domain.ErrDocumentNotFound
	}
	return d, nil
}

func (m *mockDocRepo) ListByTenant(ctx context.Context, tenantID string, page, limit int) ([]*domain.Document, int64, error) {
	var result []*domain.Document
	for _, d := range m.docs {
		if d.TenantID == tenantID {
			result = append(result, d)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockDocRepo) Delete(ctx context.Context, id string) error {
	delete(m.docs, id)
	return nil
}

func (m *mockDocRepo) AddVersion(ctx context.Context, version *domain.DocumentVersion) error {
	m.versions[version.DocumentID] = append(m.versions[version.DocumentID], version)
	return nil
}

func (m *mockDocRepo) ListVersions(ctx context.Context, documentID string) ([]*domain.DocumentVersion, error) {
	return m.versions[documentID], nil
}

// Tests

func TestCreateTemplate(t *testing.T) {
	logger := zap.NewNop()
	formRepo := newMockFormRepo()
	subRepo := newMockSubmissionRepo()
	svc := application.NewFormService(formRepo, subRepo, logger)

	req := ports.CreateFormRequest{
		Name:        "Test Form",
		Description: "A test form",
		Fields: []domain.FormField{
			{Name: "email", Type: domain.FieldTypeEmail, Label: "Email", Required: true},
			{Name: "name", Type: domain.FieldTypeText, Label: "Name", Required: false},
		},
	}

	form, err := svc.CreateTemplate(context.Background(), "tenant-1", "user-1", req)
	require.NoError(t, err)
	assert.NotEmpty(t, form.ID)
	assert.Equal(t, "Test Form", form.Name)
	assert.Equal(t, "tenant-1", form.TenantID)
	assert.Equal(t, domain.FormStatusDraft, form.Status)
	assert.Equal(t, 1, form.Version)
	assert.Len(t, form.Fields, 2)
}

func TestSubmitForm_Valid(t *testing.T) {
	logger := zap.NewNop()
	formRepo := newMockFormRepo()
	subRepo := newMockSubmissionRepo()
	svc := application.NewFormService(formRepo, subRepo, logger)

	createReq := ports.CreateFormRequest{
		Name: "Contact Form",
		Fields: []domain.FormField{
			{Name: "email", Type: domain.FieldTypeEmail, Label: "Email", Required: true},
			{Name: "message", Type: domain.FieldTypeTextarea, Label: "Message", Required: true},
		},
	}
	form, err := svc.CreateTemplate(context.Background(), "tenant-1", "user-1", createReq)
	require.NoError(t, err)

	submitReq := ports.SubmitFormRequest{
		FormID: form.ID,
		Data: map[string]any{
			"email":   "test@example.com",
			"message": "Hello world",
		},
	}

	sub, err := svc.SubmitForm(context.Background(), "tenant-1", "user-2", submitReq)
	require.NoError(t, err)
	assert.NotEmpty(t, sub.ID)
	assert.Equal(t, form.ID, sub.FormID)
	assert.Equal(t, domain.SubmissionStatusSubmitted, sub.Status)
}

func TestSubmitForm_MissingRequired(t *testing.T) {
	logger := zap.NewNop()
	formRepo := newMockFormRepo()
	subRepo := newMockSubmissionRepo()
	svc := application.NewFormService(formRepo, subRepo, logger)

	createReq := ports.CreateFormRequest{
		Name: "Required Form",
		Fields: []domain.FormField{
			{Name: "name", Type: domain.FieldTypeText, Label: "Name", Required: true},
		},
	}
	form, err := svc.CreateTemplate(context.Background(), "tenant-1", "user-1", createReq)
	require.NoError(t, err)

	submitReq := ports.SubmitFormRequest{
		FormID: form.ID,
		Data:   map[string]any{},
	}

	_, err = svc.SubmitForm(context.Background(), "tenant-1", "user-2", submitReq)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidFormData)
}

func TestUploadDocument(t *testing.T) {
	logger := zap.NewNop()
	docRepo := newMockDocRepo()
	svc := application.NewDocumentService(docRepo, t.TempDir(), logger)

	req := ports.UploadRequest{
		Filename:  "test.pdf",
		MimeType:  "application/pdf",
		SizeBytes: 1024,
		Content:   []byte("fake pdf content"),
	}

	doc, err := svc.UploadDocument(context.Background(), "tenant-1", "user-1", req)
	require.NoError(t, err)
	assert.NotEmpty(t, doc.ID)
	assert.Equal(t, "test.pdf", doc.Filename)
	assert.Equal(t, int64(1024), doc.SizeBytes)
}

func TestGetDocument_NotFound(t *testing.T) {
	logger := zap.NewNop()
	docRepo := newMockDocRepo()
	svc := application.NewDocumentService(docRepo, t.TempDir(), logger)

	_, err := svc.GetDocument(context.Background(), "nonexistent-id")
	assert.ErrorIs(t, err, domain.ErrDocumentNotFound)
}
