BINARY = prmessage
VERSION = $(shell grep 'const version' main.go | cut -d'"' -f2)

.PHONY: build install clean

build:
	go build -o $(BINARY) .

install: build
	cp $(BINARY) ~/bin/

clean:
	rm -f $(BINARY)
