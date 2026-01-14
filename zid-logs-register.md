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

## 5) Permissoes e criacao dos logs
- Garanta que os arquivos de log existam e tenham permissao de leitura para o ZID Logs.
- Se o log for criado pelo seu pacote, mantenha o caminho estavel (o ZID Logs usa este caminho para estado e envio incremental).

## 6) Boas praticas
- Use `package` em lowercase e sem espacos (ex.: `zid-firewall`).
- Use `log_id` curto e estavel (ex.: `main`, `access`, `error`).
- Evite apontar para arquivos temporarios que mudam de nome.

## 7) Validacao
Para validar rapidamente:
- Verifique o arquivo JSON em `/var/db/zid-logs/inputs.d`.
- Execute `zid-logs status` e confira se o log aparece em `inputs`.

## Exemplo completo com multiplos logs
```json
[
  {
    "package": "zid-firewall",
    "log_id": "main",
    "path": "/var/log/zid-firewall.log"
  },
  {
    "package": "zid-firewall",
    "log_id": "alerts",
    "path": "/var/log/zid-firewall-alerts.log",
    "policy": {
      "max_size_mb": 100,
      "keep": 7,
      "compress": true,
      "max_age_days": 10
    }
  }
]
```
