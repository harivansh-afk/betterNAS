#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
compose_file="$repo_root/infra/docker/compose.dev.yml"
default_env_file="$repo_root/.env.agent"
env_file="${BETTERNAS_ENV_FILE:-$default_env_file}"

# shellcheck disable=SC1091
source "$repo_root/scripts/lib/agent-env.sh"

if [[ -f "$env_file" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$env_file"
  set +a
fi

if [[ -z "${BETTERNAS_CLONE_NAME:-}" ]]; then
  BETTERNAS_CLONE_NAME="$(betternas_default_clone_name "$repo_root")"
fi

COMPOSE_PROJECT_NAME="$(
  betternas_resolve_compose_project_name "$repo_root" "${COMPOSE_PROJECT_NAME:-}" "$BETTERNAS_CLONE_NAME"
)"

read -r default_nextcloud_port default_node_agent_port default_control_plane_port <<<"$(betternas_default_ports "$repo_root" "$BETTERNAS_CLONE_NAME")"

: "${BETTERNAS_CONTROL_PLANE_PORT:=$default_control_plane_port}"
: "${BETTERNAS_NODE_AGENT_PORT:=$default_node_agent_port}"
: "${BETTERNAS_NEXTCLOUD_PORT:=$default_nextcloud_port}"
: "${BETTERNAS_VERSION:=local-dev}"
: "${BETTERNAS_USERNAME:=${BETTERNAS_CLONE_NAME}-user}"
: "${BETTERNAS_PASSWORD:=${BETTERNAS_CLONE_NAME}-password123}"
: "${BETTERNAS_NODE_MACHINE_ID:=${BETTERNAS_CLONE_NAME}-node}"
: "${BETTERNAS_NODE_DISPLAY_NAME:=${BETTERNAS_CLONE_NAME} node}"
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
export BETTERNAS_USERNAME
export BETTERNAS_PASSWORD
export BETTERNAS_NODE_MACHINE_ID
export BETTERNAS_NODE_DISPLAY_NAME
export NEXTCLOUD_ADMIN_USER
export NEXTCLOUD_ADMIN_PASSWORD
export BETTERNAS_NODE_DIRECT_ADDRESS
export BETTERNAS_EXAMPLE_MOUNT_URL
export NEXTCLOUD_BASE_URL

mkdir -p "$BETTERNAS_EXPORT_PATH"

compose() {
  docker compose -f "$compose_file" "$@"
}
