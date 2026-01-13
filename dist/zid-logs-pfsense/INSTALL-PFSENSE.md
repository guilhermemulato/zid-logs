# ZID Logs - Instalacao pfSense

## Requisitos
- pfSense 2.8.1
- Acesso root via shell

## Instalacao manual
1) Copie o bundle para o pfSense
2) Extraia e rode o instalador:

```sh
# No pfSense
tar -xzf zid-logs-latest.tar.gz
cd zid-logs-pfsense/pkg-zid-logs
sh install.sh
```

## Update
- CLI:

```sh
/usr/local/sbin/zid-logs-update
```

- WebGUI:
  - Services > ZID Logs > Config
  - Clique em "Atualizar"

## Desinstalacao

```sh
cd zid-logs-pfsense/pkg-zid-logs
sh uninstall.sh
```
