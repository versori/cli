#!/usr/bin/env sh
set -e

REPO="versori/cli"
BINARY_NAME="versori"
INSTALL_DIR="/usr/local/bin"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

info()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
ok()    { printf '\033[1;32m  ✓\033[0m %s\n' "$*"; }
die()   { printf '\033[1;31mError:\033[0m %s\n' "$*" >&2; exit 1; }

need() {
  command -v "$1" >/dev/null 2>&1 || die "'$1' is required but not found. Please install it and re-run."
}

# ---------------------------------------------------------------------------
# Detect OS & arch
# ---------------------------------------------------------------------------

detect_os() {
  case "$(uname -s)" in
    Linux)  echo "linux"  ;;
    Darwin) echo "darwin" ;;
    *)      die "Unsupported OS: $(uname -s). Only Linux and macOS are supported." ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) die "Unsupported architecture: $(uname -m)." ;;
  esac
}

# ---------------------------------------------------------------------------
# Resolve version
# ---------------------------------------------------------------------------

latest_version() {
  need curl
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed 's/.*"tag_name": *"\(.*\)".*/\1/'
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

main() {
  need curl
  need tar

  VERSION="${VERSORI_VERSION:-}"

  if [ -z "$VERSION" ]; then
    info "Fetching latest release version..."
    VERSION="$(latest_version)"
    [ -n "$VERSION" ] || die "Could not determine the latest release version."
  fi

  # Strip leading 'v' for the filename (goreleaser uses bare semver in filenames)
  VERSION_NUM="${VERSION#v}"

  OS="$(detect_os)"
  ARCH="$(detect_arch)"

  FILENAME="cli_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
  URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

  info "Installing ${BINARY_NAME} ${VERSION} (${OS}/${ARCH})..."
  info "Downloading ${URL}"

  TMP_DIR="$(mktemp -d)"
  trap 'rm -rf "$TMP_DIR"' EXIT

  curl -fsSL "$URL" -o "${TMP_DIR}/${FILENAME}" || die "Download failed. Check the version and your internet connection."

  # ---------------------------------------------------------------------------
  # Verify checksum
  # ---------------------------------------------------------------------------

  CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"
  info "Downloading checksums from ${CHECKSUMS_URL}"
  curl -fsSL "$CHECKSUMS_URL" -o "${TMP_DIR}/checksums.txt" || die "Failed to download checksums file."

  EXPECTED_SUM="$(grep "${FILENAME}" "${TMP_DIR}/checksums.txt" | awk '{print $1}')"
  [ -n "$EXPECTED_SUM" ] || die "Could not find checksum for ${FILENAME} in checksums.txt."

  info "Verifying checksum..."
  if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL_SUM="$(sha256sum "${TMP_DIR}/${FILENAME}" | awk '{print $1}')"
  elif command -v shasum >/dev/null 2>&1; then
    ACTUAL_SUM="$(shasum -a 256 "${TMP_DIR}/${FILENAME}" | awk '{print $1}')"
  else
    die "Neither 'sha256sum' nor 'shasum' found. Cannot verify checksum."
  fi

  if [ "$EXPECTED_SUM" != "$ACTUAL_SUM" ]; then
    die "Checksum mismatch!\n  Expected: ${EXPECTED_SUM}\n  Actual:   ${ACTUAL_SUM}\nThe downloaded file may be corrupted or tampered with."
  fi
  ok "Checksum verified."

  tar -xzf "${TMP_DIR}/${FILENAME}" -C "$TMP_DIR"

  BINARY="${TMP_DIR}/${BINARY_NAME}"
  [ -f "$BINARY" ] || die "Binary '${BINARY_NAME}' not found in the archive."

  chmod +x "$BINARY"

  # Install – try without sudo first, fall back to sudo
  if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY" "${INSTALL_DIR}/${BINARY_NAME}"
  else
    info "Writing to ${INSTALL_DIR} requires elevated permissions (sudo)..."
    sudo mv "$BINARY" "${INSTALL_DIR}/${BINARY_NAME}"
  fi

  ok "${BINARY_NAME} ${VERSION} installed to ${INSTALL_DIR}/${BINARY_NAME}"

  # Sanity check
  if command -v "$BINARY_NAME" >/dev/null 2>&1; then
    ok "Run '${BINARY_NAME} version' to get started."
  else
    printf '\n\033[1;33mNote:\033[0m %s is not on your PATH.\n' "$INSTALL_DIR"
    printf 'Add the following to your shell profile and restart your terminal:\n'
    printf '  export PATH="%s:$PATH"\n\n' "$INSTALL_DIR"
  fi
}

main "$@"

