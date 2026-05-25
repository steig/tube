#!/bin/sh
# tube — local development proxy with .test domains
# Install script. Run with:
#   curl -fsSL https://raw.githubusercontent.com/steig/tube/main/scripts/install.sh | sh
#
# Environment overrides:
#   TUBE_VERSION=vX.Y.Z   — install a specific version (default: latest release)
#   TUBE_PREFIX=/path     — install root (default: /usr/local)
#   TUBE_REPO=owner/repo  — github repo (default: steig/tube)
#   TUBE_NO_COLOR=1       — disable colored output
#
# Flags (only when invoked as `sh install.sh` directly):
#   --version <ver>
#   --prefix <path>
#   --yes                 — non-interactive (skip sudo prompt; assume yes)
#   -h, --help

set -eu

# ---------- defaults ----------
: "${TUBE_REPO:=steig/tube}"
: "${TUBE_PREFIX:=/usr/local}"
: "${TUBE_VERSION:=}"
ASSUME_YES=0

# ---------- arg parsing ----------
while [ $# -gt 0 ]; do
    case "$1" in
        --version) TUBE_VERSION="$2"; shift 2 ;;
        --version=*) TUBE_VERSION="${1#*=}"; shift ;;
        --prefix) TUBE_PREFIX="$2"; shift 2 ;;
        --prefix=*) TUBE_PREFIX="${1#*=}"; shift ;;
        --yes|-y) ASSUME_YES=1; shift ;;
        -h|--help)
            sed -n '2,16p' "$0" | sed 's/^# \{0,1\}//'
            exit 0
            ;;
        *) printf 'unknown argument: %s\n' "$1" >&2; exit 2 ;;
    esac
done

# ---------- colors ----------
if [ -t 1 ] && [ -z "${TUBE_NO_COLOR:-}" ] && command -v tput >/dev/null 2>&1; then
    BOLD=$(tput bold)
    DIM=$(tput dim)
    RED=$(tput setaf 1)
    GREEN=$(tput setaf 2)
    YELLOW=$(tput setaf 3)
    BLUE=$(tput setaf 4)
    MAGENTA=$(tput setaf 5)
    RESET=$(tput sgr0)
else
    BOLD=""; DIM=""; RED=""; GREEN=""; YELLOW=""; BLUE=""; MAGENTA=""; RESET=""
fi

info()    { printf '%s==>%s %s\n' "$BLUE" "$RESET" "$*"; }
success() { printf '%s ✓%s %s\n' "$GREEN" "$RESET" "$*"; }
warn()    { printf '%s ⚠%s %s\n' "$YELLOW" "$RESET" "$*" >&2; }
fail()    { printf '%s ✗%s %s\n' "$RED" "$RESET" "$*" >&2; exit 1; }
step()    { printf '   %s%s%s\n' "$DIM" "$*" "$RESET"; }

banner() {
    cat <<EOF
${MAGENTA}${BOLD}
   ┌─┐ ┬ ┬ ┌┐ ┌─┐
   │ │ │ │ ├┴┐├┤
   ┴ ┴ └─┘ └─┘└─┘${RESET}
   ${DIM}local development proxy${RESET}

EOF
}

# ---------- dependency checks ----------
need() {
    command -v "$1" >/dev/null 2>&1 || fail "missing required tool: $1"
}

need uname
need mkdir
need rm
need mv
need tar
if command -v curl >/dev/null 2>&1; then
    DOWNLOADER=curl
elif command -v wget >/dev/null 2>&1; then
    DOWNLOADER=wget
else
    fail "need curl or wget on PATH"
fi

# Prefer shasum (macOS default) then sha256sum (Linux default).
if command -v sha256sum >/dev/null 2>&1; then
    SHA256="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
    SHA256="shasum -a 256"
else
    warn "no sha256sum/shasum found — install will skip checksum verification"
    SHA256=""
fi

# ---------- platform detection ----------
detect_os() {
    case "$(uname -s)" in
        Darwin) echo darwin ;;
        Linux)  echo linux ;;
        *) fail "unsupported OS: $(uname -s) — tube only ships darwin and linux binaries" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        arm64|aarch64) echo arm64 ;;
        x86_64|amd64)  echo amd64 ;;
        *) fail "unsupported architecture: $(uname -m) — tube ships amd64 and arm64 only" ;;
    esac
}

OS=$(detect_os)
ARCH=$(detect_arch)

# ---------- download helpers ----------
http_get() {
    # http_get URL OUT — downloads URL to OUT path
    if [ "$DOWNLOADER" = curl ]; then
        curl -fsSL --retry 3 --retry-delay 1 -o "$2" "$1"
    else
        wget -qO "$2" --tries=3 "$1"
    fi
}

http_get_stdout() {
    if [ "$DOWNLOADER" = curl ]; then
        curl -fsSL --retry 3 --retry-delay 1 "$1"
    else
        wget -qO - --tries=3 "$1"
    fi
}

# ---------- version resolution ----------
resolve_version() {
    if [ -n "$TUBE_VERSION" ]; then
        echo "$TUBE_VERSION"
        return
    fi
    # GitHub redirects /releases/latest to the actual tag URL; parse Location.
    # Avoids needing jq and survives anonymous rate limits better than the API.
    url="https://github.com/${TUBE_REPO}/releases/latest"
    if [ "$DOWNLOADER" = curl ]; then
        tag=$(curl -fsSLI -o /dev/null -w '%{url_effective}\n' "$url" | sed 's#.*/tag/##')
    else
        tag=$(wget --max-redirect=0 -S -O /dev/null "$url" 2>&1 | sed -n 's#.*Location: .*/tag/\([^ ]*\).*#\1#p' | head -1)
    fi
    [ -n "$tag" ] || fail "could not resolve latest release; pin a version with TUBE_VERSION=vX.Y.Z"
    echo "$tag"
}

# ---------- main install ----------
banner

info "Detecting platform"
step "OS:   $OS"
step "Arch: $ARCH"

info "Resolving version"
VERSION=$(resolve_version)
step "Version: $VERSION"
NAKED_VERSION=${VERSION#v}

ARCHIVE="tube_${NAKED_VERSION}_${OS}_${ARCH}.tar.gz"
ARCHIVE_URL="https://github.com/${TUBE_REPO}/releases/download/${VERSION}/${ARCHIVE}"
CHECKSUM_URL="https://github.com/${TUBE_REPO}/releases/download/${VERSION}/checksums.txt"

# Skip if the same version is already installed.
# `tube --version` prints "tube version X.Y.Z (commit: ..., date: ...)".
INSTALL_BIN="${TUBE_PREFIX}/bin/tube"
if [ -x "$INSTALL_BIN" ]; then
    current=$("$INSTALL_BIN" --version 2>/dev/null | awk '{print $3}' || true)
    if [ "$current" = "$NAKED_VERSION" ]; then
        success "tube $VERSION already installed at $INSTALL_BIN"
        exit 0
    fi
    step "Replacing existing tube ($current) with $NAKED_VERSION"
fi

# Use mktemp if available; fall back to a deterministic path.
TMPDIR=$(mktemp -d 2>/dev/null || mktemp -d -t tube-install)
trap 'rm -rf "$TMPDIR"' EXIT INT TERM

info "Downloading $ARCHIVE"
http_get "$ARCHIVE_URL" "$TMPDIR/$ARCHIVE" || fail "download failed: $ARCHIVE_URL"
step "Saved to $TMPDIR/$ARCHIVE"

if [ -n "$SHA256" ]; then
    info "Verifying checksum"
    http_get "$CHECKSUM_URL" "$TMPDIR/checksums.txt" || fail "could not download checksums.txt"
    expected=$(grep " $ARCHIVE\$" "$TMPDIR/checksums.txt" | awk '{print $1}')
    [ -n "$expected" ] || fail "no checksum entry for $ARCHIVE"
    actual=$(cd "$TMPDIR" && $SHA256 "$ARCHIVE" | awk '{print $1}')
    [ "$expected" = "$actual" ] || fail "checksum mismatch: expected $expected, got $actual"
    step "SHA256 OK"
fi

info "Extracting"
# Extract the whole archive — older archives shipped tube only, newer darwin
# archives also include tube-gui. The two `tar t` lookups below decide which
# binaries we actually install.
tar -C "$TMPDIR" -xzf "$TMPDIR/$ARCHIVE"
[ -f "$TMPDIR/tube" ] || fail "archive did not contain a 'tube' binary"
chmod +x "$TMPDIR/tube"

HAS_GUI=0
if [ -f "$TMPDIR/tube-gui" ]; then
    chmod +x "$TMPDIR/tube-gui"
    HAS_GUI=1
fi

# Decide whether we need sudo. Try a no-op write to the target dir; if it
# fails, fall back to sudo. Skip the elevated path entirely when --yes is set
# and we can't write (so curl|bash doesn't hang prompting for a password).
INSTALL_DIR="${TUBE_PREFIX}/bin"
mkdir -p "$INSTALL_DIR" 2>/dev/null || true
if [ -w "$INSTALL_DIR" ]; then
    SUDO=""
elif [ "$ASSUME_YES" -eq 1 ]; then
    fail "$INSTALL_DIR is not writable and --yes is set; rerun with TUBE_PREFIX=\$HOME/.local"
else
    if command -v sudo >/dev/null 2>&1; then
        info "Installation directory $INSTALL_DIR requires elevated privileges"
        SUDO="sudo"
    else
        fail "$INSTALL_DIR not writable and sudo not available"
    fi
fi

info "Installing to $INSTALL_DIR/tube"
$SUDO mv "$TMPDIR/tube" "$INSTALL_DIR/tube"
$SUDO chmod 0755 "$INSTALL_DIR/tube"

success "Installed tube $VERSION to $INSTALL_DIR/tube"

if [ "$HAS_GUI" -eq 1 ]; then
    $SUDO mv "$TMPDIR/tube-gui" "$INSTALL_DIR/tube-gui"
    $SUDO chmod 0755 "$INSTALL_DIR/tube-gui"
    success "Installed tube-gui to $INSTALL_DIR/tube-gui"
fi

# Sanity: verify the binary on PATH matches what we just wrote.
if ! command -v tube >/dev/null 2>&1; then
    warn "tube is installed but not on PATH. Add this to your shell rc:"
    # shellcheck disable=SC2016  # literal $PATH on purpose — user pastes into rc
    printf '    export PATH="%s:$PATH"\n' "$INSTALL_DIR"
fi

# ---------- next steps ----------
cat <<EOF

${BOLD}Next steps:${RESET}

  1. Install runtime dependencies:
       ${DIM}brew install nginx dnsmasq mkcert${RESET}

  2. Initialize tube and generate SSL certs:
       ${DIM}tube init${RESET}

  3. Set up macOS DNS resolver (one-time, needs sudo):
       ${DIM}tube setup${RESET}

  4. Add a project and start services:
       ${DIM}tube add myapp 3000${RESET}
       ${DIM}tube start${RESET}
EOF

if [ "$HAS_GUI" -eq 1 ]; then
    cat <<EOF

  5. Launch the menu bar app (optional):
       ${DIM}tube-gui &${RESET}
EOF
fi

cat <<EOF

  Check status anytime with: ${DIM}tube doctor${RESET}
  Docs: ${BLUE}https://steig.github.io/tube/${RESET}

EOF
