#!/usr/bin/env bash
set -e

# ── Configuration ─────────────────────────────────────────────────────────────
DB_CONTAINER="isu-zero-test-db"
DB_PORT=5433
DB_NAME="isu_zero_test"
DB_USER="isu"
DB_PASS="testpassword"

NATS_CONTAINER="isu-zero-test-nats"
NATS_PORT=4223

TEST_NETWORK="isu-zero-test-net"

# URLs used inside the test runner container — use container names not localhost
INTERNAL_DB_URL="postgres://${DB_USER}:${DB_PASS}@${DB_CONTAINER}:5432/${DB_NAME}?sslmode=disable"
INTERNAL_NATS_URL="nats://${NATS_CONTAINER}:4222"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# ── Cleanup ───────────────────────────────────────────────────────────────────
cleanup() {
  echo ""
  echo "→ Cleaning up test containers and network..."
  docker rm -f "${DB_CONTAINER}" "${NATS_CONTAINER}" 2>/dev/null || true
  docker network rm "${TEST_NETWORK}" 2>/dev/null || true
}
trap cleanup EXIT

# ── Create test network ───────────────────────────────────────────────────────
echo "→ Creating test network..."
docker network create "${TEST_NETWORK}" 2>/dev/null || true

# ── Start Postgres ────────────────────────────────────────────────────────────
echo "→ Starting test Postgres on port ${DB_PORT}..."
docker rm -f "${DB_CONTAINER}" 2>/dev/null || true
docker run -d \
  --name "${DB_CONTAINER}" \
  --network "${TEST_NETWORK}" \
  -e POSTGRES_DB="${DB_NAME}" \
  -e POSTGRES_USER="${DB_USER}" \
  -e POSTGRES_PASSWORD="${DB_PASS}" \
  -p "${DB_PORT}:5432" \
  postgres:16-alpine

# ── Start NATS ────────────────────────────────────────────────────────────────
echo "→ Starting test NATS on port ${NATS_PORT}..."
docker rm -f "${NATS_CONTAINER}" 2>/dev/null || true
docker run -d \
  --name "${NATS_CONTAINER}" \
  --network "${TEST_NETWORK}" \
  -p "${NATS_PORT}:4222" \
  -p 8222:8222 \
  nats:2.10-alpine --jetstream --http_port 8222

# ── Wait for Postgres ─────────────────────────────────────────────────────────
echo "→ Waiting for Postgres to be ready..."
until docker exec "${DB_CONTAINER}" pg_isready -U "${DB_USER}" -d "${DB_NAME}" > /dev/null 2>&1; do
  sleep 1
done
echo "  Postgres ready."

# ── Wait for NATS ─────────────────────────────────────────────────────────────
echo "→ Waiting for NATS to be ready..."
until docker exec "${NATS_CONTAINER}" \
  wget -q --spider http://localhost:8222/healthz > /dev/null 2>&1; do
  sleep 1
done
echo "  NATS ready."

# ── Load schema ───────────────────────────────────────────────────────────────
echo "→ Loading schema..."
docker exec -i "${DB_CONTAINER}" psql -U "${DB_USER}" -d "${DB_NAME}" \
  < "${PROJECT_ROOT}/infra/init.sql"

# ── Load test seed data ───────────────────────────────────────────────────────
echo "→ Loading test seed data..."
docker exec -i "${DB_CONTAINER}" psql -U "${DB_USER}" -d "${DB_NAME}" \
  < "${PROJECT_ROOT}/infra/seed_test.sql"

# ── Run tests inside a Go container ──────────────────────────────────────────
echo ""
echo "→ Running tests..."
echo "  PROJECT_ROOT=${PROJECT_ROOT}"
echo "  Backend path=${PROJECT_ROOT}/backend"
echo "→ Verifying mount..."
docker run --rm \
  -v "${PROJECT_ROOT}/backend:/app" \
  -w /app \
  golang:1.26-alpine \
  ls -la /app
docker run --rm \
  --network "${TEST_NETWORK}" \
  -e DATABASE_URL="${INTERNAL_DB_URL}" \
  -e NATS_URL="${INTERNAL_NATS_URL}" \
  -v "${PROJECT_ROOT}/backend:/app" \
  -w /app \
  golang:1.26-alpine \
  sh -c "go test -mod=vendor \
    ./internal/products/tests/... \
    ./internal/interactions/tests/... \
    ./internal/navigation/tests/... \
    -v -count=1"

echo ""
echo "✓ All tests passed."