## Padroes e Regras de Desenvolvimento

- Sempre leia o arquivo **specs.md**, nela vai encontrar tudo que precisa referente ao projeto.
- Mantenha informacoes como funcionalidades sempre atualizada no arquivo **specs.md**
- Sempre responda em portugues PT-BR
- Use o MCP Context7 para buscar informaoes sobre documentacoes, se manter atualizado
- Use o MCP github para acessar os repositorios que tem relacao com esse projeto, como o **zid-proxy(guilhermemulato/zid-proxy)** e **zid-geolocation(guilhermemulato/zid-geolocation)**. Neles tambem ja foram implementadas diversas funcionalidades na WEB Gui do pfsense, entao voce pode consulta la quando tiver duvidas para ver como foi feito
- Go: sempre rodar `gofmt -w .` antes de entregar alteracoes.
- Testes: preferir testes determinísticos em `internal/*/*_test.go`.
- Mudou codigo? Atualize o `CHANGELOG.md` e **bump de versao** no `Makefile`.
- Alteracao pequena: use sufixo incremental (ex.: `1.6.1`).
- Alteracao grande: use sufixo incremental (ex.: `1.6`).
- Ao final, gere novamente os bundles (`make bundle-latest`) e garanta:
    - `zid-logs-latest.version` atualizado
    - `sha256.txt` atualizado
    - **bundles obrigatorios** (zid-logs-latest.tar.gz)
- Sempre executar ao final de cada implementacao: `go test ./...` e `go build ./...`

## Estrutura atual do repositorio

- cmd/zid-logs/
- internal/{config,registry,rotate,shipper,state,status}/
- packaging/pfsense/
- gui/
- tests/

## Comandos

Ainda nao ha comandos de build/test definidos no repositorio.
