# Database Standards — OpsNexus

Standards for MySQL, MongoDB, and DynamoDB usage across all services. Each database has a designated service owner; only that service reads and writes to it directly.

---

## 1. MySQL — Auth, Case, and Workflow Services

MySQL 8 is used for structured relational data with strong consistency requirements.

### Primary Keys

All primary keys are `VARCHAR(36)` UUIDs generated in the application layer before insertion. Never use `AUTO_INCREMENT` integer PKs.

```sql
-- CORRECT
CREATE TABLE cases (
    id          VARCHAR(36)  NOT NULL,
    tenant_id   VARCHAR(36)  NOT NULL,
    title       VARCHAR(200) NOT NULL,
    ...
    PRIMARY KEY (id)
);

-- WRONG
CREATE TABLE cases (
    id INT AUTO_INCREMENT PRIMARY KEY,
    ...
);
```

Rationale: UUIDs allow the application to assign IDs before the DB write, which simplifies event sourcing, distributed systems, and test data setup. They also don't leak row counts to clients.

### Timestamps

Every table with mutable data has both timestamps:

```sql
created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
```

### Character Set and Collation

All tables and columns use `utf8mb4` charset with `utf8mb4_unicode_ci` collation. This handles all Unicode including emoji. Set it at the table level:

```sql
CREATE TABLE users (
    ...
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

Alternatively, set it as the database default — but be explicit in every `CREATE TABLE` to avoid inheriting wrong defaults from the MySQL server config.

### Migrations

Sequential numbered SQL files are the source of truth for the database schema:

```
services/{name}/
  migrations/
    001_create_users.sql
    002_create_sessions.sql
    003_add_users_avatar_url.sql
    004_create_roles.sql
```

Rules:
- Files are never modified after they've been applied to any environment
- New changes get a new file with the next sequential number
- Each file is idempotent where possible (`CREATE TABLE IF NOT EXISTS`, `ADD COLUMN IF NOT EXISTS`)
- In development, GORM AutoMigrate may be used for convenience, but the numbered SQL files are the source of truth for production
- Migration runner (`migrate` CLI or a simple Go startup script) applies pending migrations on service startup

**Never DROP columns or tables in the same migration as a code change that still uses that column.** The safe sequence:
1. Deploy code that ignores the column (reads but doesn't require it)
2. Deploy migration that removes the column
3. Deploy code that no longer references the column at all

### Indexes

Index every column that appears in a `WHERE` clause:
- `tenant_id` — every table with tenant data
- Status columns: `status`, `state`
- Foreign key columns
- Columns used in ORDER BY on large tables

```sql
-- Required indexes
CREATE INDEX idx_cases_tenant_id ON cases (tenant_id);
CREATE INDEX idx_cases_status ON cases (status);
CREATE INDEX idx_cases_tenant_status ON cases (tenant_id, status); -- composite for common query
CREATE INDEX idx_cases_assignee ON cases (assigned_to);
```

### Multi-Tenant Schema Pattern

Every table that holds tenant-specific data must have `tenant_id`:

```sql
CREATE TABLE cases (
    id          VARCHAR(36)  NOT NULL,
    tenant_id   VARCHAR(36)  NOT NULL,  -- REQUIRED
    title       VARCHAR(200) NOT NULL,
    status      ENUM('open','in_progress','resolved','closed') NOT NULL DEFAULT 'open',
    priority    ENUM('low','medium','high','critical') NOT NULL DEFAULT 'medium',
    assigned_to VARCHAR(36)  NULL,
    created_by  VARCHAR(36)  NOT NULL,
    created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_tenant (tenant_id),
    INDEX idx_tenant_status (tenant_id, status),
    INDEX idx_assigned (assigned_to)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

---

## 2. MongoDB — Document Service

MongoDB is used for the Document Service which stores forms, submissions, and document metadata. Schema is flexible but conventions are strict.

### Collections

| Collection | Owner Service | Purpose |
|-----------|-------------|---------|
| `forms` | Document Service | Form template definitions |
| `form_submissions` | Document Service | Filled-in form instances |
| `documents` | Document Service | Document metadata (not content) |
| `document_versions` | Document Service | Version history of documents |

### Document Structure

Every document in every collection must have:

```go
type BaseDocument struct {
    ID        string    `bson:"_id"`            // UUID string, not ObjectID
    TenantID  string    `bson:"tenantId"`        // Required, always indexed
    CreatedAt time.Time `bson:"createdAt"`
    UpdatedAt time.Time `bson:"updatedAt"`
}
```

Use `bson:"field,omitempty"` for optional fields to avoid storing null values for unset fields.

### Indexes

Create indexes at service startup via `EnsureIndexes()`:

```go
func (r *MongoDocumentRepo) EnsureIndexes(ctx context.Context) error {
    models := []mongo.IndexModel{
        {
            Keys:    bson.D{{Key: "tenantId", Value: 1}},
            Options: options.Index().SetBackground(true),
        },
        {
            Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "createdAt", Value: -1}},
            Options: options.Index().SetBackground(true),
        },
        {
            Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "status", Value: 1}},
            Options: options.Index().SetBackground(true),
        },
    }
    _, err := r.collection.Indexes().CreateMany(ctx, models)
    return err
}
```

Never let indexes be created lazily or manually. `EnsureIndexes()` is called during service startup.

### Write Concerns

In production, use majority write concern for any write that must be durable:

```go
opts := options.Collection().SetWriteConcern(writeconcern.Majority())
collection := db.Collection("documents", opts)
```

For reads that serve real-time user requests, use primary read preference. For bulk or reporting reads, ReadPreference can be Secondary.

### File Content Storage

Binary document content (PDFs, images, etc.) is never stored in MongoDB. Only metadata is stored:
- Document name, type, size
- `storageKey` — the S3 object key where the actual bytes live
- Version information, access controls, tenant association

The Document Service generates presigned S3 URLs for upload/download via the application tier. The browser uploads/downloads directly to/from S3 using short-lived presigned URLs.

---

## 3. DynamoDB — Notification Service

DynamoDB is used for the Notification Service due to its high write throughput and flexible querying of notification streams.

### Table Design

Use purpose-built tables, one per primary access pattern. Single-table design is acceptable when access patterns are well-defined, but do not use it speculatively.

```
notifications table:
  PK: tenantId#userId  (e.g., "tenant-1#user-42")
  SK: timestamp#notificationId  (e.g., "2024-01-15T10:30:00Z#notif-abc")

notification_preferences table:
  PK: tenantId
  SK: userId
```

### Access Patterns

Always use `KeyConditionExpression`, never `Scan`:

```go
// CORRECT: query by PK (and optional SK prefix)
result, err := client.Query(ctx, &dynamodb.QueryInput{
    TableName:              aws.String("notifications"),
    KeyConditionExpression: aws.String("pk = :pk AND begins_with(sk, :prefix)"),
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":pk":     &types.AttributeValueMemberS{Value: "tenant-1#user-42"},
        ":prefix": &types.AttributeValueMemberS{Value: "2024-01-"},
    },
    Limit: aws.Int32(50),
})

// WRONG: full table scan
result, err := client.Scan(ctx, &dynamodb.ScanInput{
    TableName: aws.String("notifications"),
})
```

`Scan` is only acceptable for administrative tasks (one-time data migrations) and never in the application code path.

### Timestamps

Store timestamps as ISO 8601 strings (`2024-01-15T10:30:00Z`), not Unix epoch numbers. This makes the sort key human-readable and debuggable.

### Billing Mode

Use `PAY_PER_REQUEST` (on-demand) in development and staging. Switch to `PROVISIONED` with Auto Scaling in production once traffic patterns are known.

### GSIs for Non-Primary Access Patterns

When a query pattern can't be served by the primary key:

```go
// GSI: status-index
// PK: tenantId
// SK: status#createdAt
// Allows querying: "all unread notifications for tenant X"
```

Define GSIs in the Terraform/CloudFormation definition, not ad-hoc.

### Conditional Writes for Optimistic Locking

When two concurrent writers might overwrite each other:

```go
_, err := client.PutItem(ctx, &dynamodb.PutItemInput{
    TableName: aws.String("notification_preferences"),
    Item:      item,
    ConditionExpression: aws.String("attribute_not_exists(pk) OR version = :expectedVersion"),
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":expectedVersion": &types.AttributeValueMemberN{Value: strconv.Itoa(expectedVersion)},
    },
})
if err != nil {
    var ccf *types.ConditionalCheckFailedException
    if errors.As(err, &ccf) {
        return domain.ErrConflict // another writer won; caller should retry
    }
    return fmt.Errorf("saving preferences: %w", err)
}
```

---

## 4. General Database Rules

### No business logic in the database

No stored procedures, no triggers that implement business rules, no database-level computed columns for business logic. The application layer owns business logic. The database stores and retrieves data.

Acceptable DB-level concerns: constraints (NOT NULL, UNIQUE, FK), default values, timestamps, indexes.

### No raw SQL in application code

SQL belongs in migration files. Application code uses GORM or the official drivers via repository implementations. One exception: complex analytical queries that GORM cannot express cleanly may use `db.Raw()`, but must be isolated in a clearly-named repository method with a comment explaining why.

### Nullable foreign keys

When a foreign key column is nullable (e.g., `assigned_to VARCHAR(36) NULL`), always handle the nil case explicitly in Go:

```go
// CORRECT: use a pointer type or sql.NullString
type CaseModel struct {
    ID         string
    TenantID   string
    AssignedTo *string `gorm:"column:assigned_to"` // nil when unassigned
}

// When mapping to domain:
func toDomainCase(m CaseModel) *domain.Case {
    c := &domain.Case{
        ID:       m.ID,
        TenantID: m.TenantID,
    }
    if m.AssignedTo != nil {
        c.AssigneeID = *m.AssignedTo
    }
    return c
}
```

### Slow query logging

Configure the database to log queries taking longer than 100ms. In GORM:

```go
newLogger := logger.New(
    log.New(os.Stdout, "\r\n", log.LstdFlags),
    logger.Config{
        SlowThreshold:             100 * time.Millisecond,
        LogLevel:                  logger.Warn,
        IgnoreRecordNotFoundError: true,
    },
)
db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: newLogger})
```

Slow queries in staging must be investigated and either optimized or have an index added before they reach production.
