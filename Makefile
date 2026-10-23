.PHONY: run build test race cover vet docker-build docker-run clean

PORT ?= 7860
IMAGE_NAME ?= bioscope:latest

run:
	PORT=$(PORT) go run ./cmd/server

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bioscope ./cmd/server

test:
	go test -v ./...

race:
	go test -race -v ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

vet:
	go vet ./...

docker-build:
	docker build -t $(IMAGE_NAME) .

docker-run: docker-build
	docker run --rm -p $(PORT):7860 -e PORT=7860 $(IMAGE_NAME)

clean:
	rm -f bioscope coverage.out
