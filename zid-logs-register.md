# Registro de logs no ZID Logs

Este documento descreve o passo a passo para que outros pacotes registrem seus logs no ZID Logs.

## Visao geral
O ZID Logs descobre logs por meio de arquivos JSON em `/var/db/zid-logs/inputs.d`. Cada arquivo pode conter um ou mais registros de log. O daemon recarrega esses arquivos periodicamente (no loop de rotacao/envio), entao alteracoes passam a valer sem reiniciar o servico.

## 1) Escolha do nome do arquivo
Crie um arquivo JSON em `/var/db/zid-logs/inputs.d` com um nome exclusivo do seu pacote.

Recomendacao:
- `/var/db/zid-logs/inputs.d/<seu-pacote>.json`

Exemplo:
- `/var/db/zid-logs/inputs.d/zid-firewall.json`

## 2) Formatos aceitos
Voce pode usar qualquer um dos formatos abaixo no arquivo JSON:

1) Array de objetos (recomendado):
```json
[
  {
    "package": "zid-firewall",
    "log_id": "main",
    "path": "/var/log/zid-firewall.log"
  }
]
```

2) Objeto unico:
```json
{
  "package": "zid-firewall",
  "log_id": "main",
  "path": "/var/log/zid-firewall.log"
}
```

3) Objeto com chave `inputs`:
```json
{
  "inputs": [
    {
      "package": "zid-firewall",
      "log_id": "main",
      "path": "/var/log/zid-firewall.log"
    }
  ]
}
```

## 3) Campos obrigatorios
Cada registro precisa conter:
- `package` (string): identificador do pacote (ex.: `zid-firewall`).
- `log_id` (string): identificador do log dentro do pacote (ex.: `main`, `events`, `alerts`).
- `path` (string): caminho absoluto do arquivo de log.

Exemplo minimo:
```json
{
  "package": "zid-firewall",
  "log_id": "main",
  "path": "/var/log/zid-firewall.log"
}
```

## 4) Politicas opcionais por log (`policy`)
Voce pode definir politicas especificas por log. Se nao definir, os defaults do ZID Logs serao usados.

Campos disponiveis:
- `max_size_mb` (int): tamanho maximo em MB antes de rotacionar.
- `keep` (int): quantidade de arquivos rotacionados para manter.
- `compress` (bool): se verdadeiro, comprime arquivos rotacionados (a partir do .2).
- `max_age_days` (int): rotaciona se o arquivo tiver idade maior que este valor.
- `ship_enabled` (bool): se falso, este log nao sera enviado.

Exemplo com policy:
```json
{
  "package": "zid-firewall",
  "log_id": "alerts",
  "path": "/var/log/zid-firewall-alerts.log",
  "policy": {
    "max_size_mb": 100,
    "keep": 7,
    "compress": true,
    "max_age_days": 10,
    "ship_enabled": true
  }
}
```

## 5) Timestamp layout (rotacao inteligente)
Para rotacao diaria no horario configurado, mesmo quando o daemon estava parado, informe o layout de timestamp do log.

Campo:
- `timestamp_layout` (string): layout do timestamp no formato do Go.

Exemplos:
- `2006-01-02T15:04:05-07:00` (ex.: `2026-01-13T23:45:20-03:00`)
- `02/01/2006 15:04:05` (ex.: `14/01/2026 10:51:37`)

Exemplo completo:
```json
{
  "package": "zid-proxy",
  "log_id": "proxy-main",
  "path": "/var/log/zid-proxy.log",
  "timestamp_layout": "2006-01-02T15:04:05-07:00",
  "policy": {
    "max_size_mb": 50,
    "keep": 10,
    "compress": true
  }
}
```

## 6) Post-rotate notify (SIGHUP ou comando)
Apos a rotacao, alguns produtores precisam reabrir o arquivo. Para isso, configure um sinal ou comando.

Campos:
- `post_rotate_signal` (string): nome do sinal (ex.: `HUP`). Default: `HUP`.
- `post_rotate_pidfile` (string): caminho do pidfile para enviar sinal.
- `post_rotate_match` (string): padrao para `pgrep -f`.
- `post_rotate_command` (string): comando a executar apos rotacao.

Regra de prioridade:
1) `post_rotate_command`
2) `post_rotate_pidfile` + `post_rotate_signal`
3) `post_rotate_match` + `post_rotate_signal`

Exemplos:
```json
{
  "package": "zid-proxy",
  "log_id": "proxy-main",
  "path": "/var/log/zid-proxy.log",
  "timestamp_layout": "2006-01-02T15:04:05-07:00",
  "post_rotate_signal": "HUP",
  "post_rotate_pidfile": "/var/run/zid-proxy.pid"
}
```

```json
{
  "package": "zid-geolocation",
  "log_id": "geo-main",
  "path": "/var/log/zid-geolocation.log",
  "timestamp_layout": "02/01/2006 15:04:05",
  "post_rotate_signal": "HUP",
  "post_rotate_match": "/usr/local/sbin/zid-geolocation"
}
```

## 7) Permissoes e criacao dos logs
- Garanta que os arquivos de log existam e tenham permissao de leitura para o ZID Logs.
- Se o log for criado pelo seu pacote, mantenha o caminho estavel (o ZID Logs usa este caminho para estado e envio incremental).

## 8) Boas praticas
- Use `package` em lowercase e sem espacos (ex.: `zid-firewall`).
- Use `log_id` curto e estavel (ex.: `main`, `access`, `error`).
- Evite apontar para arquivos temporarios que mudam de nome.

## 9) Validacao
Para validar rapidamente:
- Verifique o arquivo JSON em `/var/db/zid-logs/inputs.d`.
- Execute `zid-logs status` e confira se o log aparece em `inputs`.

## Exemplo completo com multiplos logs
```json
[
  {
    "package": "zid-firewall",
    "log_id": "main",
    "path": "/var/log/zid-firewall.log",
    "timestamp_layout": "2006-01-02T15:04:05-07:00",
    "post_rotate_signal": "HUP",
    "post_rotate_pidfile": "/var/run/zid-firewall.pid"
  },
  {
    "package": "zid-firewall",
    "log_id": "alerts",
    "path": "/var/log/zid-firewall-alerts.log",
    "timestamp_layout": "02/01/2006 15:04:05",
    "post_rotate_signal": "HUP",
    "post_rotate_match": "/usr/local/sbin/zid-firewall",
    "policy": {
      "max_size_mb": 100,
      "keep": 7,
      "compress": true,
      "max_age_days": 10
    }
  }
]
```
