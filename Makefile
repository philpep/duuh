all: build
.PHONY: all

build: main.go
	go build
.PHONY: build

install: build
	go install
.PHONY: install

clean:
	rm -f duuh
.PHONY: clean

test:
	go test -v
.PHONY: test

check:
	errcheck ./...
	go fmt
	goimports -w .
.PHONY: check
