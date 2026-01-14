# Especificacoes do projeto

## Visao geral
zid-logs e um pacote base para pfSense que centraliza rotacao e envio incremental de logs para os pacotes ZID.

## Status atual
- Estrutura inicial criada (cmd/, internal/, packaging/pfsense/, gui/, tests/).
- Documentacao inicial em README.md.
- Plano detalhado em TODO-FINAL-ZID-LOGS.md.
- Modulos base criados: config, registry e state (bbolt).
- go.mod inicial com dependencia bbolt.
- Rotacao de logs implementada em internal/rotate.
- Envio incremental implementado em internal/shipper (payload JSON + gzip).
- Daemon e comandos CLI implementados em cmd/zid-logs.
- Status abre o state.db em modo somente leitura com timeout para evitar bloqueios na WebGUI.
- Update remoto roda em background para evitar travamento da WebGUI.
- Documentacao de registro de logs publicada em zid-logs-register.md.
- GUI inicial e rc.d adicionados para integracao pfSense.
- Scripts de install/update/uninstall e bootstrap updater adicionados.
- Testes unitarios iniciais e bundle latest para pfSense adicionados.
- Registro do pacote pfSense via XML/INC e scripts de ativacao/registro.

## Build e binarios
- Sempre gerar binarios para pfSense (FreeBSD/amd64, CGO=0) ao final de cada implementacao:
  - `GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build -o build/zid-logs ./cmd/zid-logs`
  - Regerar bundle: `make bundle-latest`

## Estrutura de modulos (proposta)
- cmd/zid-logs/
- internal/config
- internal/registry
- internal/rotate
- internal/shipper
- internal/state
- internal/status
- packaging/pfsense/
- gui/
