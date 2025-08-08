.PHONY: help build test clean docker-build docker-run deploy docker-build-optimized

# 默认目标
help:
	@echo "可用的命令:"
	@echo "  build        - 构建截图服务"
	@echo "  clean        - 清理构建文件"
	@echo "  docker-build - 构建Docker镜像（传统方式）"
	@echo "  docker-build-optimized - 构建Docker镜像（优化方式，分离基础镜像）"
	@echo "  docker-run   - 运行Docker容器"
	@echo "  help         - 显示此帮助信息"

# 构建应用
build:
	@echo "构建截图服务..."
	go build -o bin/screenshot-server cmd/screenshot-server/main.go
	@echo "构建完成: bin/screenshot-server"

# 清理构建文件
clean:
	@echo "清理构建文件..."
	rm -rf bin/
	rm -rf dist/
	@echo "清理完成"

# 构建Docker镜像（传统方式）
docker-build:
	@echo "构建Docker镜像..."
	docker build -t screenshot-server:latest .
	@echo "Docker镜像构建完成"

# 构建Docker镜像（优化方式，分离基础镜像）
docker-build-optimized:
	@echo "构建优化版Docker镜像..."
	@echo "📦 构建基础镜像..."
	docker build -f Dockerfile.base -t screenshot-server-base:latest .
	@echo "🚀 构建应用镜像..."
	docker build -f Dockerfile -t screenshot-server:latest .
	@echo "✅ 优化版Docker镜像构建完成"
	@echo ""
	@echo "📋 可用镜像："
	@echo "  - screenshot-server-base:latest (基础环境)"
	@echo "  - screenshot-server:latest (应用镜像)"

# 测试Docker构建
docker-test:
	@echo "测试Docker构建..."
	./deploy/test-docker-build.sh
	@echo "Docker构建测试完成"

# 运行Docker容器
docker-run:
	@echo "运行Docker容器..."
	docker-compose up -d
	@echo "容器启动完成"

# 停止Docker容器
docker-stop:
	@echo "停止Docker容器..."
	docker-compose down
	@echo "容器停止完成"

# 查看Docker日志
docker-logs:
	docker-compose logs -f



# 本地开发
dev:
	@echo "启动开发模式..."
	go run cmd/screenshot-server/main.go

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 代码检查
lint:
	@echo "运行代码检查..."
	golangci-lint run

# 生成文档
docs:
	@echo "生成API文档..."
	@echo "文档已生成在 docs/ 目录下"

# 打包发布
package:
	@echo "打包应用..."
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -o dist/screenshot-server-linux-amd64 cmd/screenshot-server/main.go
	GOOS=darwin GOARCH=amd64 go build -o dist/screenshot-server-darwin-amd64 cmd/screenshot-server/main.go
	GOOS=windows GOARCH=amd64 go build -o dist/screenshot-server-windows-amd64.exe cmd/screenshot-server/main.go
	@echo "打包完成: dist/"

# 安装依赖
deps:
	@echo "安装依赖..."
	go mod download
	go mod tidy

# 更新依赖
deps-update:
	@echo "更新依赖..."
	go get -u ./...
	go mod tidy

# 检查安全漏洞
security:
	@echo "检查安全漏洞..."
	gosec ./...

# 性能测试
bench:
	@echo "运行性能测试..."
	go test -bench=. ./...

# 覆盖率测试
coverage:
	@echo "运行覆盖率测试..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告: coverage.html"

# 创建发布版本
release:
	@echo "创建发布版本..."
	@read -p "请输入版本号 (如 v1.0.0): " version; \
	git tag $$version; \
	git push origin $$version; \
	echo "发布版本 $$version 创建完成"

# 清理Docker
docker-clean:
	@echo "清理Docker资源..."
	docker system prune -a -f
	docker volume prune -f
	@echo "Docker清理完成"

# 查看服务状态
status:
	@echo "服务状态:"
	@docker-compose ps 2>/dev/null || echo "Docker Compose未运行"
	@echo
	@echo "端口占用:"
	@netstat -tlnp | grep 8080 2>/dev/null || echo "端口8080未占用"

# 备份配置
backup:
	@echo "备份配置文件..."
	tar -czf backup-$(shell date +%Y%m%d-%H%M%S).tar.gz configs/
	@echo "备份完成"

# 恢复配置
restore:
	@echo "恢复配置文件..."
	@read -p "请输入备份文件名: " backup_file; \
	tar -xzf $$backup_file
	@echo "恢复完成" 