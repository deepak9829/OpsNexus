-- ============================================================
-- OpsNexus MySQL Initialization Script
-- Runs automatically on first container startup via
-- /docker-entrypoint-initdb.d/init.sql
-- ============================================================

-- Auth Service Database
CREATE DATABASE IF NOT EXISTS auth_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Tenant Service Database
CREATE DATABASE IF NOT EXISTS tenant_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Workflow Service Database
CREATE DATABASE IF NOT EXISTS workflow_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Auth Service User
CREATE USER IF NOT EXISTS 'auth_user'@'%' IDENTIFIED BY 'auth_pass';
GRANT ALL PRIVILEGES ON auth_db.* TO 'auth_user'@'%';

-- Tenant Service User
CREATE USER IF NOT EXISTS 'tenant_user'@'%' IDENTIFIED BY 'tenant_pass';
GRANT ALL PRIVILEGES ON tenant_db.* TO 'tenant_user'@'%';

-- Workflow Service User
CREATE USER IF NOT EXISTS 'workflow_user'@'%' IDENTIFIED BY 'workflow_pass';
GRANT ALL PRIVILEGES ON workflow_db.* TO 'workflow_user'@'%';

FLUSH PRIVILEGES;
