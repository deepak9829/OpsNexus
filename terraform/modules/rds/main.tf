data "aws_caller_identity" "current" {}

# Security Group - owned by this module
resource "aws_security_group" "rds" {
  name        = "${var.project_name}-${var.environment}-rds-sg"
  description = "Allow MySQL from EKS nodes"
  vpc_id      = var.vpc_id

  ingress {
    description     = "MySQL from EKS nodes"
    from_port       = 3306
    to_port         = 3306
    protocol        = "tcp"
    security_groups = [var.eks_node_sg_id]
  }

  egress {
    description = "No outbound"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = []
  }

  tags = { Name = "${var.project_name}-${var.environment}-rds-sg" }

  lifecycle {
    create_before_destroy = true
  }
}

# KMS key for RDS storage encryption
resource "aws_kms_key" "rds" {
  description             = "RDS encryption for ${var.project_name}-${var.environment}"
  deletion_window_in_days = 7
  enable_key_rotation     = true
  tags                    = { Name = "${var.project_name}-${var.environment}-rds-kms" }
}

resource "aws_kms_alias" "rds" {
  name          = "alias/${var.project_name}-${var.environment}-rds"
  target_key_id = aws_kms_key.rds.key_id
}

# Random password - never stored in state in plaintext
resource "random_password" "db" {
  length           = 32
  special          = true
  override_special = "!#%&*()-_=+[]{}<>:?"
}

# Secrets Manager
resource "aws_secretsmanager_secret" "rds" {
  name                    = "${var.project_name}/${var.environment}/rds/mysql"
  description             = "MySQL credentials for ${var.project_name}-${var.environment}"
  kms_key_id              = aws_kms_key.rds.arn
  recovery_window_in_days = var.environment == "prod" ? 30 : 0
  tags                    = { Name = "${var.project_name}-${var.environment}-rds-secret" }
}

resource "aws_secretsmanager_secret_version" "rds" {
  secret_id = aws_secretsmanager_secret.rds.id
  secret_string = jsonencode({
    username = "admin"
    password = random_password.db.result
    host     = aws_db_instance.mysql.address
    port     = 3306
    dbname   = "opsnexus"
    engine   = "mysql"
  })
}

# Parameter group for MySQL 8.0 utf8mb4
resource "aws_db_parameter_group" "mysql8" {
  name        = "${var.project_name}-${var.environment}-mysql8"
  family      = "mysql8.0"
  description = "MySQL 8.0 parameters for ${var.project_name}-${var.environment}"

  parameter {
    name  = "character_set_server"
    value = "utf8mb4"
  }
  parameter {
    name  = "collation_server"
    value = "utf8mb4_unicode_ci"
  }
  parameter {
    name  = "slow_query_log"
    value = "1"
  }
  parameter {
    name  = "long_query_time"
    value = "2"
  }

  tags = { Name = "${var.project_name}-${var.environment}-mysql8-params" }
}

# Subnet group - passed in from subnets module
resource "aws_db_subnet_group" "main" {
  name        = "${var.project_name}-${var.environment}-rds-subnet-group"
  description = "RDS subnet group for ${var.project_name}-${var.environment}"
  subnet_ids  = var.private_db_subnet_ids
  tags        = { Name = "${var.project_name}-${var.environment}-rds-subnet-group" }
}

# RDS MySQL instance
resource "aws_db_instance" "mysql" {
  identifier = "${var.project_name}-${var.environment}-mysql"

  # Engine
  engine         = "mysql"
  engine_version = "8.0"
  instance_class = var.db_instance_class

  # Storage
  allocated_storage     = var.db_allocated_storage
  max_allocated_storage = var.db_max_allocated_storage
  storage_type          = "gp3"
  storage_encrypted     = true
  kms_key_id            = aws_kms_key.rds.arn

  # Database
  db_name  = "opsnexus"
  username = "admin"
  password = random_password.db.result

  # Network
  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  publicly_accessible    = false
  multi_az               = var.multi_az

  # Parameters
  parameter_group_name = aws_db_parameter_group.mysql8.name

  # Backup
  backup_retention_period = var.backup_retention_period
  backup_window           = "03:00-04:00"
  maintenance_window      = "Mon:04:00-Mon:05:00"
  copy_tags_to_snapshot   = true

  # Finalization
  deletion_protection       = var.deletion_protection
  skip_final_snapshot       = var.skip_final_snapshot
  final_snapshot_identifier = var.skip_final_snapshot ? null : "${var.project_name}-${var.environment}-mysql-final-snapshot"

  # Features
  enabled_cloudwatch_logs_exports     = ["error", "general", "slowquery"]
  performance_insights_enabled        = var.enable_performance_insights
  iam_database_authentication_enabled = true
  auto_minor_version_upgrade          = true
  apply_immediately                   = var.environment != "prod"

  tags = { Name = "${var.project_name}-${var.environment}-mysql" }

  lifecycle {
    ignore_changes = [password]
  }
}
