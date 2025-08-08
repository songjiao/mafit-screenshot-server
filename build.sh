#!/bin/bash

set -e

echo "🔧 开始构建Docker镜像..."

# 构建基础镜像（包含运行环境依赖）
echo "📦 构建基础镜像..."
docker build -f Dockerfile.base -t screenshot-server-base:latest .

# 构建应用镜像（基于基础镜像）
echo "🚀 构建应用镜像..."
docker build -f Dockerfile -t screenshot-server:latest .

echo "✅ 镜像构建完成！"
echo ""
echo "📋 可用镜像："
echo "  - screenshot-server-base:latest (基础环境)"
echo "  - screenshot-server:latest (应用镜像)"
echo ""
echo "🚀 启动服务："
echo "  docker compose up -d"
