#!/usr/bin/env bash
#
# Ko build wrapper for Tilt integration
# Usage: ko-tilt-build.sh <module-dir> <expected-ref>
#
# This script wraps ko build for use with Tilt's custom_build.
# It detects the platform and builds to the local registry.

set -euo pipefail

if [ $# -ne 2 ]; then
  echo "Error: Missing required arguments" >&2
  echo "Usage: $0 <module-dir> <expected-ref>" >&2
  exit 1
fi

MODULE_DIR="$1"
EXPECTED_REF="$2"

cd "$MODULE_DIR"

# Detect platform
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)
    PLATFORM="linux/amd64"
    ;;
  aarch64|arm64)
    PLATFORM="linux/arm64"
    ;;
  *)
    echo "Error: Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# Parse repo and tag from expected reference
REPO="${EXPECTED_REF%:*}"
TAG="${EXPECTED_REF##*:}"

# Build with ko and push to registry
KO_DOCKER_REPO="$REPO" ko build --bare --platform="${PLATFORM}" --tags="$TAG" ./

echo "$EXPECTED_REF"
