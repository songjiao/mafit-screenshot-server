# 使用官方Go镜像作为构建阶段
FROM golang:1.22 AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的系统依赖
RUN apt-get update && apt-get install -y git ca-certificates tzdata

# 复制go mod文件
COPY go.mod go.sum ./

# 设置Go代理
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o screenshot-server cmd/screenshot-server/main.go

# 使用Ubuntu作为运行阶段
FROM ubuntu:22.04

# 安装必要的运行时依赖
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    chromium-browser \
    chromium-chromedriver \
    wget \
    curl \
    libnss3 \
    libatk-bridge2.0-0 \
    libdrm2 \
    libxkbcommon0 \
    libxcomposite1 \
    libxdamage1 \
    libxrandr2 \
    libgbm1 \
    libasound2 \
    libpango-1.0-0 \
    libcairo2 \
    libatspi2.0-0 \
    libcups2 \
    libxss1 \
    libxtst6 \
    libx11-xcb1 \
    libxcb-dri3-0 \
    libdrm2 \
    libgbm1 \
    libasound2 \
    libpulse0 \
    libxfixes3 \
    libxrender1 \
    libxrandr2 \
    libxcomposite1 \
    libxcursor1 \
    libxi6 \
    libxt6 \
    libxext6 \
    libx11-6 \
    libxcb1 \
    libxau6 \
    libxdmcp6 \
    fonts-noto-cjk \
    fonts-wqy-microhei \
    fonts-wqy-zenhei \
    && rm -rf /var/lib/apt/lists/*

# 创建非root用户
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -m appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/screenshot-server .

# 复制配置文件
COPY --from=builder /app/configs ./configs

# 复制web静态文件
COPY --from=builder /app/web ./web

# 创建必要的目录
RUN mkdir -p /app/screenshots /app/logs

# 设置权限
RUN chown -R appuser:appgroup /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 设置环境变量
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV CHROME_PATH=/usr/bin/chromium-browser

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动应用
CMD ["./screenshot-server"] 