data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

# JWT signing secret — set real value after first apply:
#   aws secretsmanager put-secret-value \
#     --secret-id opsnexus/{env}/jwt-secret \
#     --secret-string "$(openssl rand -base64 48)"
resource "aws_secretsmanager_secret" "jwt" {
  name                    = "${var.project_name}/${var.environment}/jwt-secret"
  description             = "JWT signing secret for ${var.project_name}-${var.environment}"
  recovery_window_in_days = var.environment == "prod" ? 7 : 0
  tags                    = { Name = "${var.project_name}-${var.environment}-jwt-secret" }
}

resource "aws_secretsmanager_secret_version" "jwt" {
  secret_id     = aws_secretsmanager_secret.jwt.id
  secret_string = "PLACEHOLDER_REPLACE_AFTER_FIRST_APPLY"
  lifecycle {
    ignore_changes = [secret_string]
  }
}
