#!/bin/bash
set -e
echo "==> Compilando PNGTuber Lite..."
go build -ldflags="-s -w" -o pngtuber-lite main.go
echo "==> Concluído! Executável gerado: ./pngtuber-lite"
