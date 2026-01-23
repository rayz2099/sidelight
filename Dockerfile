# 多阶段构建 - 构建阶段
FROM golang:1.25.4-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装 git 和 ca-certificates
RUN apk add --no-cache git ca-certificates

# 设置 Go 代理和环境变量
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org
ENV CGO_ENABLED=0

# 复制 go.mod 和 go.sum 并下载依赖
COPY go.mod go.sum ./
RUN go mod tidy && go mod download

# 复制源代码
COPY . .

# 构建二进制文件
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/sidelight ./cmd/sidelight

# 运行时阶段
FROM ubuntu:22.04

# 设置非交互式安装，避免提示
ENV DEBIAN_FRONTEND=noninteractive

# 更新包列表并安装必要的软件包
RUN apt-get update && \
    apt-get install -y \
    ca-certificates \
    libimage-exiftool-perl \
    rawtherapee \
    curl \
    && rm -rf /var/lib/apt/lists/*

# 验证 rawtherapee-cli 安装并创建软链接（如果需要）
RUN which rawtherapee-cli || \
    (test -f /usr/bin/rawtherapee && ln -s /usr/bin/rawtherapee /usr/bin/rawtherapee-cli) || \
    echo "Warning: rawtherapee-cli not found"

# 创建非 root 用户
RUN useradd -r -s /bin/false -m -d /app sidelight

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/bin/sidelight /usr/local/bin/sidelight
COPY --from=builder /app/assets /app/assets

# 创建必要的目录
RUN mkdir -p /app/data /app/config && \
    chown -R sidelight:sidelight /app

# 设置环境变量
ENV RT_CLI_PATH=/usr/bin/rawtherapee-cli

# 切换到非 root 用户
USER sidelight

# 暴露端口（用于 web server 模式）
EXPOSE 8080

# 设置默认命令
ENTRYPOINT ["sidelight"]
CMD ["--help"]