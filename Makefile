all: build

good_env:
	if [ -z "$(GOPATH)" ] ; then (echo "ERROR: must set GOPATH, got: $(GOPATH)" ; exit 1) ; fi

build: good_env *.go
	go get -u github.com/pointlander/peg
	$(GOPATH)/bin/peg -print grammar/dsl.peg
	mkdir -p dist
	go build -o dist/dsl_to_es_json_parser -v main.go

clean:
	rm -rf dist &>/dev/null

.PHONY: all build clean
