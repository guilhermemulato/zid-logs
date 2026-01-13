<?php
require_once('guiconfig.inc');

$inputs_dir = '/var/db/zid-logs/inputs.d';
$manual_path = $inputs_dir . '/manual.json';

function parse_inputs_file($path) {
    $data = file_get_contents($path);
    $json = json_decode($data, true);
    if (!is_array($json)) {
        return array();
    }
    if (isset($json['inputs']) && is_array($json['inputs'])) {
        return $json['inputs'];
    }
    if (array_keys($json) === range(0, count($json) - 1)) {
        return $json;
    }
    return array($json);
}

function load_all_inputs($dir) {
    $result = array();
    if (!is_dir($dir)) {
        return $result;
    }
    $files = scandir($dir);
    foreach ($files as $file) {
        if (substr($file, -5) !== '.json') {
            continue;
        }
        $path = $dir . '/' . $file;
        $inputs = parse_inputs_file($path);
        foreach ($inputs as $input) {
            $input['_source'] = $file;
            $result[] = $input;
        }
    }
    return $result;
}

function save_manual_inputs($path, $inputs) {
    $dir = dirname($path);
    if (!is_dir($dir)) {
        @mkdir($dir, 0755, true);
    }
    $json = json_encode($inputs, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES);
    file_put_contents($path, $json . "\n");
}

$input_errors = array();
$savemsg = '';

if ($_POST) {
    $manual_inputs = array();
    if (file_exists($manual_path)) {
        $manual_inputs = parse_inputs_file($manual_path);
    }

    if (isset($_POST['add_manual'])) {
        $new_input = array(
            'package' => trim($_POST['package']),
            'log_id' => trim($_POST['log_id']),
            'path' => trim($_POST['path']),
            'policy' => array(
                'max_size_mb' => intval($_POST['max_size_mb']),
                'keep' => intval($_POST['keep']),
                'compress' => isset($_POST['compress']),
                'ship_enabled' => isset($_POST['ship_enabled']),
            ),
        );

        if (empty($new_input['package']) || empty($new_input['log_id']) || empty($new_input['path'])) {
            $input_errors[] = 'Package, Log ID e Path sao obrigatorios.';
        } else {
            $manual_inputs[] = $new_input;
            save_manual_inputs($manual_path, $manual_inputs);
            $savemsg = 'Input manual adicionado.';
        }
    }

    if (isset($_POST['remove_index'])) {
        $idx = intval($_POST['remove_index']);
        if (isset($manual_inputs[$idx])) {
            unset($manual_inputs[$idx]);
            $manual_inputs = array_values($manual_inputs);
            save_manual_inputs($manual_path, $manual_inputs);
            $savemsg = 'Input manual removido.';
        }
    }
}

$inputs = load_all_inputs($inputs_dir);
$manual_inputs = file_exists($manual_path) ? parse_inputs_file($manual_path) : array();

$pgtitle = 'ZID Logs - Inputs';
include('head.inc');
?>

<body>
<?php include('fbegin.inc'); ?>

<?php if ($savemsg) { print_info_box($savemsg, 'success'); } ?>
<?php if ($input_errors) { print_input_errors($input_errors); } ?>

<h2>Inputs registrados</h2>
<table class="table">
    <thead>
        <tr>
            <th>Package</th>
            <th>Log ID</th>
            <th>Path</th>
            <th>Source</th>
        </tr>
    </thead>
    <tbody>
        <?php foreach ($inputs as $row): ?>
        <tr>
            <td><?=htmlspecialchars($row['package']);?></td>
            <td><?=htmlspecialchars($row['log_id']);?></td>
            <td><?=htmlspecialchars($row['path']);?></td>
            <td><?=htmlspecialchars($row['_source']);?></td>
        </tr>
        <?php endforeach; ?>
    </tbody>
</table>

<h2>Inputs manuais</h2>
<table class="table">
    <thead>
        <tr>
            <th>Package</th>
            <th>Log ID</th>
            <th>Path</th>
            <th>Acao</th>
        </tr>
    </thead>
    <tbody>
        <?php foreach ($manual_inputs as $idx => $row): ?>
        <tr>
            <td><?=htmlspecialchars($row['package']);?></td>
            <td><?=htmlspecialchars($row['log_id']);?></td>
            <td><?=htmlspecialchars($row['path']);?></td>
            <td>
                <form method="post" style="display:inline">
                    <input type="hidden" name="remove_index" value="<?=intval($idx);?>">
                    <input type="submit" value="Remover">
                </form>
            </td>
        </tr>
        <?php endforeach; ?>
    </tbody>
</table>

<h2>Adicionar input manual</h2>
<form method="post">
    <table class="formtable">
        <tr>
            <td>Package</td>
            <td><input type="text" name="package"></td>
        </tr>
        <tr>
            <td>Log ID</td>
            <td><input type="text" name="log_id"></td>
        </tr>
        <tr>
            <td>Path</td>
            <td><input type="text" name="path"></td>
        </tr>
        <tr>
            <td>Max size (MB)</td>
            <td><input type="number" name="max_size_mb" value="50"></td>
        </tr>
        <tr>
            <td>Keep</td>
            <td><input type="number" name="keep" value="10"></td>
        </tr>
        <tr>
            <td>Compress</td>
            <td><input type="checkbox" name="compress" checked></td>
        </tr>
        <tr>
            <td>Ship enabled</td>
            <td><input type="checkbox" name="ship_enabled" checked></td>
        </tr>
    </table>
    <div>
        <input type="submit" name="add_manual" value="Adicionar">
    </div>
</form>

<?php include('fend.inc'); ?>
</body>
</html>
