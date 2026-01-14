#!/usr/local/bin/php
<?php
/*
 * register-package.php
 *
 * Registers the ZID Logs package in pfSense's config.xml.
 *
 * Usage: php register-package.php
 */

echo "=========================================\n";
echo " ZID Logs Package Registration\n";
echo "=========================================\n\n";

if (posix_geteuid() !== 0) {
    echo "Error: This script must be run as root\n";
    exit(1);
}

if (!file_exists('/etc/inc/config.inc')) {
    echo "Error: This does not appear to be a pfSense system\n";
    exit(1);
}

require_once('/etc/inc/config.inc');
require_once('/etc/inc/util.inc');

$config = parse_config(true);

if (!is_array($config['installedpackages'])) {
    $config['installedpackages'] = array();
}
if (!is_array($config['installedpackages']['package'])) {
    $config['installedpackages']['package'] = array();
}
if (!is_array($config['installedpackages']['menu'])) {
    $config['installedpackages']['menu'] = array();
}
if (!is_array($config['installedpackages']['zidlogs'])) {
    $config['installedpackages']['zidlogs'] = array();
}
if (!is_array($config['installedpackages']['zidlogs']['config'])) {
    $config['installedpackages']['zidlogs']['config'] = array();
}

function zidlogs_detect_version() {
    $bin = '/usr/local/sbin/zid-logs';
    if (!is_executable($bin)) {
        return '';
    }
    $out = array();
    $rc = 0;
    exec(escapeshellcmd($bin) . " -version 2>&1", $out, $rc);
    if ($rc !== 0 || empty($out)) {
        return '';
    }
    foreach ($out as $line) {
        $line = trim($line);
        if ($line === '') {
            continue;
        }
        if (preg_match('/version\\s+([0-9][0-9A-Za-z\\.-]*)/i', $line, $matches)) {
            return $matches[1];
        }
        return $line;
    }
    return '';
}

foreach ($config['installedpackages']['package'] as $idx => $pkg) {
    if (isset($pkg['name']) && $pkg['name'] == 'zid-logs') {
        unset($config['installedpackages']['package'][$idx]);
    }
}
$config['installedpackages']['package'] = array_values($config['installedpackages']['package']);

$detected_version = zidlogs_detect_version();
if ($detected_version === '') {
    $detected_version = 'unknown';
}

$config['installedpackages']['package'][] = array(
    'name' => 'zid-logs',
    'version' => $detected_version,
    'descr' => 'ZID Logs - rotacao e envio incremental de logs',
    'website' => '',
    'configurationfile' => 'zid-logs.xml',
    'include_file' => '/usr/local/pkg/zid-logs.inc'
);

foreach ($config['installedpackages']['menu'] as $idx => $menu) {
    if (isset($menu['name']) && $menu['name'] === 'ZID Logs') {
        unset($config['installedpackages']['menu'][$idx]);
    }
}
$config['installedpackages']['menu'] = array_values($config['installedpackages']['menu']);

$config['installedpackages']['menu'][] = array(
    'name' => 'ZID Logs',
    'tooltiptext' => 'Centraliza rotacao e envio de logs ZID',
    'section' => 'Services',
    'url' => '/zid-logs_config.php'
);

if (empty($config['installedpackages']['zidlogs']['config'])) {
    $config['installedpackages']['zidlogs']['config'][0] = array(
        'enable' => 'off'
    );
}

write_config("ZID Logs package registered");

echo "Registration written to config.xml\n";

if (file_exists('/usr/local/pkg/zid-logs.inc')) {
    require_once('/usr/local/pkg/zid-logs.inc');
    zidlogs_install();
}

echo "\n=========================================\n";
echo " Registration Complete!\n";
echo "=========================================\n\n";

echo "Reload pfSense web GUI:\n";
echo "  /etc/rc.restart_webgui\n\n";
?>
