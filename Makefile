BINARY := googrec
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X github.com/boris/googrec/cmd.Version=$(VERSION)"

.PHONY: build test vet clean install

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)

install:
	go install $(LDFLAGS) .
