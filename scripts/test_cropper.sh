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

INTERNAL_DB_URL="postgres://${DB_USER}:${DB_PASS}@${DB_CONTAINER}:5432/${DB_NAME}?sslmode=disable"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# ── Cleanup ───────────────────────────────────────────────────────────────────
cleanup() {
  echo ""
  echo "→ Cleaning up test containers, network and image..."
  docker rm -f "${DB_CONTAINER}" "${NATS_CONTAINER}" 2>/dev/null || true
  docker network rm "${TEST_NETWORK}" 2>/dev/null || true
  docker rmi isu-zero-cropper-test 2>/dev/null || true
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

# ── Run tests inside the cropper Docker image ─────────────────────────────────
echo ""
echo "→ Building cropper image..."
docker build --no-cache -t isu-zero-cropper-test "${PROJECT_ROOT}/cropper"

echo ""
echo "→ Running cropper tests..."
docker run --rm \
  --network "${TEST_NETWORK}" \
  -e DATABASE_URL="${INTERNAL_DB_URL}" \
  -v "${PROJECT_ROOT}/cropper:/app" \
  -w /app \
  isu-zero-cropper-test \
  python -m pytest tests/test_hasher.py tests/test_storage.py -v

echo ""
echo "✓ All cropper tests passed."

# Remove the test image to free disk space
docker rmi isu-zero-cropper-test 2>/dev/null || true