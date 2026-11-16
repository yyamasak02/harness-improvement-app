.PHONY: dev test build dmg verify bindings
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

# macOS 専用。署名・公証はしない。
dmg:
	$(WAILS) build -platform darwin/universal -clean
	hdiutil create -volname "harness-app" -srcfolder build/bin/harness-app.app -ov -format UDZO build/bin/harness-app.dmg

bindings:
	$(WAILS) generate module
