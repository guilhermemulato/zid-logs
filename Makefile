VERSION ?= 0.1.7

.PHONY: version build clean bundle-latest

build:
	@mkdir -p build
	go build -o build/zid-logs ./cmd/zid-logs

clean:
	rm -rf build dist zid-logs-latest.tar.gz sha256.txt zid-logs-latest.version

bundle-latest: build
	@printf "%s\n" "$(VERSION)" > zid-logs-latest.version
	chmod +x scripts/bundle-latest.sh
	./scripts/bundle-latest.sh

.PHONY: version
version:
	@echo $(VERSION)
