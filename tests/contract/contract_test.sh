#!/usr/bin/env bash
# contract_test.sh — Run schemathesis contract tests for all OpsNexus services.
#
# Prerequisites:
#   - schemathesis is installed (pip install schemathesis)
#   - The target services are running (docker compose up, or started manually)
#   - Run from the repository root so that contracts/*.yaml is resolvable
#
# Usage:
#   bash tests/contract/contract_test.sh
#
# Environment variables:
#   CONTRACTS_DIR   Path to directory containing *.yaml contract files
#                   (default: contracts)
#   AUTH_TOKEN      Bearer token for authenticated endpoints (optional)
#   RESULTS_DIR     Directory to write JUnit XML reports (default: tests/contract/results)
#   MAX_EXAMPLES    Hypothesis max examples per operation (default: 50)
#   REQUEST_TIMEOUT Request timeout in seconds (default: 10)

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

CONTRACTS_DIR="${CONTRACTS_DIR:-contracts}"
RESULTS_DIR="${RESULTS_DIR:-tests/contract/results}"
MAX_EXAMPLES="${MAX_EXAMPLES:-50}"
REQUEST_TIMEOUT="${REQUEST_TIMEOUT:-10}"
AUTH_TOKEN="${AUTH_TOKEN:-}"

# Map each contract base name (filename without .yaml) to its service port.
declare -A SERVICE_PORTS=(
    ["auth"]=8081
    ["tenant"]=8082
    ["workflow"]=8083
    ["document"]=8084
    ["notification"]=8085
)

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # no color

info()    { echo -e "${NC}[INFO]  $*${NC}"; }
success() { echo -e "${GREEN}[PASS]  $*${NC}"; }
warn()    { echo -e "${YELLOW}[SKIP]  $*${NC}"; }
fail()    { echo -e "${RED}[FAIL]  $*${NC}"; }

# is_reachable <url> — returns 0 if the URL responds to a GET, non-zero otherwise.
is_reachable() {
    curl --silent --fail --max-time 3 "$1" > /dev/null 2>&1
}

# check_dependency <command>
check_dependency() {
    if ! command -v "$1" &> /dev/null; then
        echo "ERROR: '$1' is not installed or not on PATH."
        echo "       Install with:  pip install schemathesis"
        exit 1
    fi
}

# ---------------------------------------------------------------------------
# Pre-flight checks
# ---------------------------------------------------------------------------

check_dependency schemathesis
check_dependency curl

if [ ! -d "$CONTRACTS_DIR" ]; then
    echo "ERROR: Contracts directory '$CONTRACTS_DIR' not found."
    echo "       Run this script from the repository root."
    exit 1
fi

mkdir -p "$RESULTS_DIR"

# ---------------------------------------------------------------------------
# Main loop
# ---------------------------------------------------------------------------

PASS=0
FAIL=0
SKIP=0
declare -a FAILED_SERVICES=()
declare -a SKIPPED_SERVICES=()

# Collect all contract files, sorted for deterministic output.
mapfile -t CONTRACT_FILES < <(ls "${CONTRACTS_DIR}"/*.yaml 2>/dev/null | sort)

if [ ${#CONTRACT_FILES[@]} -eq 0 ]; then
    echo "WARNING: No contract files found in '${CONTRACTS_DIR}'. Nothing to test."
    exit 0
fi

echo ""
echo "================================================================"
echo " OpsNexus Contract Test Suite"
echo "================================================================"
echo " Contracts dir : $CONTRACTS_DIR"
echo " Results dir   : $RESULTS_DIR"
echo " Max examples  : $MAX_EXAMPLES"
echo " Timeout       : ${REQUEST_TIMEOUT}s"
echo "================================================================"
echo ""

for contract_file in "${CONTRACT_FILES[@]}"; do
    service=$(basename "$contract_file" .yaml)
    port="${SERVICE_PORTS[$service]:-}"

    if [ -z "$port" ]; then
        warn "No port mapping for '$service' — add it to SERVICE_PORTS in $0"
        SKIP=$((SKIP + 1))
        SKIPPED_SERVICES+=("$service (no port mapping)")
        continue
    fi

    base_url="http://localhost:${port}"
    health_url="${base_url}/health"
    junit_report="${RESULTS_DIR}/${service}-report.xml"

    info "Checking $service at $base_url ..."

    # Health check — skip if service is not reachable
    if ! is_reachable "$health_url"; then
        warn "$service not reachable at $health_url — skipping"
        SKIP=$((SKIP + 1))
        SKIPPED_SERVICES+=("$service (not reachable on port $port)")
        continue
    fi

    info "Running schemathesis for $service ..."

    # Build the schemathesis command
    SCHEMATHESIS_ARGS=(
        run "$contract_file"
        --base-url "$base_url"
        --validate-schema=true
        --checks all
        --hypothesis-max-examples "$MAX_EXAMPLES"
        --request-timeout "$REQUEST_TIMEOUT"
        --junit-xml "$junit_report"
        --output-truncation-size 500
    )

    # Attach bearer token if provided
    if [ -n "$AUTH_TOKEN" ]; then
        SCHEMATHESIS_ARGS+=(--auth "Bearer $AUTH_TOKEN")
    fi

    # Run schemathesis; capture exit code without aborting the script
    set +e
    schemathesis "${SCHEMATHESIS_ARGS[@]}" 2>&1
    exit_code=$?
    set -e

    if [ $exit_code -eq 0 ]; then
        success "$service — all contract checks passed"
        PASS=$((PASS + 1))
    else
        fail "$service — one or more contract checks failed (exit code $exit_code)"
        FAIL=$((FAIL + 1))
        FAILED_SERVICES+=("$service")
    fi

    echo ""
done

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------

echo ""
echo "================================================================"
echo " Contract Test Results"
echo "================================================================"
echo -e " ${GREEN}Passed${NC} : $PASS"
echo -e " ${RED}Failed${NC} : $FAIL"
echo -e " ${YELLOW}Skipped${NC}: $SKIP"

if [ ${#FAILED_SERVICES[@]} -gt 0 ]; then
    echo ""
    echo -e "${RED}Failed services:${NC}"
    for svc in "${FAILED_SERVICES[@]}"; do
        echo "  - $svc"
    done
fi

if [ ${#SKIPPED_SERVICES[@]} -gt 0 ]; then
    echo ""
    echo -e "${YELLOW}Skipped services:${NC}"
    for svc in "${SKIPPED_SERVICES[@]}"; do
        echo "  - $svc"
    done
fi

if [ -d "$RESULTS_DIR" ] && ls "${RESULTS_DIR}"/*.xml &>/dev/null; then
    echo ""
    echo "JUnit XML reports written to: $RESULTS_DIR/"
fi

echo "================================================================"
echo ""

if [ $FAIL -gt 0 ]; then
    exit 1
fi

exit 0
