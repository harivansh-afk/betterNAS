#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
compose_file="$repo_root/infra/docker/compose.dev.yml"
default_env_file="$repo_root/.env.agent"
env_file="${BETTERNAS_ENV_FILE:-$default_env_file}"

if [[ -f "$env_file" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$env_file"
  set +a
fi

: "${BETTERNAS_CLONE_NAME:=betternas-main}"
: "${COMPOSE_PROJECT_NAME:=betternas-${BETTERNAS_CLONE_NAME}}"
: "${BETTERNAS_CONTROL_PLANE_PORT:=3001}"
: "${BETTERNAS_NODE_AGENT_PORT:=3090}"
: "${BETTERNAS_NEXTCLOUD_PORT:=8080}"
: "${BETTERNAS_VERSION:=local-dev}"
: "${NEXTCLOUD_ADMIN_USER:=admin}"
: "${NEXTCLOUD_ADMIN_PASSWORD:=admin}"

if [[ -z "${BETTERNAS_EXPORT_PATH:-}" ]]; then
  BETTERNAS_EXPORT_PATH="$repo_root/.state/$BETTERNAS_CLONE_NAME/export"
fi

if [[ "$BETTERNAS_EXPORT_PATH" != /* ]]; then
  BETTERNAS_EXPORT_PATH="$repo_root/$BETTERNAS_EXPORT_PATH"
fi

: "${BETTERNAS_NODE_DIRECT_ADDRESS:=http://localhost:${BETTERNAS_NODE_AGENT_PORT}}"
: "${BETTERNAS_EXAMPLE_MOUNT_URL:=http://localhost:${BETTERNAS_NODE_AGENT_PORT}/dav/}"
: "${NEXTCLOUD_BASE_URL:=http://localhost:${BETTERNAS_NEXTCLOUD_PORT}}"

export repo_root
export compose_file
export env_file
export BETTERNAS_CLONE_NAME
export COMPOSE_PROJECT_NAME
export BETTERNAS_CONTROL_PLANE_PORT
export BETTERNAS_NODE_AGENT_PORT
export BETTERNAS_NEXTCLOUD_PORT
export BETTERNAS_EXPORT_PATH
export BETTERNAS_VERSION
export NEXTCLOUD_ADMIN_USER
export NEXTCLOUD_ADMIN_PASSWORD
export BETTERNAS_NODE_DIRECT_ADDRESS
export BETTERNAS_EXAMPLE_MOUNT_URL
export NEXTCLOUD_BASE_URL

mkdir -p "$BETTERNAS_EXPORT_PATH"

compose() {
  docker compose -f "$compose_file" "$@"
}
