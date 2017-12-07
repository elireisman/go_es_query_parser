all: build

build: *.go
	mkdir -p dist
	go build -o dist/dsl_to_es_json_parser -v main.go

clean:
	rm -rf dist &>/dev/null

.PHONY: all build clean
