<?php
require_once('guiconfig.inc');

$pgtitle = 'ZID Logs - Status';
include('head.inc');

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

<body>
<?php include('fbegin.inc'); ?>

<h2>Status</h2>
<table class="table">
    <thead>
        <tr>
            <th>Package</th>
            <th>Log ID</th>
            <th>Path</th>
            <th>Size</th>
            <th>Backlog</th>
            <th>Last offset</th>
            <th>Last sent</th>
            <th>Last error</th>
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

<form method="post">
    <input type="submit" name="rotate_now" value="Rotate now">
    <input type="submit" name="ship_now" value="Ship now">
</form>

<?php
if ($_POST) {
    if (isset($_POST['rotate_now'])) {
        shell_exec('/usr/local/sbin/zid-logs rotate > /dev/null 2>&1');
        print_info_box('Rotacao executada.', 'success');
    }
    if (isset($_POST['ship_now'])) {
        shell_exec('/usr/local/sbin/zid-logs ship > /dev/null 2>&1');
        print_info_box('Envio executado.', 'success');
    }
}
?>

<?php include('fend.inc'); ?>
</body>
</html>
