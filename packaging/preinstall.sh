#!/bin/sh
# preinstall.sh — runs before the package files are placed on disk.
# Called on both fresh install and upgrade.
# deb: $1 = "install" | "upgrade"
# rpm: $1 = 1 (install/upgrade)

set -e

TGSM_USER=tgsm
TGSM_GROUP=tgsm
TGSM_HOME=/var/lib/tgsm

# Create group if it doesn't exist
if ! getent group "${TGSM_GROUP}" >/dev/null 2>&1; then
    groupadd --system "${TGSM_GROUP}"
fi

# Create user if it doesn't exist
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

# Create the data/home directory with correct ownership
install -d -m 0750 -o "${TGSM_USER}" -g "${TGSM_GROUP}" "${TGSM_HOME}"
