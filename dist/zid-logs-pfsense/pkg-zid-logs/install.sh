#!/bin/sh
#
# ZID Logs pfSense Package Installation Script
#
# Usage: ./install.sh
#

set -e

echo "========================================="
echo " ZID Logs pfSense Package Installer"
echo "========================================="

if [ "$(id -u)" != "0" ]; then
    echo "Error: This script must be run as root"
    exit 1
fi

PREFIX="/usr/local"
PKG_DIR="$(dirname "$0")"
FILES_DIR="${PKG_DIR}/files"

echo ""
echo "Installing from: ${PKG_DIR}"
echo ""

echo "Creating directories..."
mkdir -p ${PREFIX}/www
mkdir -p ${PREFIX}/pkg
mkdir -p ${PREFIX}/etc/rc.d
mkdir -p ${PREFIX}/sbin
mkdir -p ${PREFIX}/etc/zid-logs
mkdir -p ${PREFIX}/share/pfSense-pkg-zid-logs
mkdir -p /var/db/zid-logs/inputs.d
mkdir -p /var/log

# Install web pages
echo "Installing web pages..."
cp -f ${FILES_DIR}${PREFIX}/www/zid-logs_config.php ${PREFIX}/www/ 2>/dev/null || true
cp -f ${FILES_DIR}${PREFIX}/www/zid-logs_status.php ${PREFIX}/www/ 2>/dev/null || true
cp -f ${FILES_DIR}${PREFIX}/www/zid-logs_inputs.php ${PREFIX}/www/ 2>/dev/null || true

# Install package configuration
echo "Installing package configuration..."
cp -f ${FILES_DIR}${PREFIX}/pkg/zid-logs.xml ${PREFIX}/pkg/ 2>/dev/null || true
cp -f ${FILES_DIR}${PREFIX}/pkg/zid-logs.inc ${PREFIX}/pkg/ 2>/dev/null || true

# Install rc.d script
echo "Installing rc.d script..."
cp -f ${FILES_DIR}${PREFIX}/etc/rc.d/zid_logs ${PREFIX}/etc/rc.d/ 2>/dev/null || true
chmod 755 ${PREFIX}/etc/rc.d/zid_logs 2>/dev/null || true

# Install updater helper
if [ -f "${PKG_DIR}/update-bootstrap.sh" ]; then
    echo "Installing updater helper..."
    TMP_UPDATER="${PREFIX}/sbin/.zid-logs-update.new.$$"
    cp "${PKG_DIR}/update-bootstrap.sh" "${TMP_UPDATER}"
    chmod 755 "${TMP_UPDATER}"
    mv -f "${TMP_UPDATER}" "${PREFIX}/sbin/zid-logs-update"

    TMP_UPDATER_INFO="${PREFIX}/share/pfSense-pkg-zid-logs/.zid-logs-update.new.$$"
    cp "${PKG_DIR}/update-bootstrap.sh" "${TMP_UPDATER_INFO}"
    chmod 755 "${TMP_UPDATER_INFO}"
    mv -f "${TMP_UPDATER_INFO}" "${PREFIX}/share/pfSense-pkg-zid-logs/zid-logs-update"
fi

# Install binary
BINARY_PATH="${PKG_DIR}/../build/zid-logs"
if [ -f "${BINARY_PATH}" ]; then
    echo "Installing binary..."
    TMP_BIN="${PREFIX}/sbin/.zid-logs.new.$$"
    cp "${BINARY_PATH}" "${TMP_BIN}"
    chmod 755 "${TMP_BIN}"
    mv -f "${TMP_BIN}" "${PREFIX}/sbin/zid-logs"
    chmod 755 ${PREFIX}/sbin/zid-logs
else
    echo "Warning: Binary not found at ${BINARY_PATH}"
    echo "         You need to copy the zid-logs binary to ${PREFIX}/sbin/ manually"
fi

# Create log file
touch /var/log/zid-logs.log
chmod 644 /var/log/zid-logs.log

# Set permissions
echo "Setting permissions..."
if [ -f ${PREFIX}/www/zid-logs_config.php ]; then
    chmod 644 ${PREFIX}/www/zid-logs_config.php
fi
if [ -f ${PREFIX}/www/zid-logs_status.php ]; then
    chmod 644 ${PREFIX}/www/zid-logs_status.php
fi
if [ -f ${PREFIX}/www/zid-logs_inputs.php ]; then
    chmod 644 ${PREFIX}/www/zid-logs_inputs.php
fi
if [ -f ${PREFIX}/pkg/zid-logs.xml ]; then
    chmod 644 ${PREFIX}/pkg/zid-logs.xml
fi
if [ -f ${PREFIX}/pkg/zid-logs.inc ]; then
    chmod 644 ${PREFIX}/pkg/zid-logs.inc
fi

# Reload GUI to show menu
if [ -x /usr/local/sbin/pfSsh.php ]; then
    /usr/local/sbin/pfSsh.php playback reloadwebgui >/dev/null 2>&1 || true
elif [ -x /etc/rc.restart_webgui ]; then
    /etc/rc.restart_webgui >/dev/null 2>&1 || true
elif [ -x /usr/local/etc/rc.d/php-fpm ]; then
    /usr/local/etc/rc.d/php-fpm restart >/dev/null 2>&1 || true
fi

# Activate and register package
SCRIPT_DIR="$(dirname "$0")"
if [ -f "${SCRIPT_DIR}/activate-package.php" ]; then
    php "${SCRIPT_DIR}/activate-package.php" || true
fi
if [ -f "${SCRIPT_DIR}/register-package.php" ]; then
    php "${SCRIPT_DIR}/register-package.php" || true
fi

echo ""
echo "========================================="
echo " File Installation Complete!"
echo "========================================="
echo ""

echo "Files installed:"
echo "  • Binary: ${PREFIX}/sbin/zid-logs"
echo "  • Web interface: ${PREFIX}/www/zid-logs_*.php"
echo "  • RC script: ${PREFIX}/etc/rc.d/zid_logs"
echo "  • Updater: ${PREFIX}/sbin/zid-logs-update"
