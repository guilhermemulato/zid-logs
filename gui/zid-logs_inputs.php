<?php
require_once('guiconfig.inc');
require_once('/usr/local/pkg/zid-logs.inc');

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

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $manual_inputs = array();
    if (file_exists($manual_path)) {
        $manual_inputs = parse_inputs_file($manual_path);
    }

    if (isset($_POST['remove_index'])) {
        $idx = intval($_POST['remove_index']);
        if (isset($manual_inputs[$idx])) {
            unset($manual_inputs[$idx]);
            $manual_inputs = array_values($manual_inputs);
            save_manual_inputs($manual_path, $manual_inputs);
            $savemsg = 'Manual input removed.';
        }
    } elseif (isset($_POST['save']) && isset($_POST['zidlogs_inputs_form']) && $_POST['zidlogs_inputs_form'] === '1') {
        $new_input = array(
            'package' => trim((string)($_POST['package'] ?? '')),
            'log_id' => trim((string)($_POST['log_id'] ?? '')),
            'path' => trim((string)($_POST['path'] ?? '')),
            'policy' => array(
                'max_size_mb' => intval($_POST['max_size_mb'] ?? 0),
                'keep' => intval($_POST['keep'] ?? 0),
                'compress' => isset($_POST['compress']),
                'ship_enabled' => isset($_POST['ship_enabled']),
            ),
        );

        if ($new_input['package'] === '' || $new_input['log_id'] === '' || $new_input['path'] === '') {
            $input_errors[] = 'Package, Log ID and Path are required.';
        } else {
            $manual_inputs[] = $new_input;
            save_manual_inputs($manual_path, $manual_inputs);
            $savemsg = 'Manual input added.';
        }
    }
}

$inputs = load_all_inputs($inputs_dir);
$manual_inputs = file_exists($manual_path) ? parse_inputs_file($manual_path) : array();

$pconfig = array(
    'package' => '',
    'log_id' => '',
    'path' => '',
    'max_size_mb' => '50',
    'keep' => '10',
    'compress' => true,
    'ship_enabled' => true,
);

$pgtitle = array(gettext('Services'), gettext('ZID Logs'), gettext('Inputs'));
include('head.inc');

display_top_tabs(zidlogs_tabs('inputs'));
?>

<?php if ($savemsg) { print_info_box($savemsg, 'success'); } ?>
<?php if ($input_errors) { print_input_errors($input_errors); } ?>

<div class="panel panel-default">
    <div class="panel-heading"><h2 class="panel-title"><?=gettext('Registered inputs')?></h2></div>
    <div class="panel-body">
        <table class="table table-striped table-hover">
            <thead>
                <tr>
                    <th><?=gettext('Package')?></th>
                    <th><?=gettext('Log ID')?></th>
                    <th><?=gettext('Path')?></th>
                    <th><?=gettext('Source')?></th>
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
    </div>
</div>

<div class="panel panel-default">
    <div class="panel-heading"><h2 class="panel-title"><?=gettext('Manual inputs')?></h2></div>
    <div class="panel-body">
        <table class="table table-striped table-hover">
            <thead>
                <tr>
                    <th><?=gettext('Package')?></th>
                    <th><?=gettext('Log ID')?></th>
                    <th><?=gettext('Path')?></th>
                    <th><?=gettext('Action')?></th>
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
                            <button type="submit" class="btn btn-xs btn-danger">
                                <i class="fa fa-trash"></i> <?=gettext('Remove')?>
                            </button>
                        </form>
                    </td>
                </tr>
                <?php endforeach; ?>
            </tbody>
        </table>
    </div>
</div>

<?php
$form = new Form();
$form->addGlobal(new Form_Input(
    'zidlogs_inputs_form',
    '',
    'hidden',
    '1'
));

$section = new Form_Section(gettext('Add manual input'));
$section->addInput(new Form_Input(
    'package',
    gettext('Package'),
    'text',
    $pconfig['package']
));
$section->addInput(new Form_Input(
    'log_id',
    gettext('Log ID'),
    'text',
    $pconfig['log_id']
));
$section->addInput(new Form_Input(
    'path',
    gettext('Path'),
    'text',
    $pconfig['path']
));
$section->addInput(new Form_Input(
    'max_size_mb',
    gettext('Max size (MB)'),
    'number',
    $pconfig['max_size_mb']
));
$section->addInput(new Form_Input(
    'keep',
    gettext('Keep'),
    'number',
    $pconfig['keep']
));
$section->addInput(new Form_Checkbox(
    'compress',
    gettext('Compress'),
    gettext('Compress rotated files'),
    !empty($pconfig['compress'])
));
$section->addInput(new Form_Checkbox(
    'ship_enabled',
    gettext('Ship enabled'),
    gettext('Allow shipping for this input'),
    !empty($pconfig['ship_enabled'])
));
$form->add($section);

print($form);

include('foot.inc');
?>
