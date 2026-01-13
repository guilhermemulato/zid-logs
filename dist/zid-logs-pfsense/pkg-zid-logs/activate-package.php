#!/usr/local/bin/php
<?php
/*
 * activate-package.php
 *
 * Activates the ZID Logs package by executing installation hooks.
 *
 * Usage: php activate-package.php
 */

echo "=========================================\n";
echo " ZID Logs Package Activation\n";
echo "=========================================\n\n";

if (posix_geteuid() !== 0) {
    echo "Error: This script must be run as root\n";
    exit(1);
}

if (!file_exists('/usr/local/pkg/zid-logs.inc')) {
    echo "Error: Package files not found. Please run install.sh first.\n";
    exit(1);
}

echo "Loading package functions...\n";
require_once('/usr/local/pkg/zid-logs.inc');

echo "Executing installation hook...\n";
$result = zidlogs_install();

if ($result === false) {
    echo "\nError: Installation hook failed!\n";
    exit(1);
}

echo "\nInstallation hook executed successfully!\n\n";

$rcfile = '/usr/local/etc/rc.d/zid_logs';
if (file_exists($rcfile)) {
    echo "[OK] RC script found: {$rcfile}\n";
    chmod($rcfile, 0755);
} else {
    echo "[WARN] RC script not found at {$rcfile}\n";
}

$config_dir = '/usr/local/etc/zid-logs';
if (is_dir($config_dir)) {
    echo "[OK] Config directory created: {$config_dir}\n";
} else {
    echo "[WARN] Config directory not found at {$config_dir}\n";
}

$inputs_dir = '/var/db/zid-logs/inputs.d';
if (is_dir($inputs_dir)) {
    echo "[OK] Inputs directory created: {$inputs_dir}\n";
} else {
    echo "[WARN] Inputs directory not found at {$inputs_dir}\n";
}

echo "\n=========================================\n";
echo " Activation Complete!\n";
echo "=========================================\n\n";

echo "Note: The pfSense web interface may not show the 'Services > ZID Logs'\n";
echo "menu until you run register-package.php or restart pfSense.\n\n";
?>
