BINARY   := storage-service
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -ldflags "-s -w -X main.version=$(VERSION)"
TRIMPATH := -trimpath

# ── Local build ──────────────────────────────────────────────────────────────

.PHONY: build
build:
	go build $(TRIMPATH) $(LDFLAGS) -o bin/$(BINARY) ./cmd/server

# ── Quality gates ─────────────────────────────────────────────────────────────

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt-check
fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "These files need gofmt:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

.PHONY: test
test:
	go test -race -count=1 -timeout 60s ./...

.PHONY: lint
lint: vet fmt-check

# ── Cross-compilation ─────────────────────────────────────────────────────────
# CGO_ENABLED=0 produces fully static binaries (no glibc dependency on Linux).
# GOFLAGS is intentionally not used so targets are explicit.

.PHONY: cross
cross: cross-linux-amd64 cross-linux-arm64 cross-darwin-amd64 cross-windows-amd64

.PHONY: cross-linux-amd64
cross-linux-amd64:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 \
		go build $(TRIMPATH) $(LDFLAGS) -o bin/$(BINARY)-linux-amd64 ./cmd/server

.PHONY: cross-linux-arm64
cross-linux-arm64:
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 \
		go build $(TRIMPATH) $(LDFLAGS) -o bin/$(BINARY)-linux-arm64 ./cmd/server

.PHONY: cross-darwin-amd64
cross-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 \
		go build $(TRIMPATH) $(LDFLAGS) -o bin/$(BINARY)-darwin-amd64 ./cmd/server

.PHONY: cross-windows-amd64
cross-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
		go build $(TRIMPATH) $(LDFLAGS) -o bin/$(BINARY)-windows-amd64.exe ./cmd/server

.PHONY: clean
clean:
	rm -rf bin/
