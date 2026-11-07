.PHONY: all test test-race build run tidy clean help

all: test build

test:
	go test -short -v ./...

test-race:
	go test -short -v -race ./...

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

tidy:
	go mod tidy
	go mod verify

clean:
	rm -rf bin/

help:
	@echo "Perintah yang tersedia:"
	@echo "  make test      - Menjalankan unit test"
	@echo "  make test-race - Menjalankan unit test dengan race detector"
	@echo "  make build     - Melakukan kompilasi binary ke bin/server"
	@echo "  make run       - Menjalankan server lokal"
	@echo "  make tidy      - Merapikan dependensi go.mod dan go.sum"
	@echo "  make clean     - Menghapus direktori bin/"
