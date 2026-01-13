#!/bin/sh
#
# uninstall.sh
#
# Uninstalls ZID Logs from pfSense
#
# Usage: ./uninstall.sh
#

set -e

echo "========================================="
echo " ZID Logs Uninstaller"
echo "========================================="
echo ""

if [ "$(id -u)" != "0" ]; then
    echo "Error: This script must be run as root"
    exit 1
fi

echo "This will remove ZID Logs and all its files from your system."
echo ""
read -p "Are you sure you want to continue? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "Uninstallation cancelled."
    exit 0
fi

echo ""
echo "Starting uninstallation..."
echo ""

echo "Stopping service..."
if [ -f /usr/local/etc/rc.d/zid_logs ]; then
    /usr/local/etc/rc.d/zid_logs stop 2>/dev/null || true
fi

if [ -f /var/run/zid-logs.pid ]; then
    pid=$(cat /var/run/zid-logs.pid)
    if kill -0 "$pid" 2>/dev/null; then
        echo "Killing process $pid..."
        kill "$pid" 2>/dev/null || true
        sleep 1
    fi
fi

echo "Removing binary..."
rm -f /usr/local/sbin/zid-logs

rm -f /usr/local/sbin/zid-logs-update

echo "Removing web interface files..."
rm -f /usr/local/www/zid-logs_config.php
rm -f /usr/local/www/zid-logs_status.php
rm -f /usr/local/www/zid-logs_inputs.php

echo "Removing RC scripts..."
rm -f /usr/local/etc/rc.d/zid_logs

rm -f /var/run/zid-logs.pid

echo ""
read -p "Remove configuration directory (/usr/local/etc/zid-logs)? (yes/no): " remove_config
if [ "$remove_config" = "yes" ]; then
    rm -rf /usr/local/etc/zid-logs
fi

read -p "Remove state directory (/var/db/zid-logs)? (yes/no): " remove_state
if [ "$remove_state" = "yes" ]; then
    rm -rf /var/db/zid-logs
fi

read -p "Remove log file (/var/log/zid-logs.log)? (yes/no): " remove_log
if [ "$remove_log" = "yes" ]; then
    rm -f /var/log/zid-logs.log
fi

echo "Removing rc.conf entries..."
if [ -f /etc/rc.conf.local ]; then
    sed -i.bak '/zid_logs/d' /etc/rc.conf.local
fi
if [ -f /etc/rc.conf ]; then
    sed -i.bak '/zid_logs/d' /etc/rc.conf
fi

echo ""
echo "========================================="
echo " Uninstallation Complete!"
echo "========================================="
