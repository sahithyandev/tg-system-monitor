#!/bin/sh
# postremove.sh — runs after the package files are removed from disk.
# deb: $1 = "remove" | "purge" | "upgrade"
# rpm: $1 = 0 (remove) | 1 (upgrade)

set -e

# Reload systemd so the removed unit is no longer known
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload || true
fi

# NOTE: We intentionally do NOT delete /var/lib/tgsm or the tgsm user.
# This preserves the database and configuration across reinstalls/upgrades.
# To fully purge, run:
#   userdel tgsm && groupdel tgsm && rm -rf /var/lib/tgsm
