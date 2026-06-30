package application_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/opsnexus/notification-service/internal/application"
	"github.com/opsnexus/notification-service/internal/domain"
	"github.com/opsnexus/notification-service/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Mock notification repository

type mockNotificationRepo struct {
	mu            sync.RWMutex
	notifications map[string]*domain.Notification
}

func newMockNotificationRepo() *mockNotificationRepo {
	return &mockNotificationRepo{notifications: make(map[string]*domain.Notification)}
}

func (m *mockNotificationRepo) Create(ctx context.Context, n *domain.Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications[n.ID] = n
	return nil
}

func (m *mockNotificationRepo) FindByID(ctx context.Context, tenantID, id string) (*domain.Notification, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, ok := m.notifications[id]
	if !ok || n.TenantID != tenantID {
		return nil, domain.ErrNotificationNotFound
	}
	return n, nil
}

func (m *mockNotificationRepo) ListByUser(ctx context.Context, tenantID, userID string, page, limit int) ([]*domain.Notification, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.Notification
	for _, n := range m.notifications {
		if n.TenantID == tenantID && n.UserID == userID {
			result = append(result, n)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockNotificationRepo) MarkRead(ctx context.Context, tenantID, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, ok := m.notifications[id]
	if !ok || n.TenantID != tenantID {
		return domain.ErrNotificationNotFound
	}
	n.Read = true
	now := time.Now()
	n.ReadAt = &now
	return nil
}

func (m *mockNotificationRepo) MarkAllRead(ctx context.Context, tenantID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for _, n := range m.notifications {
		if n.TenantID == tenantID && n.UserID == userID {
			n.Read = true
			n.ReadAt = &now
		}
	}
	return nil
}

func (m *mockNotificationRepo) Delete(ctx context.Context, tenantID, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.notifications, id)
	return nil
}

// Mock audit repository

type mockAuditRepo struct {
	mu     sync.RWMutex
	events map[string]*domain.AuditEvent
}

func newMockAuditRepo() *mockAuditRepo {
	return &mockAuditRepo{events: make(map[string]*domain.AuditEvent)}
}

func (m *mockAuditRepo) Record(ctx context.Context, event *domain.AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events[event.ID] = event
	return nil
}

func (m *mockAuditRepo) FindByID(ctx context.Context, tenantID, id string) (*domain.AuditEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.events[id]
	if !ok || e.TenantID != tenantID {
		return nil, domain.ErrAuditEventNotFound
	}
	return e, nil
}

func (m *mockAuditRepo) Query(ctx context.Context, tenantID string, filter ports.AuditFilter, page, limit int) ([]*domain.AuditEvent, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.AuditEvent
	for _, e := range m.events {
		if e.TenantID != tenantID {
			continue
		}
		if filter.ActorID != nil && e.ActorID != *filter.ActorID {
			continue
		}
		if filter.Action != nil && e.Action != *filter.Action {
			continue
		}
		if filter.Resource != nil && e.Resource != *filter.Resource {
			continue
		}
		result = append(result, e)
	}
	return result, int64(len(result)), nil
}

// Tests

func TestSend_Success(t *testing.T) {
	logger := zap.NewNop()
	repo := newMockNotificationRepo()
	svc := application.NewNotificationService(repo, logger)

	req := ports.SendNotificationRequest{
		TenantID: "tenant-1",
		UserID:   "user-1",
		Type:     domain.NotificationTypeInfo,
		Title:    "Test Notification",
		Body:     "This is a test",
		Channel:  domain.ChannelInApp,
	}

	n, err := svc.Send(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, n.ID)
	assert.Equal(t, "tenant-1", n.TenantID)
	assert.Equal(t, "user-1", n.UserID)
	assert.Equal(t, "Test Notification", n.Title)
	assert.False(t, n.Read)
}

func TestMarkRead_Success(t *testing.T) {
	logger := zap.NewNop()
	repo := newMockNotificationRepo()
	svc := application.NewNotificationService(repo, logger)

	req := ports.SendNotificationRequest{
		TenantID: "tenant-1",
		UserID:   "user-1",
		Type:     domain.NotificationTypeInfo,
		Title:    "Test",
		Body:     "Body",
		Channel:  domain.ChannelInApp,
	}

	n, err := svc.Send(context.Background(), req)
	require.NoError(t, err)

	err = svc.MarkRead(context.Background(), "tenant-1", n.ID)
	require.NoError(t, err)

	updated, err := svc.GetNotification(context.Background(), "tenant-1", n.ID)
	require.NoError(t, err)
	assert.True(t, updated.Read)
	assert.NotNil(t, updated.ReadAt)
}

func TestRecord_AuditEvent(t *testing.T) {
	logger := zap.NewNop()
	repo := newMockAuditRepo()
	svc := application.NewAuditService(repo, logger)

	req := ports.RecordAuditRequest{
		TenantID:   "tenant-1",
		ActorID:    "user-1",
		ActorEmail: "user@example.com",
		Action:     "case.created",
		Resource:   "case",
		ResourceID: "case-123",
		IPAddress:  "127.0.0.1",
		UserAgent:  "Mozilla/5.0",
	}

	event, err := svc.Record(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, event.ID)
	assert.Equal(t, "case.created", event.Action)
	assert.Equal(t, "user-1", event.ActorID)
	assert.False(t, event.Timestamp.IsZero())
}

func TestQueryEvents(t *testing.T) {
	logger := zap.NewNop()
	repo := newMockAuditRepo()
	svc := application.NewAuditService(repo, logger)

	ctx := context.Background()

	// Create multiple events
	actions := []string{"case.created", "user.login", "case.updated", "document.uploaded"}
	for _, action := range actions {
		_, err := svc.Record(ctx, ports.RecordAuditRequest{
			TenantID:   "tenant-1",
			ActorID:    "user-1",
			ActorEmail: "user@example.com",
			Action:     action,
			Resource:   "case",
			ResourceID: "case-123",
		})
		require.NoError(t, err)
	}

	// Query all events for tenant
	events, total, err := svc.QueryEvents(ctx, "tenant-1", ports.AuditFilter{}, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)
	assert.Len(t, events, 4)

	// Query with action filter
	action := "case.created"
	filtered, filteredTotal, err := svc.QueryEvents(ctx, "tenant-1", ports.AuditFilter{Action: &action}, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), filteredTotal)
	assert.Len(t, filtered, 1)
	assert.Equal(t, "case.created", filtered[0].Action)
}
