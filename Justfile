default:
	@just --list

build:
	go build -o bin/sidelight ./cmd/sidelight

install: build
	go install ./cmd/sidelight

test:
	go test -v ./internal/...

preview-styles file="images/output/output-1.jpg": build
	@./bin/preview_all_styles.sh {{file}}
