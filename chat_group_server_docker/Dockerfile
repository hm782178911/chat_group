# 构建阶段
FROM golang:1.25-alpine AS builder

# 设置工作目录
WORKDIR /app

# 设置 GOPROXY 为国内镜像源
ENV GOPROXY=https://goproxy.cn,direct
ENV GOPRIVATE=
ENV GOSUMDB=off

# 复制 go.mod 和 go.sum 文件并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建 Go 二进制文件，禁用 CGO 并优化
# RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o chat-server .
# RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o chat-server ./chat_group_server.go
# 编译 Go 源代码
RUN go build -o chat_group_server chat_group_server.go

# 最终运行阶段：使用最精简的 Alpine 镜像
FROM alpine:latest

# 安装可能需要的运行时依赖（如 CA 证书）
RUN apk --no-cache add ca-certificates

# 设置工作目录
WORKDIR /root/

# 从构建阶段复制编译好的二进制文件
# COPY --from=builder /app/chat-server .
COPY --from=builder /app/chat_group_server /usr/local/bin/chat_group_server

# 暴露应用程序运行的端口
EXPOSE 8080

# 定义容器启动时执行的命令
# CMD ["./chat-server"]
CMD ["chat_group_server"]
