<?php
require_once('guiconfig.inc');

$config_path = '/usr/local/etc/zid-logs/config.json';

$defaults = array(
    'enabled' => false,
    'endpoint' => '',
    'auth_token' => '',
    'device_id' => '',
    'interval_rotate_seconds' => 300,
    'interval_ship_seconds' => 60,
    'max_bytes_per_ship' => 262144,
    'ship_format' => 'lines',
    'tls' => array(
        'insecure_skip_verify' => false,
        'ca_path' => '',
        'client_cert_path' => '',
        'client_key_path' => '',
    ),
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

if ($_POST) {
    $cfg = load_config_file($config_path, $defaults);

    $cfg['enabled'] = isset($_POST['enabled']);
    $cfg['endpoint'] = trim($_POST['endpoint']);
    $cfg['auth_token'] = trim($_POST['auth_token']);
    $cfg['interval_rotate_seconds'] = intval($_POST['interval_rotate_seconds']);
    $cfg['interval_ship_seconds'] = intval($_POST['interval_ship_seconds']);
    $cfg['max_bytes_per_ship'] = intval($_POST['max_bytes_per_ship']);
    $cfg['ship_format'] = trim($_POST['ship_format']);

    $cfg['tls']['insecure_skip_verify'] = isset($_POST['tls_insecure']);
    $cfg['tls']['ca_path'] = trim($_POST['tls_ca_path']);
    $cfg['tls']['client_cert_path'] = trim($_POST['tls_client_cert_path']);
    $cfg['tls']['client_key_path'] = trim($_POST['tls_client_key_path']);

    $cfg['defaults']['max_size_mb'] = intval($_POST['defaults_max_size_mb']);
    $cfg['defaults']['keep'] = intval($_POST['defaults_keep']);
    $cfg['defaults']['compress'] = isset($_POST['defaults_compress']);
    $cfg['defaults']['rotate_on_start'] = isset($_POST['defaults_rotate_on_start']);

    if ($cfg['enabled'] && empty($cfg['endpoint'])) {
        $input_errors[] = 'Endpoint e obrigatorio quando habilitado.';
    }

    if (count($input_errors) == 0) {
        save_config_file($config_path, $cfg);
        $savemsg = 'Configuracao salva.';
    }
}

$cfg = load_config_file($config_path, $defaults);

$pgtitle = 'ZID Logs - Config';
include('head.inc');
?>

<body>
<?php include('fbegin.inc'); ?>

<form method="post">
    <?php if ($savemsg) { print_info_box($savemsg, 'success'); } ?>
    <?php if ($input_errors) { print_input_errors($input_errors); } ?>

    <h2>Configuracao</h2>
    <table class="formtable">
        <tr>
            <td>Habilitar</td>
            <td>
                <input type="checkbox" name="enabled" value="yes" <?php if ($cfg['enabled']) echo 'checked'; ?>>
            </td>
        </tr>
        <tr>
            <td>Endpoint</td>
            <td><input type="text" name="endpoint" size="60" value="<?=htmlspecialchars($cfg['endpoint']);?>"></td>
        </tr>
        <tr>
            <td>Token</td>
            <td><input type="text" name="auth_token" size="60" value="<?=htmlspecialchars($cfg['auth_token']);?>"></td>
        </tr>
        <tr>
            <td>Intervalo rotacao (s)</td>
            <td><input type="number" name="interval_rotate_seconds" value="<?=intval($cfg['interval_rotate_seconds']);?>"></td>
        </tr>
        <tr>
            <td>Intervalo envio (s)</td>
            <td><input type="number" name="interval_ship_seconds" value="<?=intval($cfg['interval_ship_seconds']);?>"></td>
        </tr>
        <tr>
            <td>Max bytes por envio</td>
            <td><input type="number" name="max_bytes_per_ship" value="<?=intval($cfg['max_bytes_per_ship']);?>"></td>
        </tr>
        <tr>
            <td>Formato envio</td>
            <td>
                <select name="ship_format">
                    <option value="lines" <?php if ($cfg['ship_format'] == 'lines') echo 'selected'; ?>>lines</option>
                    <option value="raw" <?php if ($cfg['ship_format'] == 'raw') echo 'selected'; ?>>raw</option>
                </select>
            </td>
        </tr>
    </table>

    <h2>Padroes de rotacao</h2>
    <table class="formtable">
        <tr>
            <td>Max size (MB)</td>
            <td><input type="number" name="defaults_max_size_mb" value="<?=intval($cfg['defaults']['max_size_mb']);?>"></td>
        </tr>
        <tr>
            <td>Keep</td>
            <td><input type="number" name="defaults_keep" value="<?=intval($cfg['defaults']['keep']);?>"></td>
        </tr>
        <tr>
            <td>Compress</td>
            <td><input type="checkbox" name="defaults_compress" value="yes" <?php if (!empty($cfg['defaults']['compress'])) echo 'checked'; ?>></td>
        </tr>
        <tr>
            <td>Rotate on start</td>
            <td><input type="checkbox" name="defaults_rotate_on_start" value="yes" <?php if (!empty($cfg['defaults']['rotate_on_start'])) echo 'checked'; ?>></td>
        </tr>
    </table>

    <h2>TLS</h2>
    <table class="formtable">
        <tr>
            <td>Insecure skip verify</td>
            <td><input type="checkbox" name="tls_insecure" value="yes" <?php if (!empty($cfg['tls']['insecure_skip_verify'])) echo 'checked'; ?>></td>
        </tr>
        <tr>
            <td>CA path</td>
            <td><input type="text" name="tls_ca_path" size="60" value="<?=htmlspecialchars($cfg['tls']['ca_path']);?>"></td>
        </tr>
        <tr>
            <td>Client cert path</td>
            <td><input type="text" name="tls_client_cert_path" size="60" value="<?=htmlspecialchars($cfg['tls']['client_cert_path']);?>"></td>
        </tr>
        <tr>
            <td>Client key path</td>
            <td><input type="text" name="tls_client_key_path" size="60" value="<?=htmlspecialchars($cfg['tls']['client_key_path']);?>"></td>
        </tr>
    </table>

    <div>
        <input type="submit" value="Salvar">
    </div>
</form>

<?php include('fend.inc'); ?>
</body>
</html>
