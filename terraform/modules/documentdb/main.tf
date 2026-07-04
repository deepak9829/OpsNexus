data "aws_region" "current" {}
data "aws_caller_identity" "current" {}

# ─── KMS key for DocumentDB encryption ────────────────────────────────────────
resource "aws_kms_key" "docdb" {
  description             = "DocumentDB encryption for ${var.project_name}-${var.environment}"
  deletion_window_in_days = 7
  enable_key_rotation     = true
  tags                    = { Name = "${var.project_name}-${var.environment}-docdb-kms" }
}

resource "aws_kms_alias" "docdb" {
  name          = "alias/${var.project_name}-${var.environment}-docdb"
  target_key_id = aws_kms_key.docdb.key_id
}

# ─── Random password ──────────────────────────────────────────────────────────
resource "random_password" "docdb" {
  length           = 32
  special          = true
  override_special = "!#%&*()-_=+[]{}<>:?"
}

# ─── Security group ───────────────────────────────────────────────────────────
resource "aws_security_group" "docdb" {
  name        = "${var.project_name}-${var.environment}-docdb-sg"
  description = "DocumentDB - allow EKS nodes on port 27017"
  vpc_id      = var.vpc_id

  ingress {
    description     = "MongoDB from EKS nodes"
    from_port       = 27017
    to_port         = 27017
    protocol        = "tcp"
    security_groups = [var.eks_node_sg_id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "${var.project_name}-${var.environment}-docdb-sg" }
}

# ─── Subnet group ─────────────────────────────────────────────────────────────
resource "aws_docdb_subnet_group" "main" {
  name       = "${var.project_name}-${var.environment}-docdb-subnet-group"
  subnet_ids = var.private_db_subnet_ids
  tags       = { Name = "${var.project_name}-${var.environment}-docdb-subnet-group" }
}

# ─── Cluster parameter group ──────────────────────────────────────────────────
resource "aws_docdb_cluster_parameter_group" "main" {
  family      = "docdb5.0"
  name        = "${var.project_name}-${var.environment}-docdb-params"
  description = "DocumentDB cluster parameters for ${var.project_name}-${var.environment}"

  parameter {
    name  = "tls"
    value = "enabled"
  }

  tags = { Name = "${var.project_name}-${var.environment}-docdb-params" }
}

# ─── DocumentDB cluster ───────────────────────────────────────────────────────
resource "aws_docdb_cluster" "main" {
  cluster_identifier              = "${var.project_name}-${var.environment}-docdb"
  engine                          = "docdb"
  engine_version                  = var.engine_version
  master_username                 = "docdbadmin"
  master_password                 = random_password.docdb.result
  db_subnet_group_name            = aws_docdb_subnet_group.main.name
  vpc_security_group_ids          = [aws_security_group.docdb.id]
  db_cluster_parameter_group_name = aws_docdb_cluster_parameter_group.main.name
  storage_encrypted               = true
  kms_key_id                      = aws_kms_key.docdb.arn
  deletion_protection             = var.deletion_protection
  skip_final_snapshot             = var.skip_final_snapshot
  backup_retention_period         = var.backup_retention_period
  preferred_backup_window         = "03:00-04:00"
  preferred_maintenance_window    = "mon:04:00-mon:05:00"

  tags = { Name = "${var.project_name}-${var.environment}-docdb" }

  lifecycle {
    ignore_changes = [master_password]
  }
}

# ─── DocumentDB instances ─────────────────────────────────────────────────────
resource "aws_docdb_cluster_instance" "main" {
  count              = var.instance_count
  identifier         = "${var.project_name}-${var.environment}-docdb-${count.index}"
  cluster_identifier = aws_docdb_cluster.main.id
  instance_class     = var.instance_class

  tags = { Name = "${var.project_name}-${var.environment}-docdb-${count.index}" }
}

# ─── Store credentials in Secrets Manager ─────────────────────────────────────
resource "aws_secretsmanager_secret" "docdb" {
  name                    = "${var.project_name}/${var.environment}/docdb"
  description             = "DocumentDB credentials for ${var.project_name}-${var.environment}"
  kms_key_id              = aws_kms_key.docdb.arn
  recovery_window_in_days = var.environment == "prod" ? 30 : 0
  tags                    = { Name = "${var.project_name}-${var.environment}-docdb-secret" }
}

resource "aws_secretsmanager_secret_version" "docdb" {
  secret_id = aws_secretsmanager_secret.docdb.id
  secret_string = jsonencode({
    host     = aws_docdb_cluster.main.endpoint
    port     = 27017
    username = aws_docdb_cluster.main.master_username
    password = random_password.docdb.result
    # TLS required - driver must verify with the Amazon RDS CA bundle.
    # Download: https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem
    connection_string = "mongodb://${aws_docdb_cluster.main.master_username}:${random_password.docdb.result}@${aws_docdb_cluster.main.endpoint}:27017/?tls=true&tlsCAFile=/etc/ssl/docdb/global-bundle.pem&replicaSet=rs0&readPreference=secondaryPreferred&retryWrites=false"
  })

  lifecycle {
    ignore_changes = [secret_string]
  }
}
