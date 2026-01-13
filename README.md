# zid-logs

Pacote base para pfSense que centraliza rotacao e envio incremental de logs para os demais pacotes ZID.

## Estrutura do repositorio
- cmd/zid-logs/        # Entrypoint do daemon
- internal/            # Modulos internos (config, registry, rotate, shipper, state, status)
- packaging/pfsense/   # Artefatos e scripts do pacote pfSense
- gui/                 # WebGUI do pfSense
- tests/               # Testes automatizados

## Comandos
Nao ha comandos de build/test definidos ainda.
