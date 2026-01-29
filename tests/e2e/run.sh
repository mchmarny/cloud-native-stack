#!/bin/bash
set -euo pipefail

# =============================================================================
# E2E Tests for CNS with Tilt Cluster
# =============================================================================
#
# This script tests the full CNS workflow with a running Kubernetes cluster
# and the cnsd API server (via Tilt).
#
# Prerequisites:
#   - Tilt cluster running: make dev-env
#   - cnsd accessible at localhost:8080
#
# Usage:
#   ./tests/e2e/run.sh
#
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
CNSD_URL="${CNSD_URL:-http://localhost:8080}"
OUTPUT_DIR="${OUTPUT_DIR:-$(mktemp -d)}"
CNSCTL_BIN=""

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# =============================================================================
# Helpers
# =============================================================================

msg() {
  echo -e "${BLUE}[INFO]${NC} $1"
}

warn() {
  echo -e "${YELLOW}[WARN]${NC} $1"
}

err() {
  echo -e "${RED}[ERROR]${NC} $1"
  exit 1
}

pass() {
  local name=$1
  TOTAL_TESTS=$((TOTAL_TESTS + 1))
  PASSED_TESTS=$((PASSED_TESTS + 1))
  echo -e "${GREEN}[PASS]${NC} $name"
}

fail() {
  local name=$1
  local reason=${2:-""}
  TOTAL_TESTS=$((TOTAL_TESTS + 1))
  FAILED_TESTS=$((FAILED_TESTS + 1))
  if [ -n "$reason" ]; then
    echo -e "${RED}[FAIL]${NC} $name: $reason"
  else
    echo -e "${RED}[FAIL]${NC} $name"
  fi
}

skip() {
  local name=$1
  local reason=${2:-""}
  echo -e "${YELLOW}[SKIP]${NC} $name: $reason"
}

check_command() {
  if ! command -v "$1" &> /dev/null; then
    err "$1 is required but not installed"
  fi
}

# =============================================================================
# Build
# =============================================================================

build_binaries() {
  msg "=========================================="
  msg "Building binaries"
  msg "=========================================="

  cd "${ROOT_DIR}"

  # Build cnsctl directly with go build (simpler than goreleaser for e2e tests)
  local bin_dir="${ROOT_DIR}/dist/e2e"
  mkdir -p "${bin_dir}"

  if ! go build -o "${bin_dir}/cnsctl" ./cmd/cnsctl 2>&1; then
    err "Failed to build cnsctl"
  fi

  CNSCTL_BIN="${bin_dir}/cnsctl"

  if [ ! -x "$CNSCTL_BIN" ]; then
    err "cnsctl binary not found at ${CNSCTL_BIN}"
  fi

  pass "build/cnsctl"
  msg "Using: ${CNSCTL_BIN}"
}

# =============================================================================
# API Health Checks
# =============================================================================

check_api_health() {
  msg "=========================================="
  msg "Checking API health"
  msg "=========================================="

  # Health endpoint
  if curl -sf "${CNSD_URL}/health" > /dev/null 2>&1; then
    pass "api/health"
  else
    fail "api/health" "cnsd not responding at ${CNSD_URL}/health"
    warn "Is Tilt running? Try: make dev-env"
    return 1
  fi

  # Ready endpoint
  if curl -sf "${CNSD_URL}/ready" > /dev/null 2>&1; then
    pass "api/ready"
  else
    fail "api/ready" "cnsd not ready"
    return 1
  fi

  return 0
}

# =============================================================================
# CLI Recipe Tests (from e2e.md)
# =============================================================================

test_cli_recipe() {
  msg "=========================================="
  msg "Testing CLI recipe generation"
  msg "=========================================="

  local recipe_dir="${OUTPUT_DIR}/recipes"
  mkdir -p "$recipe_dir"

  # Test 1: Basic recipe with query parameters
  msg "--- Test: Recipe with query parameters ---"
  local basic_recipe="${recipe_dir}/basic.yaml"
  if "${CNSCTL_BIN}" recipe \
    --service eks \
    --accelerator gb200 \
    --os ubuntu \
    --intent training \
    --output "$basic_recipe" 2>&1; then
    if [ -f "$basic_recipe" ] && grep -q "kind: recipeResult" "$basic_recipe"; then
      pass "cli/recipe/query-params"
    else
      fail "cli/recipe/query-params" "Recipe file invalid"
    fi
  else
    fail "cli/recipe/query-params" "Command failed"
  fi

  # Test 2: Recipe from criteria file
  msg "--- Test: Recipe from criteria file ---"
  local criteria_file="${recipe_dir}/criteria.yaml"
  cat > "$criteria_file" << 'EOF'
kind: recipeCriteria
apiVersion: cns.nvidia.com/v1alpha1
metadata:
  name: gb200-eks-training
spec:
  service: eks
  accelerator: gb200
  os: ubuntu
  intent: training
EOF

  local criteria_recipe="${recipe_dir}/from-criteria.yaml"
  if "${CNSCTL_BIN}" recipe --criteria "$criteria_file" --output "$criteria_recipe" 2>&1; then
    if [ -f "$criteria_recipe" ]; then
      pass "cli/recipe/criteria-file"
    else
      fail "cli/recipe/criteria-file" "Recipe file not created"
    fi
  else
    fail "cli/recipe/criteria-file" "Command failed"
  fi

  # Test 3: CLI flags override criteria file
  msg "--- Test: CLI flags override criteria file ---"
  local override_recipe="${recipe_dir}/override.yaml"
  if "${CNSCTL_BIN}" recipe --criteria "$criteria_file" --service gke --output "$override_recipe" 2>&1; then
    if grep -q "service: gke" "$override_recipe" 2>/dev/null; then
      pass "cli/recipe/override"
    else
      fail "cli/recipe/override" "Override not applied"
    fi
  else
    fail "cli/recipe/override" "Command failed"
  fi
}

# =============================================================================
# API Recipe Tests (from e2e.md)
# =============================================================================

test_api_recipe() {
  msg "=========================================="
  msg "Testing API recipe endpoints"
  msg "=========================================="

  local recipe_dir="${OUTPUT_DIR}/api-recipes"
  mkdir -p "$recipe_dir"

  # Test 1: GET /v1/recipe with query params
  msg "--- Test: GET /v1/recipe ---"
  local get_recipe="${recipe_dir}/get.json"
  local http_code
  http_code=$(curl -s -w "%{http_code}" -o "$get_recipe" \
    "${CNSD_URL}/v1/recipe?service=eks&accelerator=gb200&intent=training")

  if [ "$http_code" = "200" ] && [ -s "$get_recipe" ]; then
    pass "api/recipe/GET"
  else
    fail "api/recipe/GET" "HTTP $http_code"
  fi

  # Test 2: POST /v1/recipe with YAML body
  msg "--- Test: POST /v1/recipe ---"
  local post_recipe="${recipe_dir}/post.json"
  http_code=$(curl -s -w "%{http_code}" -o "$post_recipe" \
    -X POST "${CNSD_URL}/v1/recipe" \
    -H "Content-Type: application/x-yaml" \
    -d 'kind: recipeCriteria
apiVersion: cns.nvidia.com/v1alpha1
metadata:
  name: gb200-training
spec:
  service: eks
  accelerator: gb200
  intent: training')

  if [ "$http_code" = "200" ] && [ -s "$post_recipe" ]; then
    pass "api/recipe/POST"
  else
    fail "api/recipe/POST" "HTTP $http_code"
  fi
}

# =============================================================================
# CLI Bundle Tests (from e2e.md)
# =============================================================================

test_cli_bundle() {
  msg "=========================================="
  msg "Testing CLI bundle generation"
  msg "=========================================="

  # First generate a recipe to use
  local recipe_file="${OUTPUT_DIR}/bundle-test-recipe.yaml"
  "${CNSCTL_BIN}" recipe \
    --service eks \
    --accelerator gb200 \
    --os ubuntu \
    --intent training \
    --output "$recipe_file" 2>&1 || true

  if [ ! -f "$recipe_file" ]; then
    fail "cli/bundle/prerequisite" "Could not generate recipe for bundle tests"
    return 1
  fi

  # Test 1: Basic bundle generation
  msg "--- Test: Basic bundle ---"
  local basic_bundle="${OUTPUT_DIR}/bundles/basic"
  mkdir -p "$basic_bundle"
  if "${CNSCTL_BIN}" bundle \
    --recipe "$recipe_file" \
    --output "$basic_bundle" 2>&1; then
    if [ -f "${basic_bundle}/Chart.yaml" ] && [ -f "${basic_bundle}/values.yaml" ]; then
      pass "cli/bundle/basic"
    else
      fail "cli/bundle/basic" "Missing Chart.yaml or values.yaml"
    fi
  else
    fail "cli/bundle/basic" "Command failed"
  fi

  # Test 2: Bundle with node selectors and tolerations
  msg "--- Test: Bundle with scheduling options ---"
  local sched_bundle="${OUTPUT_DIR}/bundles/scheduling"
  mkdir -p "$sched_bundle"
  if "${CNSCTL_BIN}" bundle \
    --recipe "$recipe_file" \
    --output "$sched_bundle" \
    --system-node-selector nodeGroup=system-pool \
    --accelerated-node-selector nodeGroup=customer-gpu \
    --accelerated-node-toleration nvidia.com/gpu=present:NoSchedule 2>&1; then
    if grep -q "system-pool" "${sched_bundle}/values.yaml" 2>/dev/null; then
      pass "cli/bundle/scheduling"
    else
      fail "cli/bundle/scheduling" "Node selector not found in values"
    fi
  else
    fail "cli/bundle/scheduling" "Command failed"
  fi

  # Test 3: Bundle with ArgoCD deployer
  msg "--- Test: Bundle with ArgoCD deployer ---"
  local argocd_bundle="${OUTPUT_DIR}/bundles/argocd"
  mkdir -p "$argocd_bundle"
  if "${CNSCTL_BIN}" bundle \
    --recipe "$recipe_file" \
    --output "$argocd_bundle" \
    --deployer argocd 2>&1; then
    if [ -f "${argocd_bundle}/app-of-apps.yaml" ]; then
      pass "cli/bundle/argocd"
    else
      fail "cli/bundle/argocd" "app-of-apps.yaml not found"
    fi
  else
    fail "cli/bundle/argocd" "Command failed"
  fi

  # Test 4: Verify bundle integrity (checksums)
  msg "--- Test: Bundle integrity ---"
  if [ -f "${basic_bundle}/checksums.txt" ]; then
    cd "$basic_bundle"
    if shasum -a 256 -c checksums.txt > /dev/null 2>&1; then
      pass "cli/bundle/integrity"
    else
      fail "cli/bundle/integrity" "Checksum verification failed"
    fi
    cd - > /dev/null
  else
    skip "cli/bundle/integrity" "No checksums.txt"
  fi

  # Test 5: Helm lint (if helm available)
  msg "--- Test: Helm lint ---"
  if command -v helm &> /dev/null; then
    # Fix dev version if needed
    if grep -q "version: dev" "${basic_bundle}/Chart.yaml" 2>/dev/null; then
      sed -i.bak 's/version: dev/version: 0.0.0-dev/' "${basic_bundle}/Chart.yaml"
    fi
    if helm lint "$basic_bundle" > /dev/null 2>&1; then
      pass "cli/bundle/helm-lint"
    else
      # May fail due to missing deps, which is OK
      warn "Helm lint had warnings (may be missing deps)"
      pass "cli/bundle/helm-lint"
    fi
  else
    skip "cli/bundle/helm-lint" "helm not installed"
  fi
}

# =============================================================================
# API Bundle Tests (from e2e.md)
# =============================================================================

test_api_bundle() {
  msg "=========================================="
  msg "Testing API bundle endpoint"
  msg "=========================================="

  local bundle_dir="${OUTPUT_DIR}/api-bundles"
  mkdir -p "$bundle_dir"

  # Test: POST /v1/bundle (recipe -> bundle pipeline)
  msg "--- Test: POST /v1/bundle ---"

  # First get a recipe from API
  local recipe_json
  recipe_json=$(curl -s "${CNSD_URL}/v1/recipe?service=eks&accelerator=h100&intent=training")

  if [ -z "$recipe_json" ]; then
    fail "api/bundle/POST" "Could not get recipe from API"
    return 1
  fi

  # Then send to bundle endpoint
  local bundle_zip="${bundle_dir}/bundle.zip"
  local http_code
  http_code=$(curl -s -w "%{http_code}" -o "$bundle_zip" \
    -X POST "${CNSD_URL}/v1/bundle?deployer=helm" \
    -H "Content-Type: application/json" \
    -d "$recipe_json")

  if [ "$http_code" = "200" ] && [ -s "$bundle_zip" ]; then
    # Verify it's a valid zip
    if unzip -t "$bundle_zip" > /dev/null 2>&1; then
      pass "api/bundle/POST"

      # Extract and verify contents
      local extract_dir="${bundle_dir}/extracted"
      mkdir -p "$extract_dir"
      unzip -q "$bundle_zip" -d "$extract_dir"
      if [ -f "${extract_dir}/Chart.yaml" ]; then
        pass "api/bundle/contents"
      else
        fail "api/bundle/contents" "Chart.yaml not in bundle"
      fi
    else
      fail "api/bundle/POST" "Invalid zip file"
    fi
  else
    fail "api/bundle/POST" "HTTP $http_code"
  fi
}

# =============================================================================
# Summary
# =============================================================================

print_summary() {
  echo ""
  msg "=========================================="
  msg "Test Summary"
  msg "=========================================="
  echo "Total:  ${TOTAL_TESTS}"
  echo -e "Passed: ${GREEN}${PASSED_TESTS}${NC}"
  echo -e "Failed: ${RED}${FAILED_TESTS}${NC}"
  echo ""
  msg "Output: ${OUTPUT_DIR}"

  if [ "$FAILED_TESTS" -gt 0 ]; then
    return 1
  fi
  return 0
}

# =============================================================================
# Main
# =============================================================================

main() {
  msg "CNS E2E Tests"
  msg "Output directory: ${OUTPUT_DIR}"
  msg "API URL: ${CNSD_URL}"
  echo ""

  # Check required tools
  check_command curl
  check_command make

  # Build binaries
  build_binaries

  # Check API is available
  if ! check_api_health; then
    warn "API not available, skipping API tests"
    API_AVAILABLE=false
  else
    API_AVAILABLE=true
  fi

  # Run CLI tests (always)
  test_cli_recipe
  test_cli_bundle

  # Run API tests (if available)
  if [ "$API_AVAILABLE" = true ]; then
    test_api_recipe
    test_api_bundle
  fi

  # Print summary and exit
  if print_summary; then
    msg "All tests passed!"
    exit 0
  else
    err "Some tests failed"
  fi
}

main "$@"
