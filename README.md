# 股票截图服务

一个专门提供股票K线图截图的RESTful API服务。

## 功能特性

- 🚀 **高性能截图**: 使用本地图表服务截取股票K线图
- 📊 **多市场支持**: 支持美股(US)、港股(HK)、A股(CN)
- ⏰ **智能时间处理**: 根据市场开市时间自动调整截图策略
- 🔄 **去重机制**: 避免重复截图，提高效率
- ☁️ **云存储集成**: 自动上传到S3并返回CDN URL
- 💾 **内存优化**: 使用本地图表服务，大幅减少内存占用
- 🎯 **轻量级**: 适合低并发场景，如API调用

## 快速开始

### 1. 前置要求

本服务依赖于mafit的本地内置图表服务器，请确保：

1. **mafit已安装并运行**
2. **本地图表服务可用**: 默认地址为 `http://127.0.0.1:4009`
3. **网络连通性**: 确保服务能够访问图表服务器

### 2. 配置

编辑 `configs/config.yaml` 文件，配置以下内容：

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

s3:
  region: "ap-east-1"
  bucket: "your-bucket"
  access_key_id: "your-access-key"
  secret_access_key: "your-secret-key"

cdn:
  base_url: "https://your-cdn-domain.com"

chart_service:
  base_url: "http://192.168.1.76:4009"
```

## 部署方式

### 方式一：Docker部署（推荐）

1. **构建并启动服务**
   ```bash
   docker compose up -d
   ```

2. **查看服务状态**
   ```bash
   docker compose ps
   ```

3. **查看日志**
   ```bash
   docker compose logs -f
   ```

4. **停止服务**
   ```bash
   docker compose down
   ```

#### 配置开机自启动

为了确保服务器重启后服务能自动启动，可以安装systemd服务：

```bash
# 安装系统服务（需要sudo权限）
sudo ./install-service.sh

# 手动启动服务
sudo systemctl start screenshot-server.service

# 查看服务状态
systemctl status screenshot-server.service

# 查看服务日志
journalctl -u screenshot-server.service -f
```

### 方式二：传统部署

1. **安装依赖**
   ```bash
   go mod download
   ```

2. **启动服务**
   ```bash
   go run cmd/screenshot-server/main.go
   ```

   或者构建后运行：
   ```bash
   make build
   ./screenshot-server
   ```

服务将在 `http://localhost:8080` 启动。

## API 使用

### 健康检查

```bash
curl http://localhost:8080/health
```

### 截图API

#### POST 方式

```bash
curl -X POST http://localhost:8080/api/v1/screenshot \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "NVDA",
    "market": "us",
    "timeframe": "1d"
  }'
```

#### GET 方式

```bash
curl http://localhost:8080/api/v1/screenshot/NVDA/us/1d
```

### 带数据的截图API（推荐）

这个API会在截图完成后自动下载JSON数据文件并上传到S3。

#### POST 方式

```bash
curl -X POST http://localhost:8080/api/v1/screenshot-with-data \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "NVDA",
    "market": "us",
    "timeframe": "1d"
  }'
```

#### GET 方式

```bash
curl http://localhost:8080/api/v1/screenshot-with-data/NVDA/us/1d
```

### 响应格式

#### 普通截图响应

```json
{
  "success": true,
  "message": "Screenshot taken successfully",
  "cdn_url": "https://your-cdn-domain.com/screenshots/NVDA_us_1d_20250729.png",
  "s3_url": "screenshot/screenshots/NVDA_us_1d_20250729.png",
  "timestamp": "2025-07-29T10:46:22+08:00"
}
```

#### 带数据的截图响应

```json
{
  "success": true,
  "message": "Screenshot with data taken successfully",
  "cdn_url": "https://your-cdn-domain.com/screenshots/NVDA_us_1d_20250729.png",
  "s3_url": "screenshot/screenshots/NVDA_us_1d_20250729.png",
  "data_cdn_url": "https://your-cdn-domain.com/data/NVDA_us_1d_20250729.json",
  "data_s3_url": "screenshot/data/NVDA_us_1d_20250729.json",
  "timestamp": "2025-07-29T10:46:22+08:00"
}
```

## 参数说明

- `symbol`: 股票代码 (如: NVDA, AAPL, TSLA)
- `market`: 市场代码 (us: 美股, hk: 港股, cn: A股)
- `timeframe`: 时间框架 (1d: 日线, 1h: 小时线)

## 项目结构

```
├── cmd/
│   └── screenshot-server/    # 主程序
├── internal/
│   ├── browser/             # 浏览器管理
│   ├── config/              # 配置管理
│   ├── screenshot/          # 截图服务
│   └── s3/                  # S3客户端
├── configs/
│   └── config.yaml          # 配置文件
├── web/                     # 前端文件
├── Dockerfile               # Docker构建文件
├── docker-compose.yml       # Docker Compose配置
└── Makefile                 # 构建脚本
```

## 注意事项

1. 首次截图可能需要较长时间，因为需要启动浏览器
2. 建议在生产环境中使用HTTPS
3. 服务采用单例浏览器模式，适合低并发场景
4. 定期检查S3存储使用情况和CDN缓存状态
5. 建议定期重启服务以释放内存（如每天一次）

## 许可证

MIT License 