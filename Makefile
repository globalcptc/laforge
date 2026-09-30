# LaForge release artifacts for volunteer developers.
#
# Produces, into ./dist:
#   - the `laforge` CLI (for `laforge check`) for Linux, Windows, and macOS
#   - the `laforge-lsp` language server (the VS Code extension needs it on PATH)
#   - the `laforge-vscode-<version>.vsix` VS Code extension package
#   - SHA256SUMS.txt over all of the above
# Upload the contents of ./dist to a GitHub Release for volunteers to download.
#
# Usage:
#   make release VERSION=v3.0.0     # everything, stamped with that version
#   make cli VERSION=v3.0.0         # just the cross-compiled binaries
#   make vsix                       # just the .vsix
#   make clean
#
# Requires: Go 1.22+, Node 18+ / npm (for the .vsix). No CGO, no cross toolchain.

VERSION ?= dev
DIST    := dist
LDFLAGS := -s -w -X main.version=$(VERSION)

# os/arch pairs to cross-compile. Linux and Windows are the volunteer targets;
# macOS (darwin) is included as a bonus for Mac users. Trim this list to change
# what gets built.
PLATFORMS := linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64

# The commands built for every platform: the CLI and the language server.
BINARIES := laforge laforge-lsp

.DEFAULT_GOAL := help
.PHONY: help release cli vsix checksums clean

help:
	@echo "LaForge release build. Targets:"
	@echo "  make release VERSION=vX.Y.Z   build binaries + .vsix + checksums into ./$(DIST)"
	@echo "  make cli     VERSION=vX.Y.Z   cross-compile laforge + laforge-lsp for all platforms"
	@echo "  make vsix                     build the VS Code extension .vsix"
	@echo "  make checksums                write ./$(DIST)/SHA256SUMS.txt"
	@echo "  make clean                    remove ./$(DIST)"
	@echo ""
	@echo "  VERSION defaults to 'dev'. Platforms: $(PLATFORMS)"

release: cli vsix checksums
	@echo "Release artifacts ready in ./$(DIST):"
	@ls -1 $(DIST)

cli:
	@mkdir -p $(DIST)
	@for platform in $(PLATFORMS); do \
	  os=$${platform%/*}; arch=$${platform#*/}; ext=""; \
	  [ "$$os" = "windows" ] && ext=".exe"; \
	  for bin in $(BINARIES); do \
	    out="$(DIST)/$$bin-$(VERSION)-$$os-$$arch$$ext"; \
	    echo "  building $$out"; \
	    CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o "$$out" ./cmd/$$bin || exit 1; \
	  done; \
	done

vsix:
	@mkdir -p $(DIST)
	cp LICENSE editors/vscode/LICENSE
	cd editors/vscode && npm ci && npm run compile && \
	  npx --yes @vscode/vsce package --out "$(CURDIR)/$(DIST)/laforge-vscode-$(VERSION).vsix"

checksums:
	@cd $(DIST) && rm -f SHA256SUMS.txt && \
	  { command -v sha256sum >/dev/null 2>&1 && sha256sum * > SHA256SUMS.txt || shasum -a 256 * > SHA256SUMS.txt; } && \
	  echo "  wrote $(DIST)/SHA256SUMS.txt"

clean:
	rm -rf $(DIST)
