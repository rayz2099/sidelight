default:
	@just --list

build:
	go build -o bin/sidelight ./cmd/sidelight

install:
	go install ./cmd/sidelight

test:
	go test -v ./internal/...
