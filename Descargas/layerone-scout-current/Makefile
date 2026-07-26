BINARY=scout
VERSION=1.0.0
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

all: build

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/scout

run: build
	./$(BINARY) server

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY) profiles.json profiles.json.lock
	rm -rf backups/

build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-linux-amd64 ./cmd/scout

build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-windows-amd64.exe ./cmd/scout

build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-darwin-amd64 ./cmd/scout

release: build-linux build-windows build-darwin

docker-build:
	docker build -t layerone-scout .

docker-run:
	docker run --rm -p 8787:8787 -v $(PWD)/data:/data layerone-scout

.PHONY: all build run test fmt vet lint clean build-linux build-windows build-darwin release docker-build docker-run
