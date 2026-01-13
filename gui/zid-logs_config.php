<?php
require_once('guiconfig.inc');
require_once('/usr/local/pkg/zid-logs.inc');

$config_path = '/usr/local/etc/zid-logs/config.json';

$defaults = array(
    'enabled' => false,
    'endpoint' => '',
    'auth_token' => '',
    'auth_header_name' => 'x-auth-n8n',
    'device_id' => '',
    'interval_rotate_seconds' => 300,
    'interval_ship_seconds' => 60,
    'max_bytes_per_ship' => 262144,
    'ship_format' => 'lines',
    'defaults' => array(
        'max_size_mb' => 50,
        'keep' => 10,
        'compress' => true,
        'rotate_on_start' => false,
    ),
);

function load_config_file($path, $defaults) {
    if (!file_exists($path)) {
        return $defaults;
    }
    $data = file_get_contents($path);
    $json = json_decode($data, true);
    if (!is_array($json)) {
        return $defaults;
    }
    return array_replace_recursive($defaults, $json);
}

function save_config_file($path, $data) {
    $dir = dirname($path);
    if (!is_dir($dir)) {
        @mkdir($dir, 0755, true);
    }
    $json = json_encode($data, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES);
    file_put_contents($path, $json . "\n");
}

$input_errors = array();
$savemsg = '';
$update_msg = '';

function zidlogs_installed_version_line() {
    $bin = '/usr/local/sbin/zid-logs';
    if (!is_executable($bin)) {
        return 'Not installed';
    }
    $out = array();
    $rc = 0;
    exec(escapeshellcmd($bin) . " -version 2>&1", $out, $rc);
    if ($rc !== 0 || empty($out)) {
        return 'Unknown';
    }
    return trim($out[0]);
}

if ($_POST) {
    if (isset($_POST['svc_start'])) {
        zidlogs_start();
        $savemsg = 'Service start requested.';
    } elseif (isset($_POST['svc_stop'])) {
        zidlogs_stop();
        $savemsg = 'Service stop requested.';
    } elseif (isset($_POST['svc_restart'])) {
        zidlogs_stop();
        sleep(1);
        zidlogs_start();
        $savemsg = 'Service restart requested.';
    } elseif (isset($_POST['run_update'])) {
        $cmd = "/bin/sh /usr/local/sbin/zid-logs-update 2>&1";
        $out = array();
        $rc = 0;
        exec($cmd, $out, $rc);
        $joined = trim(implode("\n", $out));
        if (stripos($joined, "Already up-to-date") !== false) {
            $update_msg = $joined;
        } elseif ($rc === 0) {
            $update_msg = "done";
        } else {
            $update_msg = $joined !== '' ? $joined : sprintf("Update failed (exit %d).", $rc);
        }
        zidlogs_stop();
        sleep(1);
        zidlogs_start();
    } else {
        $cfg = load_config_file($config_path, $defaults);

        $cfg['enabled'] = isset($_POST['enabled']);
        $cfg['endpoint'] = trim($_POST['endpoint']);
        $cfg['auth_token'] = trim($_POST['auth_token']);
        $cfg['auth_header_name'] = trim($_POST['auth_header_name']);
        $cfg['interval_rotate_seconds'] = intval($_POST['interval_rotate_seconds']);
        $cfg['interval_ship_seconds'] = intval($_POST['interval_ship_seconds']);
        $cfg['max_bytes_per_ship'] = intval($_POST['max_bytes_per_ship']);
        $cfg['ship_format'] = trim($_POST['ship_format']);

        $cfg['defaults']['max_size_mb'] = intval($_POST['defaults_max_size_mb']);
        $cfg['defaults']['keep'] = intval($_POST['defaults_keep']);
        $cfg['defaults']['compress'] = isset($_POST['defaults_compress']);
        $cfg['defaults']['rotate_on_start'] = isset($_POST['defaults_rotate_on_start']);

        if ($cfg['enabled'] && empty($cfg['endpoint'])) {
            $input_errors[] = 'Endpoint is required when enabled.';
        }

        if (count($input_errors) == 0) {
            save_config_file($config_path, $cfg);
            $savemsg = 'Settings saved.';
            if ($cfg['enabled']) {
                zidlogs_start();
            }
        }
    }
}

$cfg = load_config_file($config_path, $defaults);
$service_enabled = !empty($cfg['enabled']);

$pgtitle = array(gettext('Services'), gettext('ZID Logs'), gettext('Settings'));
include('head.inc');

display_top_tabs(zidlogs_tabs('settings'));
?>

<?php if ($savemsg) { print_info_box($savemsg, 'success'); } ?>
<?php if ($update_msg) { print_info_box(htmlspecialchars($update_msg), 'info'); } ?>
<?php if ($input_errors) { print_input_errors($input_errors); } ?>

<form method="post">
<div class="panel panel-default">
    <div class="panel-heading"><h2 class="panel-title"><?=gettext('Installed Version')?></h2></div>
    <div class="panel-body">
        <code><?=htmlspecialchars(zidlogs_installed_version_line());?></code>
        <div style="clear: both;"></div>
    </div>
</div>

<div class="panel panel-default">
    <div class="panel-heading"><h2 class="panel-title"><?=gettext('Service controls')?></h2></div>
    <div class="panel-body">
        <?php if (zidlogs_status()): ?>
            <span class="label label-success"><?=gettext('Running')?></span>
            &nbsp;
            <button type="submit" name="svc_stop" class="btn btn-sm btn-warning">
                <i class="fa fa-stop"></i> <?=gettext('Stop')?>
            </button>
            <button type="submit" name="svc_restart" class="btn btn-sm btn-primary">
                <i class="fa fa-refresh"></i> <?=gettext('Restart')?>
            </button>
        <?php else: ?>
            <?php if ($service_enabled): ?>
                <span class="label label-danger"><?=gettext('Stopped')?></span>
                &nbsp;
                <button type="submit" name="svc_start" class="btn btn-sm btn-success">
                    <i class="fa fa-play"></i> <?=gettext('Start')?>
                </button>
            <?php else: ?>
                <span class="label label-default"><?=gettext('Disabled')?></span>
            <?php endif; ?>
        <?php endif; ?>
        <button type="submit" name="run_update" class="btn btn-sm btn-default pull-right"
                onclick="return confirm('Run update now?');"><?=gettext('Update')?></button>
        <div style="clear: both;"></div>
    </div>
</div>

<div class="panel panel-default">
    <div class="panel-heading"><h2 class="panel-title"><?=gettext('Settings')?></h2></div>
    <div class="panel-body">
        <div class="form-group">
            <label><?=gettext('Enable')?></label>
            <input type="checkbox" name="enabled" value="yes" <?php if ($cfg['enabled']) echo 'checked'; ?>>
        </div>
        <div class="form-group">
            <label><?=gettext('Endpoint')?></label>
            <input type="text" class="form-control" name="endpoint" value="<?=htmlspecialchars($cfg['endpoint']);?>">
        </div>
        <div class="form-group">
            <label><?=gettext('Auth header name')?></label>
            <input type="text" class="form-control" name="auth_header_name" value="<?=htmlspecialchars($cfg['auth_header_name']);?>">
        </div>
        <div class="form-group">
            <label><?=gettext('Auth token')?></label>
            <input type="text" class="form-control" name="auth_token" value="<?=htmlspecialchars($cfg['auth_token']);?>">
        </div>
        <div class="form-group">
            <label><?=gettext('Rotate interval (s)')?></label>
            <input type="number" class="form-control" name="interval_rotate_seconds" value="<?=intval($cfg['interval_rotate_seconds']);?>">
        </div>
        <div class="form-group">
            <label><?=gettext('Ship interval (s)')?></label>
            <input type="number" class="form-control" name="interval_ship_seconds" value="<?=intval($cfg['interval_ship_seconds']);?>">
        </div>
        <div class="form-group">
            <label><?=gettext('Max bytes per ship')?></label>
            <input type="number" class="form-control" name="max_bytes_per_ship" value="<?=intval($cfg['max_bytes_per_ship']);?>">
        </div>
        <div class="form-group">
            <label><?=gettext('Ship format')?></label>
            <select name="ship_format" class="form-control">
                <option value="lines" <?php if ($cfg['ship_format'] == 'lines') echo 'selected'; ?>>lines</option>
                <option value="raw" <?php if ($cfg['ship_format'] == 'raw') echo 'selected'; ?>>raw</option>
            </select>
        </div>
    </div>
</div>

<div class="panel panel-default">
    <div class="panel-heading"><h2 class="panel-title"><?=gettext('Rotation defaults')?></h2></div>
    <div class="panel-body">
        <div class="form-group">
            <label><?=gettext('Max size (MB)')?></label>
            <input type="number" class="form-control" name="defaults_max_size_mb" value="<?=intval($cfg['defaults']['max_size_mb']);?>">
        </div>
        <div class="form-group">
            <label><?=gettext('Keep')?></label>
            <input type="number" class="form-control" name="defaults_keep" value="<?=intval($cfg['defaults']['keep']);?>">
        </div>
        <div class="form-group">
            <label>
                <input type="checkbox" name="defaults_compress" value="yes" <?php if (!empty($cfg['defaults']['compress'])) echo 'checked'; ?>>
                <?=gettext('Compress')?>
            </label>
        </div>
        <div class="form-group">
            <label>
                <input type="checkbox" name="defaults_rotate_on_start" value="yes" <?php if (!empty($cfg['defaults']['rotate_on_start'])) echo 'checked'; ?>>
                <?=gettext('Rotate on start')?>
            </label>
        </div>
    </div>
</div>

<div class="panel panel-default">
    <div class="panel-body">
        <button type="submit" class="btn btn-primary"><?=gettext('Save')?></button>
    </div>
</div>
</form>

<?php include('foot.inc'); ?>
