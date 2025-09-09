#!/bin/bash

echo "[0]GOOS=linux GOARCH=amd64 CGO_ENABLED=0  go build -o chat_server_linux_x86 chat_group_server.go"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0  go build -o chat_server_linux_x86 chat_group_server.go


echo "[1]GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o chat_server_linux_arm64 chat_group_server.go"
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o chat_server_linux_arm64 chat_group_server.go

echo "[2]GOOS=windows GOARCH=amd64 CGO_ENABLED=0  go build -o chat_server_windows_x86.exe chat_group_server.go"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0  go build -o chat_server_windows_x86.exe chat_group_server.go

echo "[3]GOOS=windows GOARCH=arm64 CGO_ENABLED=0  go build -o chat_server_windows_arm64.exe chat_group_server.go"
GOOS=windows GOARCH=arm64 CGO_ENABLED=0  go build -o chat_server_windows_arm64.exe chat_group_server.go


echo "[4]GOOS=darwin GOARCH=amd64 CGO_ENABLED=0  go build -o chat_server_macos_x86 chat_group_server.go"
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0  go build -o chat_server_macos_x86 chat_group_server.go

echo "[5]GOOS=darwin GOARCH=arm64 CGO_ENABLED=0  go build -o chat_server_macos_arm64 chat_group_server.go"
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0  go build -o chat_server_macos_arm64 chat_group_server.go



