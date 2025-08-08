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

# 更新go.mod并构建应用
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o screenshot-server cmd/screenshot-server/main.go

# 使用基础镜像作为运行阶段
FROM screenshot-server-base:latest

# 从构建阶段复制二进制文件（这部分会经常变化，放在最后）
COPY --from=builder /app/screenshot-server .

# 复制配置文件
COPY --from=builder /app/configs ./configs

# 复制web静态文件
COPY --from=builder /app/web ./web

# 启动应用
CMD ["./screenshot-server"] 