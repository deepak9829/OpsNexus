-- Workflow Service Schema
-- Migration: 001_create_workflow_schema

CREATE TABLE IF NOT EXISTS case_counters (
    tenant_id VARCHAR(36) NOT NULL,
    counter   BIGINT      NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS workflow_templates (
    id               VARCHAR(36)  NOT NULL,
    tenant_id        VARCHAR(36)  NOT NULL,
    name             VARCHAR(255) NOT NULL,
    description      TEXT,
    states           TEXT,
    transitions      TEXT,
    default_priority VARCHAR(20)  NOT NULL DEFAULT 'medium',
    sla_hours        INT          NOT NULL DEFAULT 0,
    created_at       DATETIME(3)  NOT NULL,
    updated_at       DATETIME(3)  NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_workflow_templates_tenant_id (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cases (
    id           VARCHAR(36)  NOT NULL,
    tenant_id    VARCHAR(36)  NOT NULL,
    case_number  VARCHAR(50)  NOT NULL,
    title        VARCHAR(500) NOT NULL,
    description  TEXT,
    status       VARCHAR(30)  NOT NULL DEFAULT 'new',
    priority     VARCHAR(20)  NOT NULL DEFAULT 'medium',
    assignee_id  VARCHAR(36),
    reporter_id  VARCHAR(36)  NOT NULL,
    workflow_id  VARCHAR(36),
    sla_due_at   DATETIME(3),
    sla_breached TINYINT(1)   NOT NULL DEFAULT 0,
    tags         TEXT,
    resolved_at  DATETIME(3),
    closed_at    DATETIME(3),
    created_at   DATETIME(3)  NOT NULL,
    updated_at   DATETIME(3)  NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_cases_tenant_case_number (tenant_id, case_number),
    INDEX idx_cases_tenant_id (tenant_id),
    INDEX idx_cases_status (status),
    INDEX idx_cases_priority (priority),
    INDEX idx_cases_assignee_id (assignee_id),
    INDEX idx_cases_reporter_id (reporter_id),
    INDEX idx_cases_workflow_id (workflow_id),
    INDEX idx_cases_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS case_transitions (
    id           VARCHAR(36)  NOT NULL,
    case_id      VARCHAR(36)  NOT NULL,
    from_status  VARCHAR(30)  NOT NULL,
    to_status    VARCHAR(30)  NOT NULL,
    reason       VARCHAR(500),
    performed_by VARCHAR(36)  NOT NULL,
    performed_at DATETIME(3)  NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_case_transitions_case_id (case_id),
    INDEX idx_case_transitions_performed_at (performed_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS tasks (
    id           VARCHAR(36)  NOT NULL,
    case_id      VARCHAR(36)  NOT NULL,
    tenant_id    VARCHAR(36)  NOT NULL,
    title        VARCHAR(500) NOT NULL,
    description  TEXT,
    status       VARCHAR(30)  NOT NULL DEFAULT 'todo',
    assignee_id  VARCHAR(36),
    due_at       DATETIME(3),
    completed_at DATETIME(3),
    created_at   DATETIME(3)  NOT NULL,
    updated_at   DATETIME(3)  NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_tasks_case_id (case_id),
    INDEX idx_tasks_tenant_id (tenant_id),
    INDEX idx_tasks_status (status),
    INDEX idx_tasks_assignee_id (assignee_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS comments (
    id         VARCHAR(36) NOT NULL,
    case_id    VARCHAR(36) NOT NULL,
    tenant_id  VARCHAR(36) NOT NULL,
    author_id  VARCHAR(36) NOT NULL,
    body       TEXT        NOT NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_comments_case_id (case_id),
    INDEX idx_comments_tenant_id (tenant_id),
    INDEX idx_comments_author_id (author_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
