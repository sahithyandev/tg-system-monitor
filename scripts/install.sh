#!/bin/sh
# install.sh — one-command installer for tg-system-monitor (tgsm)
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/sahithyandev/tg-system-monitor/main/scripts/install.sh | sudo bash
#
# Override the version to install:
#   VERSION=1.2.3 curl -fsSL .../install.sh | sudo bash

set -e

REPO="sahithyandev/tg-system-monitor"
BINARY_NAME="tgsm"
INSTALL_BIN="/usr/bin/tgsm"
SYSTEMD_UNIT_DIR="/usr/lib/systemd/system"
TGSM_HOME="/var/lib/tgsm"
TGSM_USER="tgsm"
TGSM_GROUP="tgsm"

# ── helpers ────────────────────────────────────────────────────────────────────

die() { echo "ERROR: $*" >&2; exit 1; }
info() { echo "==> $*"; }

require_root() {
    [ "$(id -u)" -eq 0 ] || die "This script must be run as root (use sudo)."
}

http_get() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$1"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$1"
    else
        die "curl or wget is required. Install one and try again."
    fi
}

http_download() {
    local url="$1" dest="$2"
    info "Downloading $(basename "$dest") ..."
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$dest" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$dest" "$url"
    else
        die "curl or wget is required."
    fi
}

# ── version resolution ─────────────────────────────────────────────────────────

resolve_version() {
    if [ -n "${VERSION:-}" ]; then
        # Normalise: strip a leading v so we can re-add it consistently
        VERSION="${VERSION#v}"
        TAG="v${VERSION}"
        info "Using requested version: ${TAG}"
        return
    fi

    info "Fetching latest release ..."
    local json
    json=$(http_get "https://api.github.com/repos/${REPO}/releases/latest")

    TAG=$(echo "$json" | grep '"tag_name"' | head -1 \
        | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')

    [ -n "$TAG" ] || die "Could not determine the latest release. Check your network."
    VERSION="${TAG#v}"
    info "Latest version: ${TAG}"
}

# ── architecture detection ─────────────────────────────────────────────────────

detect_arch() {
    local machine
    machine=$(uname -m)
    case "$machine" in
        x86_64 | amd64)   ARCH="amd64" ; ARCH_LONG="x86_64" ;;
        aarch64 | arm64)  ARCH="arm64" ; ARCH_LONG="arm64" ;;
        *) die "Unsupported architecture: $machine" ;;
    esac
}

# ── download URL builders ──────────────────────────────────────────────────────

deb_url()    { echo "https://github.com/${REPO}/releases/download/${TAG}/tgsm_${VERSION}_linux_${ARCH}.deb"; }
rpm_url()    { echo "https://github.com/${REPO}/releases/download/${TAG}/tgsm_${VERSION}_linux_${ARCH}.rpm"; }
binary_url() { echo "https://github.com/${REPO}/releases/download/${TAG}/tg-system-monitor_Linux_${ARCH_LONG}"; }

# ── package-manager install paths ─────────────────────────────────────────────

install_deb() {
    local url; url=$(deb_url)
    local tmp; tmp=$(mktemp -t tgsm_XXXXXX.deb)
    http_download "$url" "$tmp"
    info "Installing .deb package ..."
    if command -v apt-get >/dev/null 2>&1; then
        apt-get install -y "$tmp"
    else
        dpkg -i "$tmp" || true
        apt-get install -f -y
    fi
    rm -f "$tmp"
}

install_rpm() {
    local url; url=$(rpm_url)
    local tmp; tmp=$(mktemp -t tgsm_XXXXXX.rpm)
    http_download "$url" "$tmp"
    info "Installing .rpm package ..."
    if command -v dnf >/dev/null 2>&1; then
        dnf install -y "$tmp"
    elif command -v yum >/dev/null 2>&1; then
        yum install -y "$tmp"
    else
        rpm -i "$tmp"
    fi
    rm -f "$tmp"
}

# ── raw-binary fallback ────────────────────────────────────────────────────────

setup_user_and_dirs() {
    # Create group
    if ! getent group "${TGSM_GROUP}" >/dev/null 2>&1; then
        groupadd --system "${TGSM_GROUP}"
    fi
    # Create user
    if ! getent passwd "${TGSM_USER}" >/dev/null 2>&1; then
        useradd \
            --system \
            --gid "${TGSM_GROUP}" \
            --home-dir "${TGSM_HOME}" \
            --no-create-home \
            --shell /usr/sbin/nologin \
            --comment "tg-system-monitor service account" \
            "${TGSM_USER}"
    fi
    install -d -m 0750 -o "${TGSM_USER}" -g "${TGSM_GROUP}" "${TGSM_HOME}"
    local config_dir="${TGSM_HOME}/.config/tg-system-monitor"
    install -d -m 0750 -o "${TGSM_USER}" -g "${TGSM_GROUP}" "$config_dir"
}

install_binary_fallback() {
    local url; url=$(binary_url)
    info "No package manager found; installing raw binary ..."

    # Download binary
    local tmp; tmp=$(mktemp -t tgsm_XXXXXX)
    http_download "$url" "$tmp"
    chmod 755 "$tmp"
    mv "$tmp" "${INSTALL_BIN}"
    info "Binary installed to ${INSTALL_BIN}"

    # User/dirs
    setup_user_and_dirs

    # Config
    local config_file="${TGSM_HOME}/.config/tg-system-monitor/config.yml"
    if [ ! -f "$config_file" ]; then
        local example_url="https://raw.githubusercontent.com/${REPO}/main/default-config.yml"
        info "Downloading default config ..."
        http_download "$example_url" "$config_file"
        chown "${TGSM_USER}:${TGSM_GROUP}" "$config_file"
        chmod 640 "$config_file"
    fi

    # Systemd unit
    mkdir -p "${SYSTEMD_UNIT_DIR}"
    local unit_url="https://raw.githubusercontent.com/${REPO}/main/tgsm.service"
    info "Installing systemd unit ..."
    http_download "$unit_url" "${SYSTEMD_UNIT_DIR}/tgsm.service"
    chmod 644 "${SYSTEMD_UNIT_DIR}/tgsm.service"

    # Enable service
    if command -v systemctl >/dev/null 2>&1; then
        systemctl daemon-reload
        systemctl enable tgsm
    fi
}

# ── next-steps message ─────────────────────────────────────────────────────────

print_next_steps() {
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║               tg-system-monitor installed!                  ║"
    echo "╠══════════════════════════════════════════════════════════════╣"
    echo "║  Next steps:                                                 ║"
    echo "║                                                              ║"
    echo "║  1. Set your Telegram bot token in:                          ║"
    echo "║     ${TGSM_HOME}/.config/tg-system-monitor/config.yml"
    echo "║                                                              ║"
    echo "║  2. Set the join password:                                   ║"
    echo "║     sudo -u tgsm HOME=${TGSM_HOME} tgsm set-password         ║"
    echo "║                                                              ║"
    echo "║  3. Start the service:                                       ║"
    echo "║     sudo systemctl start tgsm                                ║"
    echo "║                                                              ║"
    echo "║  Check logs:  journalctl -u tgsm -f                          ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
}

# ── main ───────────────────────────────────────────────────────────────────────

main() {
    require_root

    local os; os=$(uname -s)
    [ "$os" = "Linux" ] || die "This installer supports Linux only (detected: $os)."

    resolve_version
    detect_arch

    if command -v dpkg >/dev/null 2>&1; then
        install_deb
    elif command -v rpm >/dev/null 2>&1; then
        install_rpm
    else
        install_binary_fallback
        print_next_steps
    fi
}

main "$@"
