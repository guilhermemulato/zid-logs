Aqui vai um plano de execucao detalhado, dividido em fases, com exemplos de codigo e decisoes tecnicas alinhadas ao TODO. Nao vou implementar nada agora; e o plano para aprovacao.

Fase 0 — Alinhamento e preparo do repositorio (estrutura e documentacao minima)
- Objetivo: base do projeto e documentacao inicial.
- Entregas:
  - Estrutura: cmd/, internal/, packaging/pfsense/, gui/, tests/ (se aplicavel).
  - Atualizar README.md e AGENTS.md com layout real e comandos.

Fase 1 — Core Go: config, registry e state (bbolt)
- Objetivo: config global, inputs.d, device_id e checkpoint persistente.
- Entregas:
  - internal/config: /usr/local/etc/zid-logs/config.json.
  - internal/registry: /var/db/zid-logs/inputs.d/*.json.
  - internal/state: /var/db/zid-logs/state.db (bbolt).

Fase 2 — Rotacao de logs (sem logrotate externo)
- Objetivo: rotacao size-based + keep + gzip.
- Entregas:
  - internal/rotate com algoritmo de rename e recriacao.

Fase 3 — Envio incremental com checkpoint por inode+offset
- Objetivo: envio incremental confiavel e retomavel.
- Entregas:
  - internal/shipper com POST JSON + gzip opcional + update de offset apenas com HTTP 200.

Fase 4 — Daemon e comandos CLI
- Objetivo: run/rotate/ship/status/validate.
- Entregas:
  - Loops de rotate e ship, SIGHUP para recarregar, SIGTERM para sair.

Fase 5 — Integracao pfSense (rc.d e GUI)
- Objetivo: controle do servico e WebGUI funcional.
- Entregas:
  - packaging/pfsense/zid_logs (rc.d).
  - gui/zid-logs_{status,config,inputs}.php.
  - Botoes “Rotate now” e “Ship now”.

Fase 6 — Instaladores do pacote (install/update/uninstall)
- Objetivo: criar scripts no padrao zid-proxy e zid-geolocation, com update automatico via S3.
- Entregas:
  - packaging/pfsense/pkg-zid-logs/install.sh
  - packaging/pfsense/pkg-zid-logs/update.sh
  - packaging/pfsense/pkg-zid-logs/update-bootstrap.sh
  - packaging/pfsense/pkg-zid-logs/uninstall.sh
- Padroes e comportamento:
  - update.sh baixa https://s3.soulsolucoes.com.br/soul/portal/zid-logs-latest.tar.gz (override com -u ou env ZID_LOGS_UPDATE_URL), extrai em /tmp, encontra o install.sh dentro do bundle e executa.
  - Controle de versao opcional via .version (igual ao zid-proxy), e -f para forcar update.
  - Interrompe processos antes de atualizar (kill direto, evitando rc.d bloqueante).
  - install.sh copia arquivos para /usr/local/..., instala rc.d, GUI, binario, arquivos de pkg, e cria logs/configs iniciais.
  - update-bootstrap.sh vira /usr/local/sbin/zid-logs-update, permitindo atualizacao sem reenvio manual do bundle.
  - uninstall.sh remove binario, rc.d, GUI, pkg info; pergunta antes de remover config/state/logs.
- Exemplo de update.sh (adaptado do padrao existente):
  URL_DEFAULT="https://s3.soulsolucoes.com.br/soul/portal/zid-logs-latest.tar.gz"
  URL="${ZID_LOGS_UPDATE_URL:-$URL_DEFAULT}"

  # ... validacoes, downloader (fetch/curl), tmp dir ...

  tar -xzf "${TARBALL}" -C "${EXTRACT_DIR}"
  INSTALL_SH="$(find "${EXTRACT_DIR}" -maxdepth 5 -type f -path "*/pkg-zid-logs/install.sh" | head -n 1)"
  sh "${INSTALL_SH}"

Fase 7 — Testes + modo dry-run + documentacao
- Objetivo: robustez e orientacao para uso/integracao.
- Entregas:
  - Tests unitarios (config, rotate, state, inode).
  - ZID_LOGS_DRY_RUN=1.
  - README com instalacao, inputs.d, limitacoes de inode/backlog.

Fase 8 — Ajustes finais e validacao no pfSense
- Objetivo: garantir criterios de aceite.

Quer que eu prossiga com esse plano atualizado?
Responda com:
1) Aprovar e iniciar Fase 0
2) Ajustar o plano (me diga quais pontos)
