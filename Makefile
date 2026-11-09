# Punisher standalone — build helpers.
#   make build   → dist/punisher (universal macOS binary, frontend embedded)
#   make test    → go vet + go test + frontend typecheck/lint
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
WEBUI_DIST := internal/webui/dist

.PHONY: build web go-darwin test lint clean run release

build: web go-darwin

web:
	cd web && npm ci --no-audit --no-fund
	find $(WEBUI_DIST) -mindepth 1 ! -name .gitkeep -delete
	cd web && npm run build

# Pure Go (modernc SQLite, no CGO) so both architectures cross-compile anywhere.
go-darwin:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags '$(LDFLAGS)' -o dist/punisher-darwin-arm64 ./cmd/punisher
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags '$(LDFLAGS)' -o dist/punisher-darwin-amd64 ./cmd/punisher
	lipo -create -output dist/punisher dist/punisher-darwin-arm64 dist/punisher-darwin-amd64
	@lipo -info dist/punisher
	@ls -lh dist/punisher

test:
	go vet ./...
	go test ./...
	cd web && npm run typecheck && npm run lint && npm test

# Release zip for hand-off: punisher + install.sh + README + CHANGELOG (no publishing).
release: build
	mkdir -p dist/release/punisher-$(VERSION)
	cp dist/punisher install.sh README.md README.ko.md CHANGELOG.md LICENSE dist/release/punisher-$(VERSION)/
	cd dist/release && ditto -c -k --keepParent punisher-$(VERSION) punisher-$(VERSION)-macos-universal.zip
	@ls -lh dist/release/*.zip

run:
	go run ./cmd/punisher

clean:
	find dist -mindepth 1 -delete
	find $(WEBUI_DIST) -mindepth 1 ! -name .gitkeep -delete
