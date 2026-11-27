# Punisher standalone — build helpers.
#   npm ci needs NODE_AUTH_TOKEN for the @forgen-team/* packages (GitHub Packages, web/.npmrc): NODE_AUTH_TOKEN=$(gh auth token) make …
#   make build   → dist/punisher (universal macOS binary, frontend embedded)
#   make linux   → dist/punisher-linux-amd64, dist/punisher-linux-arm64
#   make test    → go vet + go test + frontend typecheck/lint
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
WEBUI_DIST := internal/webui/dist
WITEMS_DIST := internal/witems/dist

.PHONY: build web witems-web image go-darwin go-linux linux test lint clean run release

build: web go-darwin

web:
	cd web && npm ci --no-audit --no-fund
	find $(WEBUI_DIST) -mindepth 1 ! -name .gitkeep -delete
	cd web && npm run build

# Witems app mode: the Module Federation remote (web/src/witems/mount.tsx) embedded into internal/witems.
witems-web:
	cd web && npm ci --no-audit --no-fund
	find $(WITEMS_DIST) -mindepth 1 ! -name .gitkeep -delete
	cd web && npm run build:witems
	@# CSS imported from components would be injected into the shell's <head>, outside the Shadow DOM.
	@if ls $(WITEMS_DIST)/assets/*.css >/dev/null 2>&1; then echo "witems-web: stray CSS files (import CSS in styles/globals.css instead):"; ls $(WITEMS_DIST)/assets/*.css; exit 1; fi
	@# The shell owns the design tokens (@forgen-team/tokens on its <html>): the inline sheet must not redeclare them (@forgen-team/postcss-shadow-scope, web/vite.witems.config.ts).
	@if grep -lE -e '--color-(bg|accent|text|border|focus|on)[a-z-]*:' -e '--(radius|shadow|font)-[a-z-]+:' -e '--brand-orange:' $(WITEMS_DIST)/assets/*.js; then echo "witems-web: design-token declarations in the Witems build (the shell provides them)"; exit 1; fi
	@# Accent colours come from the shell's tokens, never as literals. CSS values only (prop:#hex, Tailwind's rgb(r g b …)) — the product logo SVG (stopColor:"#f97316") and mermaid's palette ("#FB923C") are not the accent.
	@if grep -liE -e ':[[:space:]]*#(f97316|ea580c|c2410c|fb923c|fff1e6|2a1708)([^0-9a-f]|$$)' -e 'rgb\((249 115 22|234 88 12|194 65 12|251 146 60|255 241 230|42 23 8)[ /)]' $(WITEMS_DIST)/assets/*.js; then echo "witems-web: standalone orange accent in the Witems build"; exit 1; fi

# Witems app-mode container image (Dockerfile). The npm token is passed as a BuildKit secret.
image:
	docker build --secret id=npm_token,env=NODE_AUTH_TOKEN -t punisher:dev --build-arg VERSION=$(VERSION) .

# Pure Go (modernc SQLite, no CGO) so both architectures cross-compile anywhere.
go-darwin:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags '$(LDFLAGS)' -o dist/punisher-darwin-arm64 ./cmd/punisher
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags '$(LDFLAGS)' -o dist/punisher-darwin-amd64 ./cmd/punisher
	lipo -create -output dist/punisher dist/punisher-darwin-arm64 dist/punisher-darwin-amd64
	@lipo -info dist/punisher
	@ls -lh dist/punisher

go-linux:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags '$(LDFLAGS)' -o dist/punisher-linux-amd64 ./cmd/punisher
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags '$(LDFLAGS)' -o dist/punisher-linux-arm64 ./cmd/punisher
	@ls -lh dist/punisher-linux-*

linux: web go-linux

test:
	go vet ./...
	go test ./...
	cd web && npm run typecheck && npm run lint && npm test

# Release archives for hand-off: punisher + install.sh + README + CHANGELOG (no publishing).
#   macOS: punisher-<v>-macos-universal.zip · Linux: punisher-<v>-linux-{amd64,arm64}.tar.gz
DOCS := install.sh README.md README.ko.md CHANGELOG.md LICENSE NOTICE
release: build go-linux
	rm -rf dist/release && mkdir -p dist/release/punisher-$(VERSION)
	cp dist/punisher $(DOCS) dist/release/punisher-$(VERSION)/
	cd dist/release && ditto -c -k --keepParent punisher-$(VERSION) punisher-$(VERSION)-macos-universal.zip
	for arch in amd64 arm64; do \
		d=dist/release/linux-$$arch/punisher-$(VERSION); mkdir -p $$d; \
		cp dist/punisher-linux-$$arch $$d/punisher; cp $(DOCS) $$d/; \
		tar -C dist/release/linux-$$arch -czf dist/release/punisher-$(VERSION)-linux-$$arch.tar.gz punisher-$(VERSION); \
	done
	@ls -lh dist/release/*.zip dist/release/*.tar.gz

run:
	go run ./cmd/punisher

clean:
	find dist -mindepth 1 -delete
	find $(WEBUI_DIST) -mindepth 1 ! -name .gitkeep -delete
