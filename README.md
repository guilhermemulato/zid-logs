# zid-logs

Pacote base para pfSense que centraliza rotacao e envio incremental de logs para os demais pacotes ZID.

## Estrutura do repositorio
- cmd/zid-logs/        # Entrypoint do daemon
- internal/            # Modulos internos (config, registry, rotate, shipper, state, status)
- packaging/pfsense/   # Artefatos e scripts do pacote pfSense
- gui/                 # WebGUI do pfSense
- tests/               # Testes automatizados
- scripts/             # Scripts utilitarios (bundle)

## Comandos

```sh
# Build local
make build

# Testes
go test ./...

# Bundle latest
make bundle-latest
```

## Configuracao

Arquivo principal:
- /usr/local/etc/zid-logs/config.json

Registro de inputs:
- /var/db/zid-logs/inputs.d/*.json

Exemplo de input:

```json
{
  "package": "zid-proxy",
  "log_id": "proxy-main",
  "path": "/var/log/zid-proxy.log",
  "policy": {
    "max_size_mb": 50,
    "keep": 10,
    "compress": true,
    "ship_enabled": true
  }
}
```

## Atualizacao

- CLI:
  - `/usr/local/sbin/zid-logs-update`
- WebGUI:
  - Services > ZID Logs > Config > Atualizar

## Observacoes
- Envio incremental por inode + offset.
- Logs rotacionados antigos nao sao reenviados (por enquanto).
