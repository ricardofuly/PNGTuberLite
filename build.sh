#!/bin/bash
set -e
echo "==> Compilando PNGTuber Lite..."
go build -tags noaudio -ldflags="-s -w" -o pngtuber-lite main.go
echo "==> Concluído! Executável gerado: ./pngtuber-lite"
