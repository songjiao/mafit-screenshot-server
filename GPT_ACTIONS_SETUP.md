# GPT Actions 接入配置指南

## 概述

本指南将帮助您将股票截图服务接入到GPT Actions中，让GPT可以直接调用您的截图API。

## 前置要求

1. **服务已部署**: 确保截图服务已正常运行
2. **网络可访问**: 确保GPT可以访问您的服务地址
3. **OpenAPI规范**: 已提供完整的OpenAPI 3.0规范文件

## 配置步骤

### 1. 准备OpenAPI规范文件

项目根目录已包含 `openapi.yaml` 文件，这是GPT Actions所需的OpenAPI 3.0规范。

### 2. 修改服务器地址

在 `openapi.yaml` 文件中，修改 `servers` 部分：

```yaml
servers:
  - url: https://your-actual-domain.com  # 修改为您的实际服务地址
    description: 生产环境
```

### 3. 在GPT中配置Actions

#### 步骤1: 创建新的GPT
1. 访问 [ChatGPT](https://chat.openai.com/)
2. 点击 "Explore GPTs"
3. 点击 "Create a GPT"

#### 步骤2: 配置Actions
1. 在GPT编辑器中，点击 "Configure"
2. 滚动到 "Actions" 部分
3. 点击 "Add actions"
4. 选择 "Import from URL"
5. 输入您的OpenAPI规范URL：
   ```
   https://your-domain.com/openapi.yaml
   ```
   或者选择 "Import from file" 并上传 `openapi.yaml` 文件

#### 步骤3: 配置GPT指令
在 "Instructions" 中添加以下内容：

```
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
```

### 4. 测试配置

#### 测试健康检查
```
请检查截图服务的健康状态
```

#### 测试截图功能
```
请帮我截取NVDA的日线图
```

#### 测试带数据的截图
```
请获取TSLA的小时线截图和数据
```

## API接口说明

### 主要接口

1. **健康检查**: `GET /health`
   - 检查服务是否正常运行

2. **服务状态**: `GET /api/v1/status`
   - 获取详细的服务状态信息

3. **截图接口**: `POST /api/v1/screenshot`
   - 截取股票K线图并上传到S3

4. **带数据截图**: `POST /api/v1/screenshot-with-data`
   - 截取股票K线图并同时获取JSON数据

5. **GET方式截图**: `GET /api/v1/screenshot/{symbol}/{market}/{timeframe}`
   - 通过URL路径参数截取截图

### 请求参数

```json
{
  "symbol": "NVDA",
  "market": "us",
  "timeframe": "1d"
}
```

### 响应格式

```json
{
  "success": true,
  "message": "Screenshot taken successfully",
  "cdn_url": "https://your-cdn-domain.com/screenshots/NVDA_us_1d_20250108.png",
  "s3_url": "screenshots/NVDA_us_1d_20250108.png",
  "data_cdn_url": "https://your-cdn-domain.com/data/NVDA_us_1d_20250108.json",
  "data_s3_url": "data/NVDA_us_1d_20250108.json",
  "timestamp": "2025-01-08T10:30:00Z"
}
```

## 使用场景

### 1. 股票分析
```
用户: "帮我分析NVDA的技术面，先看看日线图"
GPT: 我来帮您获取NVDA的日线图进行分析...
```

### 2. 市场监控
```
用户: "监控一下TSLA的小时线走势"
GPT: 我来获取TSLA的小时线截图供您分析...
```

### 3. 数据获取
```
用户: "需要AAPL的周线数据和图表"
GPT: 我来获取AAPL的周线截图和JSON数据...
```

## 注意事项

### 1. 网络配置
- 确保GPT可以访问您的服务地址
- 如果使用内网地址，需要配置公网访问

### 2. 服务稳定性
- 确保截图服务24/7运行
- 监控服务健康状态

### 3. 错误处理
- GPT会处理API返回的错误信息
- 建议在服务中添加详细的错误日志

### 4. 性能优化
- 截图服务响应时间通常在5-15秒
- 建议配置CDN加速图片访问

## 故障排除

### 常见问题

1. **GPT无法访问服务**
   - 检查网络连通性
   - 确认服务地址正确
   - 检查防火墙设置

2. **API调用失败**
   - 检查服务日志
   - 验证请求参数格式
   - 确认图表服务可用

3. **截图质量问题**
   - 检查图表服务配置
   - 验证股票代码格式
   - 确认时间框架支持

### 调试命令

```bash
# 检查服务健康状态
curl https://your-domain.com/health

# 测试截图API
curl -X POST https://your-domain.com/api/v1/screenshot \
  -H "Content-Type: application/json" \
  -d '{"symbol":"NVDA","market":"us","timeframe":"1d"}'
```

## 高级配置

### 1. 自定义GPT指令
可以根据需要自定义GPT的行为和回复风格。

### 2. 多环境配置
可以为开发、测试、生产环境配置不同的服务地址。

### 3. 监控集成
可以集成监控系统，实时监控API调用情况。

## 总结

通过以上配置，您的股票截图服务就可以被GPT直接调用了。用户可以通过自然语言与GPT交互，获取股票K线图截图和相关数据，大大提升了用户体验。

记得定期检查服务状态，确保API的稳定性和可用性。
