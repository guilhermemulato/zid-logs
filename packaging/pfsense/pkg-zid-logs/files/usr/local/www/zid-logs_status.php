<?php
require_once('guiconfig.inc');
require_once('/usr/local/pkg/zid-logs.inc');

$action_msg = '';
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    if (isset($_POST['rotate_now'])) {
        shell_exec('/usr/local/sbin/zid-logs rotate > /dev/null 2>&1');
        $action_msg = 'Rotation executed.';
    }
    if (isset($_POST['ship_now'])) {
        shell_exec('/usr/local/sbin/zid-logs ship > /dev/null 2>&1');
        $action_msg = 'Ship executed.';
    }
}

$pgtitle = array(gettext('Services'), gettext('ZID Logs'), gettext('Status'));
include('head.inc');

display_top_tabs(zidlogs_tabs('status'));

$status = array('inputs' => array());
$cmd = '/usr/local/sbin/zid-logs status 2>/dev/null';
$output = shell_exec($cmd);
if (!empty($output)) {
    $json = json_decode($output, true);
    if (is_array($json)) {
        $status = $json;
    }
}
?>

<?php if ($action_msg) { print_info_box($action_msg, 'success'); } ?>

<div class="panel panel-default">
    <div class="panel-heading"><h2 class="panel-title"><?=gettext('Status')?></h2></div>
    <div class="panel-body">
        <table class="table table-striped table-hover">
            <thead>
                <tr>
                    <th><?=gettext('Package')?></th>
                    <th><?=gettext('Log ID')?></th>
                    <th><?=gettext('Path')?></th>
                    <th><?=gettext('Size')?></th>
                    <th><?=gettext('Backlog')?></th>
                    <th><?=gettext('Last offset')?></th>
                    <th><?=gettext('Last sent')?></th>
                    <th><?=gettext('Last error')?></th>
                </tr>
            </thead>
            <tbody>
                <?php foreach ($status['inputs'] as $row): ?>
                <tr>
                    <td><?=htmlspecialchars($row['package']);?></td>
                    <td><?=htmlspecialchars($row['log_id']);?></td>
                    <td><?=htmlspecialchars($row['path']);?></td>
                    <td><?=intval($row['file_size']);?></td>
                    <td><?=intval($row['backlog']);?></td>
                    <td><?=intval($row['last_offset']);?></td>
                    <td><?=intval($row['last_sent_at']);?></td>
                    <td><?=htmlspecialchars($row['last_error']);?></td>
                </tr>
                <?php endforeach; ?>
            </tbody>
        </table>
    </div>
</div>

<form method="post">
    <div class="panel panel-default" id="zidlogs-actions">
        <div class="panel-heading"><h2 class="panel-title"><?=gettext('Actions')?></h2></div>
        <div class="panel-body">
            <button type="submit" name="rotate_now" class="btn btn-sm btn-default">
                <i class="fa fa-refresh"></i> <?=gettext('Rotate now')?>
            </button>
            <button type="submit" name="ship_now" class="btn btn-sm btn-default">
                <i class="fa fa-send"></i> <?=gettext('Ship now')?>
            </button>
        </div>
    </div>
</form>

<?php include('foot.inc'); ?>
