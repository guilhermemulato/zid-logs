# Especificacoes do projeto

## Visao geral
zid-logs e um pacote base para pfSense que centraliza rotacao e envio incremental de logs para os pacotes ZID.

## Status atual
- Estrutura inicial criada (cmd/, internal/, packaging/pfsense/, gui/, tests/).
- Documentacao inicial em README.md.
- Plano detalhado em TODO-FINAL-ZID-LOGS.md.

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
