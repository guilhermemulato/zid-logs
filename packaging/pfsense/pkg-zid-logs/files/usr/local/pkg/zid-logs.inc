<?php
/*
 * zid-logs.inc
 *
 * PHP helper functions for ZID Logs pfSense package
 */

require_once("config.inc");
require_once("functions.inc");
require_once("util.inc");

define('ZIDLOGS_CONFIG_DIR', '/usr/local/etc/zid-logs');
define('ZIDLOGS_INPUTS_DIR', '/var/db/zid-logs/inputs.d');
define('ZIDLOGS_LOG_FILE', '/var/log/zid-logs.log');
define('ZIDLOGS_PID_FILE', '/var/run/zid-logs.pid');
define('ZIDLOGS_RCFILE', '/usr/local/etc/rc.d/zid_logs');
define('ZIDLOGS_BINARY', '/usr/local/sbin/zid-logs');

function zidlogs_set_rc_enable($enabled) {
	$path = '/etc/rc.conf.local';
	$value = $enabled ? 'YES' : 'NO';
	$line = 'zid_logs_enable="' . $value . '"';
	$lines = array();
	$found = false;

	if (file_exists($path)) {
		$lines = file($path, FILE_IGNORE_NEW_LINES);
		if (!is_array($lines)) {
			$lines = array();
		}
	}

	foreach ($lines as $idx => $row) {
		if (preg_match('/^zid_logs_enable=/', $row)) {
			$lines[$idx] = $line;
			$found = true;
		}
	}

	if (!$found) {
		$lines[] = $line;
	}

	file_put_contents($path, implode("\n", $lines) . "\n");
	@chmod($path, 0644);
}

function zidlogs_tabs($active = 'status') {
	$tabs = [];
	$tabs[] = [gettext('Settings'), $active === 'settings', '/zid-logs_config.php'];
	$tabs[] = [gettext('Status'), $active === 'status', '/zid-logs_status.php'];
	$tabs[] = [gettext('Inputs'), $active === 'inputs', '/zid-logs_inputs.php'];
	return $tabs;
}

function zidlogs_get_pids() {
	$output = array();
	$ret = 0;
	exec('/usr/bin/pgrep -f "^/usr/local/sbin/zid-logs" 2>/dev/null', $output, $ret);
	if ($ret !== 0 || empty($output)) {
		return array();
	}
	$pids = array();
	foreach ($output as $pid) {
		$pid = trim($pid);
		if (is_numeric($pid)) {
			$pids[] = $pid;
		}
	}
	return $pids;
}

function zidlogs_status() {
	$pids = zidlogs_get_pids();
	return !empty($pids);
}

function zidlogs_start() {
	if (is_executable('/usr/local/etc/rc.d/zid_logs')) {
		exec('/usr/local/etc/rc.d/zid_logs onestart 2>/dev/null');
		return;
	}
	exec('service zid_logs onestart 2>/dev/null');
}

function zidlogs_stop() {
	if (is_executable('/usr/local/etc/rc.d/zid_logs')) {
		exec('/usr/local/etc/rc.d/zid_logs onestop 2>/dev/null');
	} else {
		exec('service zid_logs onestop 2>/dev/null');
	}

	$pids = zidlogs_get_pids();
	if (!empty($pids)) {
		exec('kill ' . implode(' ', $pids) . ' 2>/dev/null');
	}
}

function zidlogs_reload() {
	if (file_exists(ZIDLOGS_PID_FILE)) {
		$pid = trim((string)@file_get_contents(ZIDLOGS_PID_FILE));
		if ($pid !== '' && ctype_digit($pid)) {
			exec('kill -HUP ' . $pid . ' 2>/dev/null');
		}
	}
}

function zidlogs_install() {
	@mkdir(ZIDLOGS_CONFIG_DIR, 0755, true);
	@mkdir(ZIDLOGS_INPUTS_DIR, 0755, true);
	if (!file_exists(ZIDLOGS_LOG_FILE)) {
		@touch(ZIDLOGS_LOG_FILE);
		@chmod(ZIDLOGS_LOG_FILE, 0644);
	}
	return true;
}

function zidlogs_deinstall() {
	if (file_exists(ZIDLOGS_RCFILE)) {
		@exec(ZIDLOGS_RCFILE . ' stop >/dev/null 2>&1');
	}
	return true;
}

function zidlogs_resync() {
	return true;
}

?>
