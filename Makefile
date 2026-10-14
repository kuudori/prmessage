BINARY = prmessage
VERSION = $(shell grep 'const version' main.go | cut -d'"' -f2)
BINDIR ?= $(shell go env GOPATH)/bin

.PHONY: build install clean

build:
	go build -o $(BINARY) .

install: build
	mkdir -p $(BINDIR)
	cp $(BINARY) $(BINDIR)/
	@case ":$$PATH:" in \
	  *":$(BINDIR):"*) echo "✓ Installed to $(BINDIR) (on your PATH) — run: prmessage" ;; \
	  *) echo ""; \
	     echo "✓ Installed to $(BINDIR), but it is NOT on your PATH."; \
	     echo "  Add it (zsh):"; \
	     echo "    echo 'export PATH=\"$(BINDIR):\$$PATH\"' >> ~/.zshrc && source ~/.zshrc" ;; \
	esac

clean:
	rm -f $(BINARY)
