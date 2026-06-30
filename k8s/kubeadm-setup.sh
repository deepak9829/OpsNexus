#!/usr/bin/env bash
# kubeadm-setup.sh — Single-node OpsNexus Kubernetes cluster bootstrap
# Usage: sudo bash k8s/kubeadm-setup.sh  (must be run from repo root)
set -euo pipefail

###############################################################################
# Colour helpers
###############################################################################
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

###############################################################################
# 1. Prerequisites check
###############################################################################
info "==> Step 1: Checking prerequisites"

check_cmd() {
  if ! command -v "$1" &>/dev/null; then
    error "'$1' is not installed or not on PATH. Please install it and re-run."
  fi
  info "  Found: $(command -v "$1")"
}

check_cmd kubectl
check_cmd kubeadm
check_cmd docker

# Verify docker daemon is running
if ! docker info &>/dev/null; then
  error "Docker daemon is not running. Start it and re-run."
fi

info "All prerequisites satisfied."

###############################################################################
# 2. Initialise single-node kubeadm cluster
###############################################################################
info "==> Step 2: Initialising kubeadm single-node cluster"

# Skip if already initialised (idempotent re-runs)
if [ -f /etc/kubernetes/admin.conf ]; then
  warn "Kubernetes cluster appears to already be initialised (/etc/kubernetes/admin.conf exists). Skipping kubeadm init."
else
  kubeadm init \
    --pod-network-cidr=10.244.0.0/16 \
    --ignore-preflight-errors=NumCPU,Mem \
    | tee /tmp/kubeadm-init.log

  info "kubeadm init completed."
fi

###############################################################################
# 3. Configure kubectl for the current user
###############################################################################
info "==> Step 3: Configuring kubectl"

KUBE_CONFIG_DIR="${HOME}/.kube"
mkdir -p "${KUBE_CONFIG_DIR}"

if [ -f /etc/kubernetes/admin.conf ]; then
  cp -f /etc/kubernetes/admin.conf "${KUBE_CONFIG_DIR}/config"
  # If running as root (common with kubeadm), also set for SUDO_USER if present
  if [ -n "${SUDO_USER:-}" ]; then
    SUDO_HOME=$(getent passwd "${SUDO_USER}" | cut -d: -f6)
    mkdir -p "${SUDO_HOME}/.kube"
    cp -f /etc/kubernetes/admin.conf "${SUDO_HOME}/.kube/config"
    chown -R "${SUDO_USER}:${SUDO_USER}" "${SUDO_HOME}/.kube"
    info "Copied kubeconfig to ${SUDO_HOME}/.kube/config as well."
  fi
else
  warn "/etc/kubernetes/admin.conf not found — kubectl may not be configured."
fi

export KUBECONFIG="${KUBE_CONFIG_DIR}/config"
info "KUBECONFIG set to ${KUBECONFIG}"

###############################################################################
# 4. Install Flannel CNI
###############################################################################
info "==> Step 4: Installing Flannel CNI"

FLANNEL_MANIFEST="https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml"
kubectl apply -f "${FLANNEL_MANIFEST}"

info "Flannel CNI applied. Waiting 20s for network to settle..."
sleep 20

###############################################################################
# 5. Untaint control-plane so pods can run on it
###############################################################################
info "==> Step 5: Removing control-plane taint"

# The taint may not exist on newer Kubernetes versions — suppress errors
kubectl taint nodes --all node-role.kubernetes.io/control-plane- \
  --ignore-not-found=true 2>/dev/null || true

kubectl taint nodes --all node-role.kubernetes.io/master- \
  --ignore-not-found=true 2>/dev/null || true

info "Control-plane taint removed (or was not present)."

###############################################################################
# 6. Create host data directories for PersistentVolumes
###############################################################################
info "==> Step 6: Creating host data directories for PersistentVolumes"

for DIR in /data/opsnexus/mysql /data/opsnexus/mongodb /data/opsnexus/localstack; do
  if [ ! -d "${DIR}" ]; then
    mkdir -p "${DIR}"
    info "  Created ${DIR}"
  else
    warn "  ${DIR} already exists, skipping."
  fi
  chmod 777 "${DIR}"
done

info "Host directories ready."

###############################################################################
# 7. Build Docker images for the OpsNexus services
###############################################################################
info "==> Step 7: Building OpsNexus Docker images"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVICES_DIR="${REPO_ROOT}/services"

SERVICES=(
  "auth-service:opsnexus-auth-service:latest"
  "tenant-service:opsnexus-tenant-service:latest"
  "workflow-service:opsnexus-workflow-service:latest"
  "document-service:opsnexus-document-service:latest"
  "notification-service:opsnexus-notification-service:latest"
)

for ENTRY in "${SERVICES[@]}"; do
  SVC_DIR="${ENTRY%%:*}"
  IMG="${ENTRY#*:}"
  SVC_PATH="${SERVICES_DIR}/${SVC_DIR}"

  if [ -d "${SVC_PATH}" ]; then
    info "  Building image ${IMG} from ${SVC_PATH}"
    docker build -t "${IMG}" "${SVC_PATH}"
  else
    warn "  Service directory ${SVC_PATH} not found — skipping build for ${IMG}."
    warn "  Build it manually with: docker build -t ${IMG} <path>"
  fi
done

info "Docker image builds complete."

###############################################################################
# 8. Apply Kustomize overlay
###############################################################################
info "==> Step 8: Applying Kustomize overlay (overlays/local)"

K8S_DIR="${REPO_ROOT}/k8s"

if [ ! -d "${K8S_DIR}/overlays/local" ]; then
  error "Kustomize overlay not found at ${K8S_DIR}/overlays/local"
fi

kubectl apply -k "${K8S_DIR}/overlays/local"
info "Kustomize overlay applied."

###############################################################################
# 9. Wait for all pods to be ready
###############################################################################
info "==> Step 9: Waiting for all pods in namespace 'opsnexus' to be ready"

NAMESPACE="opsnexus"
TIMEOUT=300   # seconds
INTERVAL=10

info "  Polling every ${INTERVAL}s (timeout: ${TIMEOUT}s)..."
ELAPSED=0

while true; do
  TOTAL=$(kubectl get pods -n "${NAMESPACE}" --no-headers 2>/dev/null | wc -l | tr -d ' ')
  READY=$(kubectl get pods -n "${NAMESPACE}" --no-headers 2>/dev/null \
    | grep -c '1/1\|2/2\|3/3\|Running' || true)

  NOT_READY=$(kubectl get pods -n "${NAMESPACE}" --no-headers 2>/dev/null \
    | grep -v 'Running\|Completed' | grep -v '^$' || true)

  if [ "${TOTAL}" -gt 0 ] && [ -z "${NOT_READY}" ]; then
    info "All ${TOTAL} pods in namespace '${NAMESPACE}' are ready."
    break
  fi

  if [ "${ELAPSED}" -ge "${TIMEOUT}" ]; then
    warn "Timeout reached after ${TIMEOUT}s. Some pods may not be ready yet."
    kubectl get pods -n "${NAMESPACE}"
    break
  fi

  info "  ${ELAPSED}s elapsed — ${READY}/${TOTAL} pods ready. Waiting..."
  sleep "${INTERVAL}"
  ELAPSED=$((ELAPSED + INTERVAL))
done

###############################################################################
# Done
###############################################################################
echo ""
info "=========================================="
info "  OpsNexus cluster bootstrap complete!"
info "=========================================="
echo ""
info "Cluster status:"
kubectl get nodes
echo ""
info "Pods in namespace '${NAMESPACE}':"
kubectl get pods -n "${NAMESPACE}"
echo ""
info "Services in namespace '${NAMESPACE}':"
kubectl get services -n "${NAMESPACE}"
echo ""
info "NodePort endpoints (access via <node-ip>:<nodeport>):"
echo "  auth-service        -> :30081"
echo "  tenant-service      -> :30082"
echo "  workflow-service    -> :30083"
echo "  document-service    -> :30084"
echo "  notification-service -> :30085"
echo ""
warn "REMINDER: Update 'localstack-auth-token' in k8s/base/secrets.yaml with a real token before deploying to a shared environment."
