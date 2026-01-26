#!/bin/sh
set -e

# Install clankers-daemon binary from GitHub Releases
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/dxta-dev/clankers/main/scripts/install-daemon.sh | sh
#   curl -fsSL ... | sh -s -- v0.1.0
#   curl -fsSL ... | CLANKERS_INSTALL_DIR=/usr/local/bin sh
#
# Environment variables:
#   CLANKERS_VERSION      - Version to install (default: latest)
#   CLANKERS_INSTALL_DIR  - Override install directory (default: ~/.local/bin)
#   GITHUB_TOKEN          - Optional, for higher API rate limits

REPO="dxta-dev/clankers"
BINARY_NAME="clankers-daemon"

# --- Helpers ---

log() {
  printf '[clankers] %s\n' "$1"
}

error() {
  printf '[clankers] ERROR: %s\n' "$1" >&2
  exit 1
}

# Detect OS
detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *) error "Unsupported OS: $(uname -s)" ;;
  esac
}

# Detect architecture
detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) error "Unsupported architecture: $(uname -m)" ;;
  esac
}

# Get latest release tag from GitHub
get_latest_version() {
  url="https://api.github.com/repos/${REPO}/releases/latest"
  
  if command -v curl >/dev/null 2>&1; then
    if [ -n "$GITHUB_TOKEN" ]; then
      curl -fsSL -H "Authorization: Bearer $GITHUB_TOKEN" "$url" | grep '"tag_name"' | head -1 | cut -d'"' -f4
    else
      curl -fsSL "$url" | grep '"tag_name"' | head -1 | cut -d'"' -f4
    fi
  elif command -v wget >/dev/null 2>&1; then
    if [ -n "$GITHUB_TOKEN" ]; then
      wget -qO- --header="Authorization: Bearer $GITHUB_TOKEN" "$url" | grep '"tag_name"' | head -1 | cut -d'"' -f4
    else
      wget -qO- "$url" | grep '"tag_name"' | head -1 | cut -d'"' -f4
    fi
  else
    error "Neither curl nor wget found. Please install one of them."
  fi
}

# Download file
download() {
  src="$1"
  dest="$2"
  
  log "Downloading $src"
  
  if command -v curl >/dev/null 2>&1; then
    if [ -n "$GITHUB_TOKEN" ]; then
      curl -fsSL -H "Authorization: Bearer $GITHUB_TOKEN" -o "$dest" "$src"
    else
      curl -fsSL -o "$dest" "$src"
    fi
  elif command -v wget >/dev/null 2>&1; then
    if [ -n "$GITHUB_TOKEN" ]; then
      wget -q --header="Authorization: Bearer $GITHUB_TOKEN" -O "$dest" "$src"
    else
      wget -q -O "$dest" "$src"
    fi
  else
    error "Neither curl nor wget found."
  fi
}

# Verify checksum
verify_checksum() {
  file="$1"
  expected="$2"
  
  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$file" | cut -d' ' -f1)
  elif command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "$file" | cut -d' ' -f1)
  else
    log "Warning: No sha256sum or shasum found, skipping checksum verification"
    return 0
  fi
  
  if [ "$actual" != "$expected" ]; then
    error "Checksum mismatch! Expected: $expected, Actual: $actual"
  fi
  
  log "Checksum verified"
}

# Determine install directory
get_install_dir() {
  if [ -n "$CLANKERS_INSTALL_DIR" ]; then
    echo "$CLANKERS_INSTALL_DIR"
  elif [ -d "$HOME/.local/bin" ]; then
    echo "$HOME/.local/bin"
  else
    echo "$HOME/bin"
  fi
}

# --- Main ---

main() {
  # Version from arg, env var, or fetch latest
  VERSION="${1:-${CLANKERS_VERSION:-}}"
  
  OS=$(detect_os)
  ARCH=$(detect_arch)
  TARGET="${OS}-${ARCH}"
  
  log "Detected platform: $TARGET"
  
  # Get version
  if [ -z "$VERSION" ]; then
    log "Fetching latest version..."
    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
      error "Could not determine latest version. Set CLANKERS_VERSION=v0.1.0"
    fi
  fi
  
  log "Installing version: $VERSION"
  
  # Determine binary filename
  if [ "$OS" = "windows" ]; then
    ARTIFACT="${TARGET}-${BINARY_NAME}.exe"
    DEST_NAME="${BINARY_NAME}.exe"
  else
    ARTIFACT="${TARGET}-${BINARY_NAME}"
    DEST_NAME="${BINARY_NAME}"
  fi
  
  # URLs
  BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
  BINARY_URL="${BASE_URL}/${ARTIFACT}"
  CHECKSUMS_URL="${BASE_URL}/checksums.txt"
  
  # Create temp directory
  TMP_DIR=$(mktemp -d)
  trap 'rm -rf "$TMP_DIR"' EXIT
  
  # Download binary and checksums
  download "$BINARY_URL" "$TMP_DIR/$ARTIFACT"
  download "$CHECKSUMS_URL" "$TMP_DIR/checksums.txt"
  
  # Extract expected checksum
  EXPECTED_CHECKSUM=$(grep "$ARTIFACT" "$TMP_DIR/checksums.txt" | cut -d' ' -f1)
  if [ -z "$EXPECTED_CHECKSUM" ]; then
    error "Could not find checksum for $ARTIFACT"
  fi
  
  # Verify
  verify_checksum "$TMP_DIR/$ARTIFACT" "$EXPECTED_CHECKSUM"
  
  # Install
  INSTALL_DIR=$(get_install_dir)
  mkdir -p "$INSTALL_DIR"
  
  DEST="$INSTALL_DIR/$DEST_NAME"
  mv "$TMP_DIR/$ARTIFACT" "$DEST"
  chmod +x "$DEST"
  
  log "Installed to $DEST"
  
  # Check if in PATH
  case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
      log ""
      log "Note: $INSTALL_DIR is not in your PATH."
      log "Add it with: export PATH=\"$INSTALL_DIR:\$PATH\""
      ;;
  esac
  
  log ""
  log "Done! Run 'clankers-daemon --help' to get started."
}

main "$@"
