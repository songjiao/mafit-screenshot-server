#!/bin/bash

# GPT Actions 快速配置脚本
# 这个脚本帮助您快速配置GPT Actions

set -e

echo "=== GPT Actions 快速配置脚本 ==="
echo ""

# 检查必要文件
echo "1. 检查必要文件..."
if [ ! -f "openapi.yaml" ]; then
    echo "❌ 错误: openapi.yaml 文件不存在"
    exit 1
fi

if [ ! -f "gpt-actions-config.json" ]; then
    echo "❌ 错误: gpt-actions-config.json 文件不存在"
    exit 1
fi

echo "✅ 必要文件检查通过"
echo ""

# 显示当前配置
echo "2. 当前配置信息..."
echo "📄 OpenAPI规范文件: openapi.yaml"
echo "⚙️  GPT配置文件: gpt-actions-config.json"
echo ""

# 检查服务器地址配置
echo "3. 检查服务器地址配置..."
if grep -q "your-domain.com" openapi.yaml; then
    echo "⚠️  警告: 需要修改服务器地址"
    echo "   请在 openapi.yaml 中将 'your-domain.com' 替换为您的实际服务地址"
    echo ""
    echo "   当前配置:"
    grep -A 2 "servers:" openapi.yaml
    echo ""
else
    echo "✅ 服务器地址已配置"
fi

# 显示配置步骤
echo "4. GPT Actions 配置步骤:"
echo ""
echo "📋 步骤1: 访问 ChatGPT"
echo "   打开 https://chat.openai.com/"
echo ""
echo "📋 步骤2: 创建新的 GPT"
echo "   点击 'Explore GPTs' → 'Create a GPT'"
echo ""
echo "📋 步骤3: 配置 Actions"
echo "   在GPT编辑器中:"
echo "   - 点击 'Configure'"
echo "   - 滚动到 'Actions' 部分"
echo "   - 点击 'Add actions'"
echo "   - 选择 'Import from URL' 或 'Import from file'"
echo "   - 输入: https://your-domain.com/openapi.yaml"
echo "   或者上传 openapi.yaml 文件"
echo ""
echo "📋 步骤4: 配置GPT指令"
echo "   在 'Instructions' 中添加以下内容:"
echo ""
cat << 'EOF'
你是一个专业的股票分析助手，可以帮助用户获取股票K线图截图。

## 功能说明
- 支持美股(us)、港股(hk)、A股(cn)的股票截图
- 支持日线(1d)、小时线(1h)、周线(1wk)时间框架
- 可以同时获取截图和JSON数据

## 使用示例
用户可以说：
- "帮我截取NVDA的日线图"
- "获取TSLA的小时线截图和数据"
- "查看AAPL的周线图"

## 参数说明
- symbol: 股票代码，如 NVDA, AAPL, TSLA
- market: 市场代码，us(美股), hk(港股), cn(A股)
- timeframe: 时间框架，1d(日线), 1h(小时线), 1wk(周线)

## 返回结果
- cdn_url: 截图访问地址
- data_cdn_url: JSON数据访问地址（如果请求了数据）
EOF
echo ""
echo "📋 步骤5: 测试配置"
echo "   保存配置后，可以测试以下命令:"
echo "   - '请检查截图服务的健康状态'"
echo "   - '帮我截取NVDA的日线图'"
echo "   - '获取TSLA的小时线截图和数据'"
echo ""

# 显示API接口信息
echo "5. API接口信息:"
echo ""
echo "🔗 主要接口:"
echo "   - 健康检查: GET /health"
echo "   - 服务状态: GET /api/v1/status"
echo "   - 截图接口: POST /api/v1/screenshot"
echo "   - 带数据截图: POST /api/v1/screenshot-with-data"
echo "   - GET方式截图: GET /api/v1/screenshot/{symbol}/{market}/{timeframe}"
echo ""

echo "📊 支持的市场:"
echo "   - us: 美股"
echo "   - hk: 港股"
echo "   - cn: A股"
echo ""

echo "⏰ 支持的时间框架:"
echo "   - 1d: 日线"
echo "   - 1h: 小时线"
echo "   - 1wk: 周线"
echo ""

# 显示常见股票代码
echo "📈 常见股票代码:"
echo "   美股: NVDA, AAPL, TSLA, MSFT, GOOGL, AMZN, META, NFLX"
echo "   港股: 0700, 9988, 9618, 3690, 1810, 0941"
echo "   A股: 000001, 000002, 600036, 600519, 000858, 002415"
echo ""

# 显示注意事项
echo "6. 注意事项:"
echo ""
echo "⚠️  网络配置:"
echo "   - 确保GPT可以访问您的服务地址"
echo "   - 如果使用内网地址，需要配置公网访问"
echo ""
echo "⚠️  服务稳定性:"
echo "   - 确保截图服务24/7运行"
echo "   - 监控服务健康状态"
echo ""
echo "⚠️  性能优化:"
echo "   - 截图服务响应时间通常在5-15秒"
echo "   - 建议配置CDN加速图片访问"
echo ""

# 显示调试命令
echo "7. 调试命令:"
echo ""
echo "🔧 检查服务健康状态:"
echo "   curl https://your-domain.com/health"
echo ""
echo "🔧 测试截图API:"
echo "   curl -X POST https://your-domain.com/api/v1/screenshot \\"
echo "     -H \"Content-Type: application/json\" \\"
echo "     -d '{\"symbol\":\"NVDA\",\"market\":\"us\",\"timeframe\":\"1d\"}'"
echo ""

echo "=== 配置完成 ==="
echo ""
echo "🎉 现在您可以按照上述步骤配置GPT Actions了！"
echo ""
echo "📚 更多详细信息请查看:"
echo "   - GPT_ACTIONS_SETUP.md: 详细配置指南"
echo "   - openapi.yaml: OpenAPI规范文件"
echo "   - gpt-actions-config.json: GPT配置文件"
echo ""
