.PHONY: build test race vet fmt check clean

build:
	go build -o bin/safeops ./cmd/safeops

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w cmd internal

check: test race vet

clean:
	rm -rf bin dist
