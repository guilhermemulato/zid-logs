# Licenciamento ZID via `zid-packages` (pfSense)

## Objetivo
Centralizar a validacao de licenca dos pacotes:
- `zid-proxy` (inclui `zid-appid`)
- `zid-geolocation`
- `zid-logs`

O pacote `zid-packages` e o "servidor local" de licencas no pfSense.
Cada pacote ZID **deve consultar** o `zid-packages` para saber se pode executar.
Se o `zid-packages` estiver ausente (desinstalado) ou negar a licenca, o pacote **deve parar imediatamente**.

## Regras de execucao (obrigatorias)
1) Ao iniciar o servico do pacote, antes de processar qualquer logica, o pacote deve consultar a licenca.
2) Se a licenca for **invalida**:
   - o pacote deve encerrar o processo com exit code != 0
   - e, se aplicavel, solicitar stop no rc.d (ou simplesmente finalizar e deixar o watchdog central garantir o stop)
3) Se nao for possivel consultar o `zid-packages` (socket nao existe / erro de comunicacao):
   - tratar como **SEM LICENCA** e encerrar/parar
4) Mesmo com licenca valida no start, o pacote deve revalidar periodicamente (ex: a cada 60s ou 5min) consultando o `zid-packages`.
   - Se a licenca passar a ser invalida, parar imediatamente.

## Canal de consulta (IPC local)
O `zid-packages` expoe um Unix Domain Socket:
- Path: `/var/run/zid-packages.sock`
- Permissoes: acesso restrito (root), socket local

### Modelo de seguranca
A mensagem e autenticada com HMAC (assinatura) para evitar spoofing local.
A chave e derivada no runtime usando informacoes do proprio sistema (ex: `/var/db/uniqueid` e `hostname -s`) e um segredo embutido no binario.

### Request (exemplo)
JSON:
{
  "op": "CHECK",
  "package": "zid-proxy",
  "ts": 1700000000,
  "nonce": "random-hex",
  "sig": "hmac-hex"
}

Campos:
- `op`: sempre `CHECK`
- `package`: `zid-proxy` | `zid-geolocation` | `zid-logs`
- `ts`: timestamp unix (segundos)
- `nonce`: string aleatoria para evitar replay
- `sig`: HMAC-SHA256 do payload (sem o campo sig)

### Response (exemplo)
{
  "ok": true,
  "licensed": true,
  "mode": "OK",
  "valid_until": 1700600000,
  "reason": "valid",
  "ts": 1700000001,
  "sig": "hmac-hex"
}

Campos:
- `licensed`: true/false
- `mode`:
  - `OK` (online recente)
  - `OFFLINE_GRACE` (sem internet mas dentro da janela de 7 dias)
  - `EXPIRED` (passou de 7 dias sem sucesso)
  - `NEVER_OK` (nunca validou online e nao conseguiu validar agora)
- `valid_until`: timestamp unix do limite (last_success + 7 dias) quando aplicavel
- `reason`: texto curto para log/diagnostico
- `sig`: assinatura da resposta

## Comportamento esperado no pacote (pseudocodigo)
- Na inicializacao:
  - if !LicenseCheck("zid-proxy") => exit(2)
- Em loop/intervalo:
  - if !LicenseCheck("zid-proxy") => stop imediatamente (exit / shutdown)

## Observacoes
- O watchdog central do `zid-packages` tambem aplica enforcement (stop/start), mas o pacote deve se auto-proteger.
- Se o cliente desinstalar o `zid-packages`, o socket desaparece e todos os pacotes devem parar.
- Nao persistir chaves de licenca em arquivos. Toda validacao e por consulta ao `zid-packages`.

## Debug temporario (IPC)
Para diagnosticar comunicacao entre pacote e `zid-packages`, existem flags de ambiente:
- `ZID_PACKAGES_IPC_DEBUG=1` -> loga detalhes da request/response no `/var/log/zid-packages.log`.
- `ZID_PACKAGES_IPC_LOG_KEYS=1` -> loga a key derivada (hex) no `/var/log/zid-packages.log`.

Use apenas para diagnostico e remova apos validar.

## Erros comuns (bad_sig)
Se aparecer `bad_sig` no `zid-packages.log`, geralmente e um destes pontos:
1) **Assinando o payload errado (request)**  
   O `sig` deve ser calculado sobre o JSON do **struct completo** com `sig` vazio (`""`).  
   Se assinar apenas `requestPayload` (sem campo `sig`), a assinatura nao vai bater.
2) **Validando resposta sem `sig` vazio**  
   O servidor assina a **resposta completa** com `sig` vazio.  
   Se o cliente validar usando somente os campos sem `sig`, vai dar `bad_sig`.
3) **Uso de `map` ao inves de `struct`**  
   Assinar JSON gerado de `map` pode mudar a ordem dos campos e quebrar o HMAC.  
   Use sempre `struct` com tags iguais.

## Referencia: DeriveKey e assinatura (zid-proxy)
Para garantir compatibilidade com o `zid-packages`, use exatamente o mesmo algoritmo de derivacao e assinatura:

### DeriveKey
- `host`: hostname curto (sem dominio)
- `uid`: conteudo de `/var/db/uniqueid`
- `salt`: `uid + ":" + host`
- `info`: `"zid-packages-hkdf"`
- `masterSecret`: `"zid-packages-master-secret-2026-01"`
- HKDF SHA256 com key de 32 bytes

```go
package licensing

import (
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/hkdf"
)

const masterSecret = "zid-packages-master-secret-2026-01"

func shortHostname() (string, error) {
	host, err := os.Hostname()
	if err != nil {
		return "", err
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return "", errors.New("hostname vazio")
	}
	if idx := strings.IndexByte(host, '.'); idx > 0 {
		host = host[:idx]
	}
	return host, nil
}

func uniqueID() (string, error) {
	raw, err := os.ReadFile("/var/db/uniqueid")
	if err != nil {
		return "", err
	}
	uid := strings.TrimSpace(string(raw))
	if uid == "" {
		return "", errors.New("uniqueid vazio")
	}
	return uid, nil
}

func deriveKey() ([]byte, error) {
	host, err := shortHostname()
	if err != nil {
		return nil, err
	}
	uid, err := uniqueID()
	if err != nil {
		return nil, err
	}
	salt := []byte(uid + ":" + host)
	info := []byte("zid-packages-hkdf")

	reader := hkdf.New(sha256.New, []byte(masterSecret), salt, info)
	key := make([]byte, 32)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, err
	}
	return key, nil
}
```

### Assinatura do request
- Assinar o JSON **sem o campo `sig`**
- Use `json.Marshal` de um **struct** (nao map), para manter ordem/forma estavel

```go
type Request struct {
	Op      string `json:"op"`
	Package string `json:"package"`
	TS      int64  `json:"ts"`
	Nonce   string `json:"nonce"`
	Sig     string `json:"sig"`
}

func signHex(key, payload []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func buildRequest(pkg string, ts int64, nonce string) (Request, error) {
	key, err := deriveKey()
	if err != nil {
		return Request{}, err
	}

	req := Request{
		Op:      "CHECK",
		Package: pkg,
		TS:      ts,
		Nonce:   nonce,
		Sig:     "",
	}
	payload, _ := json.Marshal(req)
	req.Sig = signHex(key, payload)
	return req, nil
}
```