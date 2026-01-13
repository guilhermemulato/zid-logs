Você é um engenheiro senior (FreeBSD/pfSense + Go) e vai criar um NOVO pacote para pfSense chamado **zid-logs**. Ele será um pacote “base” reutilizável por TODOS os outros pacotes ZID (zid-proxy, zid-geolocation, futuros), centralizando: (1) rotação de logs e (2) envio incremental desses logs para um servidor remoto, com checkpoint por arquivo para continuar do ponto exato do último envio.

OBJETIVO
- Implementar um pacote pfSense (FreeBSD) chamado **zid-logs**.
- Ele deve instalar um daemon em Go e uma GUI simples no pfSense (WebGUI) chamada “ZID Logs”.
- Outros pacotes ZID NÃO vão mais ter seu próprio *_rotate; em vez disso, vão “registrar” seus logs em um registry local (inputs.d). O zid-logs descobre esses logs e aplica rotação + envio.

RESTRIÇÕES / PADRÕES
- Linguagem do daemon: Go (CGO desabilitado, binário estático quando possível).
- Compatível com pfSense 2.8.1 (FreeBSD).
- Arquivos e paths devem seguir padrões pfSense (/usr/local/..., /var/db/...).
- Sem dependência de “logrotate” externo: implemente rotação no próprio Go (size/time/retention + compressão gzip).
- Envio incremental: continuar sempre de onde parou usando checkpoint persistente por arquivo (inode+offset).
- O daemon deve ser controlável por rc.d (start/stop/status/restart) e habilitado/desabilitado via GUI.
- Segurança: envio HTTPS, token de autenticação por header.


ARQUITETURA (ALTA)
1) Daemon Go: /usr/local/sbin/zid-logs
   - Subcomandos:
     - `zid-logs run` (daemon)
     - `zid-logs rotate` (executa rotação uma vez)
     - `zid-logs ship` (executa envio uma vez)
     - `zid-logs status` (mostra status JSON para GUI)
     - `zid-logs validate` (valida configs/inputs)
   - Ele deve:
     - Ler config global do zid-logs
     - Descobrir inputs (logs registrados por pacotes) em inputs.d
     - Aplicar rotação por arquivo (políticas configuráveis)
     - Enviar incrementalmente os novos dados de cada arquivo para servidor
     - Persistir “state” (checkpoint) em banco local (recomendado: bbolt)
     - Expor métricas básicas no status (último envio/erro/backlog por input)

2) Registry de inputs (para outros pacotes):
   - Diretório padrão: /var/db/zid-logs/inputs.d/
   - Cada pacote ZID cria um JSON do tipo:
     - /var/db/zid-logs/inputs.d/zid-proxy.json
     - /var/db/zid-logs/inputs.d/zid-geolocation.json
   - Esse JSON declara logs (path + política rotate + ship enable).
   - O zid-logs deve ler TODOS os *.json e juntar numa lista.

3) Config global do zid-logs:
   - Arquivo: /usr/local/etc/zid-logs/config.json
   - Contém:
     - enabled (bool)
     - ship endpoint base URL (ex: https://logs.soulsolucoes.com.br/api/v1/ingest)
     - auth token (string)
     - device_id (string) (se vazio: gerar e persistir em /var/db/zid-logs/device_id)
     - interval_rotate_seconds
     - interval_ship_seconds
     - max_bytes_per_ship (chunk)
     - tls options (insecure_skip_verify false por padrão; paths de CA/client cert opcional)
     - defaults de rotação (se input não definir):
       - max_size_mb, keep, compress, rotate_on_start (bool)
     - opção “ship_format”: “lines” (default) e “raw” (para futuro)

4) State persistente:
   - Banco: /var/db/zid-logs/state.db (bbolt)
   - Deve armazenar:
     - por input (package + log_id + path) e por inode:
       - last_offset (bytes)
       - last_sent_at (unix ts)
       - last_error (string)
       - file_identity (dev+inode)
     - estatísticas agregadas opcionais (bytes_sent_total etc)

5) GUI pfSense (“ZID Logs”):
   - Menu: Services > ZID Logs (ou um menu “ZID” se existir; use Services se mais simples)
   - Páginas mínimas:
     - Status: lista inputs descobertos, último envio, offset, erros, tamanho atual do arquivo, backlog aproximado.
     - Config: enable/disable, endpoint URL, token, intervalos, TLS settings (simples), botão “Salvar” e “Aplicar”.
     - Inputs: mostrar registry atual (inputs.d) e permitir:
       - adicionar input manual (um log path)
       - editar política rotate/ship de um input manual
       - NÃO editar inputs “gerenciados por pacotes” (mostrar como read-only, com origem “zid-proxy.json” etc)
     - Ações: botões “Rotate now” e “Ship now” (chamam CLI)
   - Persistência da GUI deve escrever em /usr/local/etc/zid-logs/config.json e /var/db/zid-logs/inputs.d/manual.json (somente inputs manuais).
   - Usar o padrão pfSense: conf em config.xml via PHP? (Pode ser simples: JSON files + hooks; mas preferir padrão pfSense de configuração se der. Se ficar pesado, usar JSON é aceitável, mas documentar bem.)

DETALHES IMPORTANTES (ENVIO INCREMENTAL)
- O envio deve ser incremental por arquivo:
  - Abrir arquivo, seek para last_offset, ler até max_bytes_per_ship ou EOF.
  - Enviar chunk e, só após sucesso HTTP 200, atualizar offset no state.
- Rotação e envio precisam lidar com inode mudando:
  - Guardar dev+inode no state.
  - Ao iniciar um ship:
    - stat do arquivo atual -> inode atual.
    - se inode mudou e offset antigo > tamanho atual, reset offset = 0.
    - se inode mudou mas arquivo antigo pode existir (ex: .1, .gz): por enquanto ignore backlog de rotacionados (documente). (Opcional: enviar backlog de .1 antes de resetar, se implementável rapidamente).
- Para marcar “timestamp”, não dependa de timestamp do log; use offset mesmo. Timestamp é “last_sent_at” para status.
- Payload do envio:
  - POST /ingest
  - Headers: x-auth-n8n: <token>, Content-Encoding: gzip (se gzip), Content-Type: application/json
  - Body JSON:
    {
      "device_id": "...",
      "pf_hostname": "...",
      "package": "zid-proxy",
      "log_id": "proxy-main",
      "path": "/var/log/zid-proxy.log",
      "inode": 12345,
      "offset_start": 1000,
      "offset_end": 2000,
      "sent_at": 1700000000,
      "lines": ["...","..."]   // se ship_format=lines
    }
  - Compactar request com gzip (opcional, on por padrão).

DETALHES IMPORTANTES (ROTAÇÃO)
- Política por input:
  - max_size_mb (default 50)
  - keep (default 10)
  - compress (default true)
  - max_age_days (opcional)
- Algoritmo:
  - Se file size >= max_size -> rotate
  - rotate: renomear file -> file.1, file.1 -> file.2 ... até keep
  - criar novo arquivo vazio com mesmas permissões (e ownership se possível)
  - se compress: gzip de file.N conforme política (por ex: gzip file.2..keep ou gzip todos exceto .1)
- Garantir atomicidade razoável:
  - usar rename (atomic no mesmo filesystem)
  - abrir novo file com O_CREATE|O_TRUNC
- Importante: evitar “perder” logs durante rotação:
  - ideal: fechar e reabrir no produtor; mas como você não controla o produtor, só faça rename e recrie.
  - Documentar: produtores devem abrir com append.

INTEGRAÇÃO COM OUTROS PACOTES (MIGRAÇÃO)
- Criar documentação e exemplo para zid-proxy e zid-geolocation:
  - Em seus scripts de install/upgrade, criar:
    /var/db/zid-logs/inputs.d/zid-proxy.json
    /var/db/zid-logs/inputs.d/zid-geolocation.json
  - Remover/aposentar zid-proxy_rotate (fazer nota no README).
- Porém neste PR você só cria o zid-logs; exemplos podem ser docs + um “sample json” incluído.

ESTRUTURA DO REPOSITÓRIO (SUGESTÃO)
- cmd/zid-logs/main.go
- internal/config (ler/escrever config global + inputs)
- internal/registry (load inputs.d)
- internal/rotate (rotator)
- internal/shipper (ship logic + http client + gzip)
- internal/state (bbolt)
- internal/status (agregação)
- packaging/pfsense/
  - rc.d script: /usr/local/etc/rc.d/zid_logs
  - pkg plist/manifests conforme seu padrão de build
- gui/
  - pfSense PHP pages em /usr/local/www/ (seguir padrão dos seus pacotes)

RC.D (OBRIGATÓRIO)
- Criar script /usr/local/etc/rc.d/zid_logs com:
  - zid_logs_enable="YES/NO" (integrado com config)
  - comando para start: /usr/local/sbin/zid-logs run
  - pidfile em /var/run/zid-logs.pid
  - logs do serviço em /var/log/zid-logs-daemon.log (opcional)

COMPORTAMENTO DO DAEMON
- Ao rodar `run`:
  - carregar config
  - se disabled: sair com exit 0 (ou ficar em loop dormindo; escolher uma estratégia estável)
  - a cada interval_rotate_seconds: executar rotate em todos inputs
  - a cada interval_ship_seconds: executar ship incremental
  - lidar com reload: SIGHUP recarrega config e inputs
  - graceful shutdown: SIGTERM encerra loops
- Implementar locking para não rodar rotate+ship simultâneo no mesmo arquivo.

ERROS / LOGS
- Logar no syslog ou arquivo (simplifique: arquivo em /var/log/zid-logs.log)
- No status, manter last_error por input e um last_error_global.

TESTES
- Adicionar testes unitários principais:
  - parse config/inputs
  - rotate naming/keep
  - state offset update
  - inode change detection
- Adicionar um “modo simulação” via env (ex: ZID_LOGS_DRY_RUN=1) para não enviar de verdade.

ENTREGÁVEIS
1) Código Go completo + módulos internos.
2) rc.d script funcional.
3) GUI pfSense com as páginas descritas (mínimo viável, mas usável).
4) Documentação:
   - README: como instalar, configurar, registrar logs via inputs.d, como validar.
   - Exemplo de input JSON para zid-proxy e zid-geolocation.
   - Explicar checkpoint por inode+offset e limitações.

CRITÉRIOS DE ACEITE
- Instala no pfSense e inicia via rc.d.
- GUI consegue:
  - habilitar/desabilitar
  - salvar endpoint/token/intervalos
  - listar inputs descobertos
  - executar “rotate now” e “ship now”
- Daemon:
  - rotaciona log quando excede size
  - envia incrementalmente e continua do offset correto após restart
  - não reenvia dados já enviados
  - tolera arquivo inexistente sem crash
  - tolera inode change (reset ou comportamento documentado)
- Tudo pronto para ser reutilizado pelos próximos pacotes ZID.

AGORA IMPLEMENTE.
- Não invente bibliotecas complexas: use stdlib + bbolt.
- Priorize robustez e clareza.
