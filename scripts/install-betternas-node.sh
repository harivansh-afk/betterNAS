#!/usr/bin/env bash

set -euo pipefail

repo="${BETTERNAS_NODE_REPO:-harivansh-afk/betterNAS}"
binary_name="betternas-node"
install_dir="${BETTERNAS_INSTALL_DIR:-$HOME/.local/bin}"
version="${BETTERNAS_NODE_VERSION:-}"
release_api_url="${BETTERNAS_NODE_RELEASE_API_URL:-https://api.github.com/repos/${repo}/releases/latest}"
download_base_url="${BETTERNAS_NODE_DOWNLOAD_BASE_URL:-https://github.com/${repo}/releases/download}"

if [[ -z "$version" ]]; then
  version="$(
    curl -fsSL "${release_api_url}" | \
      python3 -c 'import json,sys; print(json.load(sys.stdin)["tag_name"])'
  )"
fi

os_name="$(uname -s)"
arch_name="$(uname -m)"

case "$os_name" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *)
    echo "Unsupported OS: $os_name" >&2
    exit 1
    ;;
esac

case "$arch_name" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "Unsupported architecture: $arch_name" >&2
    exit 1
    ;;
esac

archive_name="${binary_name}_${version}_${os}_${arch}.tar.gz"
download_url="${download_base_url}/${version}/${archive_name}"

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

echo "Downloading ${download_url}"
curl -fsSL "$download_url" -o "${tmp_dir}/${archive_name}"

mkdir -p "$install_dir"
tar -xzf "${tmp_dir}/${archive_name}" -C "$tmp_dir"
install -m 0755 "${tmp_dir}/${binary_name}" "${install_dir}/${binary_name}"

cat <<EOF
Installed ${binary_name} to ${install_dir}/${binary_name}

If ${install_dir} is not in your PATH, add this to your shell profile:
  export PATH="${install_dir}:\$PATH"

Then run the node with:
  BETTERNAS_USERNAME=your-username \\
  BETTERNAS_PASSWORD=your-password \\
  BETTERNAS_EXPORT_PATH=/path/to/export \\
  BETTERNAS_NODE_DIRECT_ADDRESS=https://your-public-node-url \\
  ${binary_name}

The control plane defaults to https://api.betternas.com.
EOF
