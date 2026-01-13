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

function zidlogs_tabs($active = 'status') {
	$tabs = [];
	$tabs[] = [gettext('Status'), $active === 'status', '/zid-logs_status.php'];
	$tabs[] = [gettext('Config'), $active === 'config', '/zid-logs_config.php'];
	$tabs[] = [gettext('Inputs'), $active === 'inputs', '/zid-logs_inputs.php'];
	return $tabs;
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
