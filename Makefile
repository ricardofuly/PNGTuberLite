# Makefile for PNGTuber Lite

APP_NAME = pngtuber-lite
LDFLAGS = -s -w

.PHONY: all build build-linux build-windows test clean run

all: test build

build: build-linux

build-linux:
	@echo "==> Building $(APP_NAME) for Linux..."
	go build -ldflags="$(LDFLAGS)" -o $(APP_NAME) main.go
	@echo "==> Build complete: $(APP_NAME)"

build-windows:
	@echo "==> Building $(APP_NAME) for Windows (.exe)..."
	CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
	go build -ldflags="$(LDFLAGS)" -o $(APP_NAME).exe main.go
	@echo "==> Windows build complete: $(APP_NAME).exe"

test:
	@echo "==> Running all unit tests..."
	go test -v ./pkg/...

clean:
	@echo "==> Cleaning build artifacts..."
	rm -f $(APP_NAME) $(APP_NAME).exe
	@echo "==> Clean complete."

run: build-linux
	./$(APP_NAME) -avatar assets/samples/defaultAvatar.save
