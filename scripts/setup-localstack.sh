#!/bin/bash
# ============================================================
# setup-localstack.sh
# Creates required DynamoDB tables in LocalStack.
# Run after: make dev-up  (waits for LocalStack to be healthy)
# ============================================================
set -e

ENDPOINT=${AWS_ENDPOINT_URL:-http://localhost:4566}
REGION=${AWS_REGION:-us-east-1}

echo "Creating DynamoDB tables in LocalStack at $ENDPOINT ..."

# ------------------------------------------------------------
# notifications table
# Partition key : tenantId       (String)
# Sort key      : notificationId (String)
# ------------------------------------------------------------
aws --endpoint-url="$ENDPOINT" dynamodb create-table \
  --table-name notifications \
  --attribute-definitions \
    AttributeName=tenantId,AttributeType=S \
    AttributeName=notificationId,AttributeType=S \
  --key-schema \
    AttributeName=tenantId,KeyType=HASH \
    AttributeName=notificationId,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --region "$REGION" 2>/dev/null || echo "  -> notifications table already exists, skipping."

# ------------------------------------------------------------
# audit_events table
# Partition key : tenantId (String)
# Sort key      : eventId  (String)
# ------------------------------------------------------------
aws --endpoint-url="$ENDPOINT" dynamodb create-table \
  --table-name audit_events \
  --attribute-definitions \
    AttributeName=tenantId,AttributeType=S \
    AttributeName=eventId,AttributeType=S \
  --key-schema \
    AttributeName=tenantId,KeyType=HASH \
    AttributeName=eventId,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --region "$REGION" 2>/dev/null || echo "  -> audit_events table already exists, skipping."

echo "DynamoDB tables created successfully."

# Verify
echo ""
echo "Current tables in LocalStack:"
aws --endpoint-url="$ENDPOINT" dynamodb list-tables --region "$REGION"
