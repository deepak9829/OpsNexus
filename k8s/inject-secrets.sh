#!/usr/bin/env bash
# Injects secrets from .env or environment variables into the cluster.
# Run this AFTER the namespace is created but BEFORE applying the full kustomize overlay.
# This script never writes tokens to any file tracked by git.
set -e

NAMESPACE=opsnexus
ENV_FILE="$(dirname "$0")/../.env"

if [ -f "$ENV_FILE" ]; then
  # shellcheck disable=SC1090
  set -o allexport && source "$ENV_FILE" && set +o allexport
fi

: "${LOCALSTACK_AUTH_TOKEN:?Please set LOCALSTACK_AUTH_TOKEN in .env or environment}"
: "${MYSQL_ROOT_PASSWORD:=rootpassword}"
: "${JWT_SECRET:=change-me-in-production-use-32-chars}"

kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret generic opsnexus-secrets \
  --namespace="$NAMESPACE" \
  --from-literal=mysql-root-password="$MYSQL_ROOT_PASSWORD" \
  --from-literal=auth-db-password="auth_pass" \
  --from-literal=tenant-db-password="tenant_pass" \
  --from-literal=workflow-db-password="workflow_pass" \
  --from-literal=jwt-secret="$JWT_SECRET" \
  --from-literal=localstack-auth-token="$LOCALSTACK_AUTH_TOKEN" \
  --from-literal=aws-access-key-id="test" \
  --from-literal=aws-secret-access-key="test" \
  --save-config \
  --dry-run=client -o yaml | kubectl apply -f -

echo "Secret opsnexus-secrets applied to namespace $NAMESPACE"
