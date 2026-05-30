#!/bin/sh
# postinstall.sh — runs after the package files are placed on disk.
# Called on both fresh install and upgrade.
# deb: $1 = "configure"
# rpm: $1 = 1 (install) | 2 (upgrade)

set -e

TGSM_USER=tgsm
TGSM_GROUP=tgsm
TGSM_HOME=/var/lib/tgsm
CONFIG_DIR="${TGSM_HOME}/.config/tg-system-monitor"
CONFIG_FILE="${CONFIG_DIR}/config.yml"
CONFIG_EXAMPLE="/usr/share/tgsm/config.yml.example"

# Ensure the config directory exists and is owned by tgsm
install -d -m 0750 -o "${TGSM_USER}" -g "${TGSM_GROUP}" "${CONFIG_DIR}"

# On fresh install: copy the example config if no config exists yet
if [ ! -f "${CONFIG_FILE}" ] && [ -f "${CONFIG_EXAMPLE}" ]; then
    install -m 0640 -o "${TGSM_USER}" -g "${TGSM_GROUP}" "${CONFIG_EXAMPLE}" "${CONFIG_FILE}"
fi

# Ensure ownership is correct (safe to run on upgrade too)
chown -R "${TGSM_USER}:${TGSM_GROUP}" "${TGSM_HOME}"

# Reload systemd and enable the service
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload

    # Enable for boot (idempotent)
    systemctl enable tgsm

    # On upgrade: if service was already running, restart it to pick up the new binary
    if systemctl is-active --quiet tgsm; then
        systemctl restart tgsm
        echo ""
        echo "tgsm has been restarted with the new version."
    else
        # Fresh install — print next-steps
        echo ""
        echo "╔══════════════════════════════════════════════════════════════╗"
        echo "║               tg-system-monitor installed!                  ║"
        echo "╠══════════════════════════════════════════════════════════════╣"
        echo "║  Next steps:                                                 ║"
        echo "║                                                              ║"
        echo "║  1. Set your Telegram bot token in:                          ║"
        echo "║       ${CONFIG_FILE}"
        echo "║                                                              ║"
        echo "║  2. Set the join password:                                   ║"
        echo "║       sudo -u tgsm HOME=${TGSM_HOME} tgsm set-password       ║"
        echo "║                                                              ║"
        echo "║  3. Start the service:                                       ║"
        echo "║       sudo systemctl start tgsm                              ║"
        echo "║                                                              ║"
        echo "║  Check logs:  journalctl -u tgsm -f                          ║"
        echo "╚══════════════════════════════════════════════════════════════╝"
        echo ""
    fi
fi
