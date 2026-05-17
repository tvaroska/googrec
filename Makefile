BINARY := googrec

.PHONY: build test vet clean install

build:
	go build -o $(BINARY) .

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)

install:
	go install .
