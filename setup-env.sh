#!/bin/bash

echo "🔧 配置环境变量..."

# 检查是否已存在.env文件
if [ -f ".env" ]; then
    echo "⚠️  .env文件已存在"
    read -p "是否覆盖？(y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "取消操作"
        exit 0
    fi
fi

# 复制模板文件
cp env.template .env

echo "✅ .env文件已创建"
echo ""
echo "📝 请编辑 .env 文件，填入实际配置："
echo "   nano .env"
echo ""
echo "🔑 必需的配置项："
echo "   - AWS_ACCESS_KEY_ID"
echo "   - AWS_SECRET_ACCESS_KEY"
echo "   - AWS_S3_BUCKET"
echo "   - CDN_BASE_URL"
echo "   - MAFIT_JWT_TOKEN"
echo ""
echo "🚀 配置完成后，可以启动服务："
echo "   docker compose up -d"
