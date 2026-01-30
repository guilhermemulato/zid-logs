VERSION ?= 0.1.10.19.4

.PHONY: version build build-freebsd clean bundle-latest

build:
	@mkdir -p build
	go build -o build/zid-logs ./cmd/zid-logs

build-freebsd:
	@mkdir -p build
	GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build -o build/zid-logs ./cmd/zid-logs

clean:
	rm -rf build dist zid-logs-latest.tar.gz sha256.txt zid-logs-latest.version

bundle-latest: build-freebsd
	@printf "%s\n" "$(VERSION)" > zid-logs-latest.version
	chmod +x scripts/bundle-latest.sh
	./scripts/bundle-latest.sh

.PHONY: version
version:
	@echo $(VERSION)
