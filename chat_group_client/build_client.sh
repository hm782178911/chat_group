#!/bin/bash

echo "[0]GOOS=linux GOARCH=amd64 CGO_ENABLED=0  go build -o chat_client_linux_x86"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0  go build -o chat_client_linux_x86

echo "[1]GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o chat_client_linux_arm64"
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o chat_client_linux_arm64

echo "[2]GOOS=windows GOARCH=amd64 CGO_ENABLED=0  go build -o chat_client_windows_x86.exe"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0  go build -o chat_client_windows_x86.exe

echo "[3]GOOS=windows GOARCH=arm64 CGO_ENABLED=0  go build -o chat_client_windows_arm64.exe"
GOOS=windows GOARCH=arm64 CGO_ENABLED=0  go build -o chat_client_windows_arm64.exe

echo "[4]GOOS=darwin GOARCH=amd64 CGO_ENABLED=0  go build -o chat_client_macos_x86"
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0  go build -o chat_client_macos_x86

echo "[5]GOOS=darwin GOARCH=arm64 CGO_ENABLED=0  go build -o chat_client_macos_arm64"
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0  go build -o chat_client_macos_arm64


