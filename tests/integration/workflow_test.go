//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// GORM models (mirror workflow-service domain models)
// ---------------------------------------------------------------------------

type Case struct {
	ID           string     `gorm:"primaryKey;type:char(36)"`
	TenantID     string     `gorm:"type:char(36);not null;index"`
	CaseNumber   string     `gorm:"type:varchar(32);not null;uniqueIndex"`
	Title        string     `gorm:"type:varchar(512);not null"`
	Description  string     `gorm:"type:text"`
	Status       string     `gorm:"type:varchar(32);not null;default:'open'"`
	Priority     string     `gorm:"type:varchar(32);not null;default:'medium'"`
	AssigneeID   *string    `gorm:"type:char(36)"`
	CreatedByID  string     `gorm:"type:char(36);not null"`
	ClosedAt     *time.Time `gorm:"default:null"`
	CreatedAt    time.Time  `gorm:"autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime"`
}

type CaseTransition struct {
	ID          string    `gorm:"primaryKey;type:char(36)"`
	CaseID      string    `gorm:"type:char(36);not null;index"`
	TenantID    string    `gorm:"type:char(36);not null;index"`
	FromStatus  string    `gorm:"type:varchar(32);not null"`
	ToStatus    string    `gorm:"type:varchar(32);not null"`
	ActorID     string    `gorm:"type:char(36);not null"`
	Comment     string    `gorm:"type:text"`
	TransitionedAt time.Time `gorm:"not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

type Task struct {
	ID          string     `gorm:"primaryKey;type:char(36)"`
	CaseID      string     `gorm:"type:char(36);not null;index"`
	TenantID    string     `gorm:"type:char(36);not null;index"`
	Title       string     `gorm:"type:varchar(512);not null"`
	AssigneeID  *string    `gorm:"type:char(36)"`
	Status      string     `gorm:"type:varchar(32);not null;default:'pending'"`
	CompletedAt *time.Time `gorm:"default:null"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`
}

type CaseCounter struct {
	TenantID string `gorm:"primaryKey;type:char(36)"`
	Counter  int64  `gorm:"not null;default:0"`
}

// ---------------------------------------------------------------------------
// Package-level workflow DB handle
// ---------------------------------------------------------------------------

// wfDB is the workflow service database connection.
// It is initialized in TestMain (auth_test.go) or separately if this package
// is run in isolation. If TEST_WF_MYSQL_DSN is not set, we reuse testDB with
// a separate table prefix to avoid collisions.
//
// For simplicity in a combined test binary, we reuse testDB (the shared
// connection from TestMain). The workflow tables are prefixed with "wf_".
func wfDB() *gorm.DB {
	return testDB
}

// ensureWorkflowSchema auto-migrates workflow tables on first use.
func ensureWorkflowSchema(t *testing.T) {
	t.Helper()
	err := testDB.AutoMigrate(&Case{}, &CaseTransition{}, &Task{}, &CaseCounter{})
	require.NoError(t, err, "AutoMigrate workflow tables")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// nextCaseNumber increments the tenant's counter and returns the formatted
// case number (e.g., "CASE-00042").
func nextCaseNumber(t *testing.T, db *gorm.DB, tenantID string) string {
	t.Helper()
	ctx := context.Background()

	// Upsert counter row
	err := db.WithContext(ctx).Exec(
		`INSERT INTO case_counters (tenant_id, counter) VALUES (?, 1)
		 ON DUPLICATE KEY UPDATE counter = counter + 1`,
		tenantID,
	).Error
	require.NoError(t, err)

	var cc CaseCounter
	require.NoError(t, db.WithContext(ctx).First(&cc, "tenant_id = ?", tenantID).Error)
	return fmt.Sprintf("CASE-%05d", cc.Counter)
}

func createTestCase(t *testing.T, tenantID, status, createdByID string) Case {
	t.Helper()
	ctx := context.Background()
	db := wfDB()

	caseNumber := nextCaseNumber(t, db, tenantID)
	c := Case{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		CaseNumber:  caseNumber,
		Title:       fmt.Sprintf("Test case %s", caseNumber),
		Description: "Integration test case",
		Status:      status,
		Priority:    "medium",
		CreatedByID: createdByID,
	}
	require.NoError(t, db.WithContext(ctx).Create(&c).Error)
	t.Cleanup(func() {
		db.Unscoped().Where("case_id = ?", c.ID).Delete(&CaseTransition{})
		db.Unscoped().Where("case_id = ?", c.ID).Delete(&Task{})
		db.Unscoped().Delete(&c)
	})
	return c
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestCaseRepository_CreateAndFind creates a case, verifies the auto-generated
// case number format, and retrieves the case by ID.
func TestCaseRepository_CreateAndFind(t *testing.T) {
	ensureWorkflowSchema(t)
	ctx := context.Background()
	db := wfDB()

	tenantID := newTenantID()
	createdByID := newUserID()
	caseNumber := nextCaseNumber(t, db, tenantID)

	c := Case{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		CaseNumber:  caseNumber,
		Title:       "Network outage in prod",
		Description: "All pods restarting on cluster A",
		Status:      "open",
		Priority:    "critical",
		CreatedByID: createdByID,
	}
	require.NoError(t, db.WithContext(ctx).Create(&c).Error)
	t.Cleanup(func() { db.Unscoped().Delete(&c) })

	// Case number must match the CASE-NNNNN pattern
	assert.Regexp(t, `^CASE-\d{5}$`, c.CaseNumber, "case number must match CASE-NNNNN format")

	// Find by ID
	var found Case
	require.NoError(t, db.WithContext(ctx).First(&found, "id = ?", c.ID).Error)
	assert.Equal(t, c.ID, found.ID)
	assert.Equal(t, c.TenantID, found.TenantID)
	assert.Equal(t, c.CaseNumber, found.CaseNumber)
	assert.Equal(t, c.Title, found.Title)
	assert.Equal(t, c.Status, found.Status)
	assert.Equal(t, c.Priority, found.Priority)
	assert.False(t, found.CreatedAt.IsZero())
}

// TestCaseRepository_ListWithFilters creates three cases with different statuses
// for the same tenant and verifies that a status filter returns only matching rows.
func TestCaseRepository_ListWithFilters(t *testing.T) {
	ensureWorkflowSchema(t)
	db := wfDB()
	ctx := context.Background()

	tenantID := newTenantID()
	createdByID := newUserID()

	openCase := createTestCase(t, tenantID, "open", createdByID)
	_ = createTestCase(t, tenantID, "in_progress", createdByID)
	closedCase := createTestCase(t, tenantID, "closed", createdByID)
	_ = closedCase // used for Cleanup

	// List only "open" cases for this tenant
	var openCases []Case
	err := db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, "open").
		Find(&openCases).Error
	require.NoError(t, err)
	require.Len(t, openCases, 1, "should find exactly 1 open case")
	assert.Equal(t, openCase.ID, openCases[0].ID)

	// List all cases for the tenant (no status filter)
	var allCases []Case
	err = db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Find(&allCases).Error
	require.NoError(t, err)
	assert.Len(t, allCases, 3, "should find all 3 cases")

	// Filter by in_progress
	var inProgressCases []Case
	err = db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, "in_progress").
		Find(&inProgressCases).Error
	require.NoError(t, err)
	assert.Len(t, inProgressCases, 1)
	assert.Equal(t, "in_progress", inProgressCases[0].Status)
}

// TestCaseRepository_RecordTransition transitions a case from "open" to
// "in_progress" by inserting a CaseTransition record and updating the case
// status, then verifies the full transition history.
func TestCaseRepository_RecordTransition(t *testing.T) {
	ensureWorkflowSchema(t)
	db := wfDB()
	ctx := context.Background()

	tenantID := newTenantID()
	actorID := newUserID()

	c := createTestCase(t, tenantID, "open", actorID)

	// Perform transition: open → in_progress
	transition := CaseTransition{
		ID:             uuid.New().String(),
		CaseID:         c.ID,
		TenantID:       tenantID,
		FromStatus:     "open",
		ToStatus:       "in_progress",
		ActorID:        actorID,
		Comment:        "Starting investigation",
		TransitionedAt: time.Now().UTC(),
	}
	require.NoError(t, db.WithContext(ctx).Create(&transition).Error)

	// Update case status
	require.NoError(t, db.WithContext(ctx).
		Model(&Case{}).
		Where("id = ?", c.ID).
		Update("status", "in_progress").Error)

	// Verify transition history
	var transitions []CaseTransition
	err := db.WithContext(ctx).
		Where("case_id = ? AND tenant_id = ?", c.ID, tenantID).
		Order("transitioned_at ASC").
		Find(&transitions).Error
	require.NoError(t, err)
	require.Len(t, transitions, 1)
	assert.Equal(t, "open", transitions[0].FromStatus)
	assert.Equal(t, "in_progress", transitions[0].ToStatus)
	assert.Equal(t, actorID, transitions[0].ActorID)
	assert.Equal(t, "Starting investigation", transitions[0].Comment)

	// Verify case status updated
	var updated Case
	require.NoError(t, db.WithContext(ctx).First(&updated, "id = ?", c.ID).Error)
	assert.Equal(t, "in_progress", updated.Status)
}

// TestTaskRepository_CreateAndComplete creates a task for a case, marks it
// completed, and verifies that CompletedAt is set and status changes.
func TestTaskRepository_CreateAndComplete(t *testing.T) {
	ensureWorkflowSchema(t)
	db := wfDB()
	ctx := context.Background()

	tenantID := newTenantID()
	createdByID := newUserID()
	assigneeID := newUserID()

	c := createTestCase(t, tenantID, "open", createdByID)

	task := Task{
		ID:         uuid.New().String(),
		CaseID:     c.ID,
		TenantID:   tenantID,
		Title:      "Investigate root cause",
		AssigneeID: &assigneeID,
		Status:     "pending",
	}
	require.NoError(t, db.WithContext(ctx).Create(&task).Error)
	t.Cleanup(func() { db.Unscoped().Delete(&task) })

	// Verify task is pending
	var pending Task
	require.NoError(t, db.WithContext(ctx).First(&pending, "id = ?", task.ID).Error)
	assert.Equal(t, "pending", pending.Status)
	assert.Nil(t, pending.CompletedAt, "CompletedAt should be nil for pending task")

	// Complete the task
	completedAt := time.Now().UTC()
	err := db.WithContext(ctx).
		Model(&Task{}).
		Where("id = ?", task.ID).
		Updates(map[string]interface{}{
			"status":       "completed",
			"completed_at": completedAt,
		}).Error
	require.NoError(t, err)

	// Verify completion
	var completed Task
	require.NoError(t, db.WithContext(ctx).First(&completed, "id = ?", task.ID).Error)
	assert.Equal(t, "completed", completed.Status)
	require.NotNil(t, completed.CompletedAt, "CompletedAt must be set after completion")
	assert.WithinDuration(t, completedAt, *completed.CompletedAt, 2*time.Second)
}

// TestCaseCounter_Increments verifies that sequential case creation for a
// given tenant results in monotonically increasing case numbers.
func TestCaseCounter_Increments(t *testing.T) {
	ensureWorkflowSchema(t)
	db := wfDB()
	ctx := context.Background()

	tenantID := newTenantID()
	createdByID := newUserID()

	// Clean up counter row after test
	t.Cleanup(func() {
		db.Unscoped().Where("tenant_id = ?", tenantID).Delete(&CaseCounter{})
	})

	const numCases = 5
	caseNumbers := make([]string, 0, numCases)

	for i := 0; i < numCases; i++ {
		cn := nextCaseNumber(t, db, tenantID)
		c := Case{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			CaseNumber:  cn,
			Title:       fmt.Sprintf("Counter test case %d", i+1),
			Status:      "open",
			Priority:    "low",
			CreatedByID: createdByID,
		}
		require.NoError(t, db.WithContext(ctx).Create(&c).Error)
		t.Cleanup(func() { db.Unscoped().Delete(&c) })
		caseNumbers = append(caseNumbers, cn)
	}

	// All case numbers must be unique
	seen := make(map[string]bool, numCases)
	for _, cn := range caseNumbers {
		assert.False(t, seen[cn], "duplicate case number: %s", cn)
		seen[cn] = true
	}

	// Each successive case number must be lexicographically greater (CASE-NNNNN)
	for i := 1; i < len(caseNumbers); i++ {
		assert.Greater(t, caseNumbers[i], caseNumbers[i-1],
			"case numbers must increment: %s should be > %s", caseNumbers[i], caseNumbers[i-1])
	}
}
