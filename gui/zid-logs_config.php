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
    'rotate_at' => '00:00',
    'ship_interval_hours' => 1,
    'interval_rotate_seconds' => 0,
    'interval_ship_seconds' => 0,
    'max_bytes_per_ship' => 5242880,
    'ship_format' => 'lines',
    'defaults' => array(
        'max_size_mb' => 50,
        'keep' => 10,
        'compress' => true,
        'rotate_on_start' => false,
    ),
);

$mask_char = "\xE2\x97\x8F";
$masked_value = str_repeat($mask_char, 8);

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

function zidlogs_bytes_to_mb($bytes) {
    $bytes = floatval($bytes);
    if ($bytes <= 0) {
        return 0;
    }
    return $bytes / 1024 / 1024;
}

function zidlogs_mb_to_bytes($mb) {
    $mb = floatval($mb);
    if ($mb <= 0) {
        return 0;
    }
    return (int)round($mb * 1024 * 1024);
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
$update_output = '';
$update_status = '';
$run_update = false;

function zidlogs_pkg_version_from_config() {
    global $config;
    if (!isset($config['installedpackages']['package']) || !is_array($config['installedpackages']['package'])) {
        return '';
    }
    foreach ($config['installedpackages']['package'] as $pkg) {
        if (!is_array($pkg)) {
            continue;
        }
        $name = (string)($pkg['name'] ?? '');
        if ($name === 'zid-logs' || $name === 'zidlogs') {
            return trim((string)($pkg['version'] ?? ''));
        }
    }
    return '';
}

function zidlogs_pkg_version_from_xml() {
    $xml_path = '/usr/local/pkg/zid-logs.xml';
    $raw = @file_get_contents($xml_path);
    if ($raw === false || $raw === '') {
        return '';
    }
    if (preg_match('/<version>([^<]+)<\\/version>/', $raw, $matches)) {
        return trim($matches[1]);
    }
    return '';
}

function zidlogs_installed_version_line() {
    $cfg_version = zidlogs_pkg_version_from_config();
    if ($cfg_version !== '') {
        return $cfg_version;
    }

    $xml_version = zidlogs_pkg_version_from_xml();
    if ($xml_version !== '') {
        return $xml_version;
    }

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
    return 'Unknown';
}

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
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
        $run_update = true;
    } elseif (isset($_POST['save']) && isset($_POST['zidlogs_settings_form']) && $_POST['zidlogs_settings_form'] === '1') {
        $cfg = load_config_file($config_path, $defaults);

        $cfg['enabled'] = isset($_POST['enabled']);
        $cfg['endpoint'] = trim((string)($_POST['endpoint'] ?? ''));
        $auth_token = trim((string)($_POST['auth_token'] ?? ''));
        if ($auth_token !== '' && $auth_token !== $masked_value) {
            $cfg['auth_token'] = $auth_token;
        }
        $auth_header_name = trim((string)($_POST['auth_header_name'] ?? ''));
        if ($auth_header_name !== '' && $auth_header_name !== $masked_value) {
            $cfg['auth_header_name'] = $auth_header_name;
        }
        $cfg['rotate_at'] = trim((string)($_POST['rotate_at'] ?? ''));
        $cfg['ship_interval_hours'] = intval($_POST['ship_interval_hours'] ?? 0);
        $cfg['interval_rotate_seconds'] = 0;
        $cfg['interval_ship_seconds'] = 0;
        $cfg['max_bytes_per_ship'] = zidlogs_mb_to_bytes($_POST['max_bytes_per_ship'] ?? 0);
        $cfg['ship_format'] = trim((string)($_POST['ship_format'] ?? 'lines'));

        $cfg['defaults']['max_size_mb'] = intval($_POST['defaults_max_size_mb'] ?? 0);
        $cfg['defaults']['keep'] = intval($_POST['defaults_keep'] ?? 0);
        $cfg['defaults']['compress'] = isset($_POST['defaults_compress']);
        $cfg['defaults']['rotate_on_start'] = isset($_POST['defaults_rotate_on_start']);

        if ($cfg['rotate_at'] === '') {
            $input_errors[] = 'Rotate time is required.';
        } elseif (!preg_match('/^([01]?[0-9]|2[0-3])(:[0-5][0-9])?$/', $cfg['rotate_at'])) {
            $input_errors[] = 'Rotate time must be in HH or HH:MM format.';
        }

        if ($cfg['ship_interval_hours'] <= 0) {
            $input_errors[] = 'Ship interval (hours) must be greater than zero.';
        }

        if ($cfg['enabled'] && $cfg['endpoint'] === '') {
            $input_errors[] = 'Endpoint is required when enabled.';
        }

        if (count($input_errors) == 0) {
            save_config_file($config_path, $cfg);
            zidlogs_set_rc_enable($cfg['enabled']);
            $savemsg = 'Settings saved.';
            if ($cfg['enabled']) {
                if (!zidlogs_status()) {
                    zidlogs_start();
                } else {
                    zidlogs_reload();
                }
            }
        }
    }
}

$cfg = load_config_file($config_path, $defaults);
$service_enabled = !empty($cfg['enabled']);

$pconfig = array(
    'enabled' => !empty($cfg['enabled']),
    'endpoint' => $cfg['endpoint'],
    'auth_token' => $cfg['auth_token'] !== '' ? $masked_value : '',
    'auth_header_name' => $cfg['auth_header_name'] !== '' ? $masked_value : '',
    'rotate_at' => $cfg['rotate_at'],
    'ship_interval_hours' => $cfg['ship_interval_hours'],
    'max_bytes_per_ship' => zidlogs_bytes_to_mb($cfg['max_bytes_per_ship']),
    'ship_format' => $cfg['ship_format'],
    'defaults_max_size_mb' => $cfg['defaults']['max_size_mb'],
    'defaults_keep' => $cfg['defaults']['keep'],
    'defaults_compress' => !empty($cfg['defaults']['compress']),
    'defaults_rotate_on_start' => !empty($cfg['defaults']['rotate_on_start']),
);

$pgtitle = array(gettext('Services'), gettext('ZID Logs'), gettext('Settings'));
include('head.inc');

display_top_tabs(zidlogs_tabs('settings'));
?>

<?php if ($savemsg) { print_info_box($savemsg, 'success'); } ?>
<?php if ($update_msg) { print_info_box(htmlspecialchars($update_msg), $update_status ?: 'info'); } ?>
<?php if ($update_output) { print_info_box('<pre>' . htmlspecialchars($update_output) . '</pre>', 'info'); } ?>
<?php if ($input_errors) { print_input_errors($input_errors); } ?>
<?php
if ($run_update) {
    set_time_limit(0);
    ignore_user_abort(true);
    while (ob_get_level() > 0) {
        ob_end_flush();
    }
    ob_implicit_flush(true);

    echo '<div class="panel panel-default">';
    echo '<div class="panel-heading"><h2 class="panel-title">' . gettext('Update output') . '</h2></div>';
    echo '<div class="panel-body"><pre>';

    $cmd = 'env ZID_LOGS_SKIP_WEBGUI_RELOAD=1 /bin/sh /usr/local/sbin/zid-logs-update 2>&1';
    $descriptors = array(
        1 => array('pipe', 'w'),
    );
    $process = proc_open($cmd, $descriptors, $pipes);
    $rc = 1;
    if (is_resource($process)) {
        while (!feof($pipes[1])) {
            $line = fgets($pipes[1]);
            if ($line === false) {
                break;
            }
            echo htmlspecialchars($line);
            flush();
        }
        fclose($pipes[1]);
        $rc = proc_close($process);
    } else {
        echo htmlspecialchars(gettext('Failed to start update.'));
    }

    echo '</pre></div></div>';

    if ($rc === 0) {
        print_info_box(gettext('Done.'), 'success');
    } else {
        print_info_box(gettext('Update failed.'), 'danger');
    }
}
?>

<div class="panel panel-default">
    <div class="panel-heading"><h2 class="panel-title"><?=gettext('Installed Version')?></h2></div>
    <div class="panel-body">
        <code><?=htmlspecialchars(zidlogs_installed_version_line());?></code>
    </div>
</div>

<div class="panel panel-default" id="zidlogs-service-controls">
    <div class="panel-heading"><h2 class="panel-title"><?=gettext('Service Controls')?></h2></div>
    <div class="panel-body">
        <form method="post">
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
                    &nbsp;
                    <button type="button" class="btn btn-sm btn-success" disabled="disabled">
                        <i class="fa fa-play"></i> <?=gettext('Start')?>
                    </button>
                    <span class="text-muted" style="margin-left:8px;">
                        <?=gettext('Ative o servico nas configuracoes para iniciar.')?>
                    </span>
                <?php endif; ?>
            <?php endif; ?>

            <button type="submit" name="run_update" class="btn btn-sm btn-default pull-right"
                    onclick="return confirm('<?=gettext("Run update now?")?>');">
                <i class="fa fa-download"></i> <?=gettext('Update')?>
            </button>
        </form>
    </div>
</div>

<?php
$form = new Form();
$form->addGlobal(new Form_Input(
    'zidlogs_settings_form',
    '',
    'hidden',
    '1'
));

$section = new Form_Section(gettext('Settings'));
$section->addInput(new Form_Checkbox(
    'enabled',
    gettext('Enable'),
    gettext('Enable ZID Logs service'),
    !empty($pconfig['enabled'])
));
$section->addInput(new Form_Input(
    'endpoint',
    gettext('Endpoint'),
    'text',
    $pconfig['endpoint']
));
$section->addInput(new Form_Input(
    'auth_header_name',
    gettext('Auth header name'),
    'password',
    $pconfig['auth_header_name']
));
$section->addInput(new Form_Input(
    'auth_token',
    gettext('Auth token'),
    'password',
    $pconfig['auth_token']
));
$section->addInput(new Form_Input(
	'rotate_at',
	gettext('Rotate time (HH:MM)'),
	'text',
	$pconfig['rotate_at']
))->setHelp(gettext('Rotation runs once per day at the specified time, regardless of size.'));
$section->addInput(new Form_Input(
	'ship_interval_hours',
	gettext('Ship interval (hours)'),
	'number',
	$pconfig['ship_interval_hours']
))->setHelp(gettext('Send logs every N hours.'));
$section->addInput(new Form_Input(
    'max_bytes_per_ship',
    gettext('Max MB per ship'),
    'number',
    $pconfig['max_bytes_per_ship']
))->setHelp(gettext('Maximum payload size per ship in MB.'));
$section->addInput(new Form_Select(
    'ship_format',
    gettext('Ship format'),
    $pconfig['ship_format'],
    array(
        'lines' => gettext('lines'),
        'raw' => gettext('raw'),
    )
));
$form->add($section);

$section = new Form_Section(gettext('Rotation defaults'));
$section->addInput(new Form_Input(
    'defaults_max_size_mb',
    gettext('Max size (MB)'),
    'number',
    $pconfig['defaults_max_size_mb']
));
$section->addInput(new Form_Input(
    'defaults_keep',
    gettext('Keep'),
    'number',
    $pconfig['defaults_keep']
));
$section->addInput(new Form_Checkbox(
    'defaults_compress',
    gettext('Compress'),
    gettext('Compress rotated files'),
    !empty($pconfig['defaults_compress'])
));
$section->addInput(new Form_Checkbox(
    'defaults_rotate_on_start',
    gettext('Rotate on start'),
    gettext('Rotate logs when service starts'),
    !empty($pconfig['defaults_rotate_on_start'])
));
$form->add($section);

print($form);

include('foot.inc');
?>
