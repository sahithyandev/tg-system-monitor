#!/bin/sh
# preremove.sh — runs before the package files are removed from disk.
# deb: $1 = "remove" (pure removal) | "upgrade" (before upgrade) | "deconfigure"
# rpm: $1 = 0 (remove) | 1 (upgrade)

set -e

is_removal() {
    # deb: $1 = "remove"; rpm: $1 = 0
    [ "$1" = "remove" ] || [ "$1" = "0" ]
}

if is_removal "$1"; then
    if command -v systemctl >/dev/null 2>&1; then
        if systemctl is-active --quiet tgsm 2>/dev/null; then
            systemctl stop tgsm || true
        fi
        if systemctl is-enabled --quiet tgsm 2>/dev/null; then
            systemctl disable tgsm || true
        fi
    fi
fi
