#!/bin/sh
# install.sh - one-line installer for sproot
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/justanotherspy/sproot/main/install.sh | sh
#
# Environment variables:
#   SPROOT_VERSION      - specific version to install (default: latest release)
#   SPROOT_INSTALL_DIR  - override install directory (default: auto-detect)
#
# Supports: Linux (amd64, arm64), macOS (amd64, arm64)

set -eu

REPO="justanotherspy/sproot"
BINARY="sproot"
RELEASES_URL="https://github.com/${REPO}/releases"
API_URL="https://api.github.com/repos/${REPO}/releases/latest"

log()  { printf -- '- %s\n' "$*"; }
ok()   { printf -- '+ %s\n' "$*"; }
fail() { printf -- 'x %s\n' "$*" >&2; exit 1; }

detect_os() {
    _os="$(uname -s)"
    case "${_os}" in
        Linux)  printf 'linux'  ;;
        Darwin) printf 'darwin' ;;
        *)      fail "Unsupported OS: ${_os}" ;;
    esac
}

detect_arch() {
    _arch="$(uname -m)"
    case "${_arch}" in
        x86_64)  printf 'amd64' ;;
        amd64)   printf 'amd64' ;;
        aarch64) printf 'arm64' ;;
        arm64)   printf 'arm64' ;;
        *)       fail "Unsupported architecture: ${_arch}" ;;
    esac
}

# Preference: SPROOT_INSTALL_DIR > /usr/local/bin (if writable or sudo cached) > ~/.local/bin
resolve_install_dir() {
    if [ -n "${SPROOT_INSTALL_DIR:-}" ]; then
        printf '%s' "${SPROOT_INSTALL_DIR}"
        return
    fi
    if [ -w /usr/local/bin ]; then
        printf '/usr/local/bin'
    elif command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
        printf '/usr/local/bin'
    else
        _fallback="${HOME}/.local/bin"
        mkdir -p "${_fallback}"
        printf '%s' "${_fallback}"
    fi
}

# Download URL to file. Prefers curl, falls back to wget.
download() {
    _url="$1"
    _dest="$2"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL --retry 3 -o "${_dest}" "${_url}"
    elif command -v wget >/dev/null 2>&1; then
        wget -q --tries=3 -O "${_dest}" "${_url}"
    else
        fail "Neither curl nor wget found. Install one and retry."
    fi
}

OS="$(detect_os)"
ARCH="$(detect_arch)"

if [ -n "${SPROOT_VERSION:-}" ]; then
    VERSION="${SPROOT_VERSION}"
    log "Using requested version: ${VERSION}"
else
    log "Fetching latest release version..."
    if command -v curl >/dev/null 2>&1; then
        VERSION="$(curl -fsSL "${API_URL}" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
    else
        VERSION="$(wget -qO- "${API_URL}" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
    fi
    [ -n "${VERSION}" ] || fail "Could not determine latest version. Set SPROOT_VERSION and retry."
    log "Latest version: ${VERSION}"
fi

# Release tags are prefixed with "v" (e.g. v0.1.0), but GoReleaser strips the
# prefix from artifact names (e.g. sproot_0.1.0_linux_amd64.tar.gz). The
# download path uses the tag, the file names use the bare version.
VERSION_NUM="${VERSION#v}"
TAG="v${VERSION_NUM}"

ARCHIVE_NAME="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
CHECKSUM_NAME="${BINARY}_${VERSION_NUM}_checksums.txt"
DOWNLOAD_BASE="${RELEASES_URL}/download/${TAG}"

TMPDIR="$(mktemp -d)"
cleanup() { rm -rf "${TMPDIR}"; }
trap cleanup EXIT

log "Downloading ${ARCHIVE_NAME}..."
download "${DOWNLOAD_BASE}/${ARCHIVE_NAME}"  "${TMPDIR}/${ARCHIVE_NAME}"
download "${DOWNLOAD_BASE}/${CHECKSUM_NAME}" "${TMPDIR}/${CHECKSUM_NAME}"

log "Verifying checksum..."
if command -v sha256sum >/dev/null 2>&1; then
    (cd "${TMPDIR}" && grep "${ARCHIVE_NAME}" "${CHECKSUM_NAME}" | sha256sum -c -)
elif command -v shasum >/dev/null 2>&1; then
    (cd "${TMPDIR}" && grep "${ARCHIVE_NAME}" "${CHECKSUM_NAME}" | shasum -a 256 -c -)
else
    printf '! sha256sum not found, skipping checksum verification\n' >&2
fi

log "Extracting ${BINARY}..."
tar -xzf "${TMPDIR}/${ARCHIVE_NAME}" -C "${TMPDIR}" "${BINARY}"
chmod 755 "${TMPDIR}/${BINARY}"

INSTALL_DIR="$(resolve_install_dir)"
DEST="${INSTALL_DIR}/${BINARY}"

log "Installing to ${DEST}..."
if [ -w "${INSTALL_DIR}" ]; then
    mv "${TMPDIR}/${BINARY}" "${DEST}"
else
    sudo mv "${TMPDIR}/${BINARY}" "${DEST}"
fi

ok "Installed ${BINARY} ${VERSION} to ${DEST}"

if "${DEST}" --version >/dev/null 2>&1; then
    ok "$("${DEST}" --version)"
fi

case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        printf '\n! %s is not in your PATH.\n' "${INSTALL_DIR}"
        printf '  Add this to your shell profile:\n'
        printf '    export PATH="%s:$PATH"\n\n' "${INSTALL_DIR}"
        ;;
esac
