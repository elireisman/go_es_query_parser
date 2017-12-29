all: build

good_env:
	if [ -z "$(GOPATH)" ] ; then (echo "ERROR: must set GOPATH, got: $(GOPATH)" ; exit 1) ; fi

build: good_env *.go
	go get -u github.com/pointlander/peg
	$(GOPATH)/bin/peg -print grammar/dsl.peg
	mkdir -p dist
	go build -o dist/es_dsl -v main.go

clean:
	rm -rf dist &>/dev/null
	find . -name '*.peg.go' -delete &>/dev/null

.PHONY: all build clean
