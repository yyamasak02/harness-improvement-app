.PHONY: dev test build dmg windows windows-installer verify bindings setup lint verify-full
WAILS ?= $(shell command -v wails || echo $(HOME)/go/bin/wails)

dev:
	$(WAILS) dev

test:
	go vet ./... && go test ./...
	cd frontend && npm run build && npm test

verify:
	bash scripts/verify.sh

build:
	$(WAILS) build

# Windows 用 exe（Linux/macOS からもクロスビルドできる）。署名はしない。
windows:
	$(WAILS) build -platform windows/amd64 -clean

# Windows 用インストーラ（NSIS の makensis が必要。Windows では `choco install nsis`）。
windows-installer:
	$(WAILS) build -platform windows/amd64 -clean -nsis

# macOS 専用。署名・公証はしない。
dmg:
	$(WAILS) build -platform darwin/universal -clean
	hdiutil create -volname "harness-app" -srcfolder build/bin/harness-app.app -ov -format UDZO build/bin/harness-app.dmg

bindings:
	$(WAILS) generate module

setup:
	pre-commit install --install-hooks

lint:
	bash scripts/lint.sh all

verify-full:
	bash scripts/verify.sh --full
