terraform {
  required_version = ">= 1.10.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# notifications table
resource "aws_dynamodb_table" "notifications" {
  name         = "${var.project_name}-${var.environment}-notifications"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "tenantId"
  range_key    = "notificationId"

  attribute {
    name = "tenantId"
    type = "S"
  }

  attribute {
    name = "notificationId"
    type = "S"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  point_in_time_recovery {
    enabled = true
  }

  server_side_encryption {
    enabled = true
  }

  tags = { Name = "${var.project_name}-${var.environment}-notifications" }
}

# audit_events table
resource "aws_dynamodb_table" "audit_events" {
  name         = "${var.project_name}-${var.environment}-audit-events"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "tenantId"
  range_key    = "eventId"

  attribute {
    name = "tenantId"
    type = "S"
  }

  attribute {
    name = "eventId"
    type = "S"
  }

  point_in_time_recovery {
    enabled = true
  }

  server_side_encryption {
    enabled = true
  }

  tags = { Name = "${var.project_name}-${var.environment}-audit-events" }
}
