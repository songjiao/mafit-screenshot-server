# 生产环境部署指南

## 🚀 Docker环境变量配置

### 方式一：使用.env文件（推荐）

1. **创建环境变量文件**：
   ```bash
   cp env.template .env
   ```

2. **编辑.env文件**：
   ```bash
   nano .env
   ```

3. **填入生产环境配置**：
   ```bash
   # AWS配置
   AWS_REGION=ap-east-1
   AWS_S3_BUCKET=your-production-bucket
   AWS_ACCESS_KEY_ID=your-production-access-key
   AWS_SECRET_ACCESS_KEY=your-production-secret-key

   # CDN配置
   CDN_BASE_URL=https://your-production-cdn.com

   # Mafit配置
   MAFIT_JWT_TOKEN=your-production-jwt-token
   ```

### 方式二：直接在docker-compose.yml中配置

```yaml
services:
  screenshot-server:
    environment:
      - AWS_ACCESS_KEY_ID=your-production-access-key
      - AWS_SECRET_ACCESS_KEY=your-production-secret-key
      - AWS_S3_BUCKET=your-production-bucket
      - CDN_BASE_URL=https://your-production-cdn.com
      - MAFIT_JWT_TOKEN=your-production-jwt-token
```

### 方式三：使用Docker Secrets（高安全性）

1. **创建secrets**：
   ```bash
   echo "your-production-access-key" | docker secret create aws_access_key_id -
   echo "your-production-secret-key" | docker secret create aws_secret_access_key -
   echo "your-production-jwt-token" | docker secret create mafit_jwt_token -
   ```

2. **在docker-compose.yml中使用**：
   ```yaml
   services:
     screenshot-server:
       secrets:
         - aws_access_key_id
         - aws_secret_access_key
         - mafit_jwt_token
       environment:
         - AWS_ACCESS_KEY_ID_FILE=/run/secrets/aws_access_key_id
         - AWS_SECRET_ACCESS_KEY_FILE=/run/secrets/aws_secret_access_key
         - MAFIT_JWT_TOKEN_FILE=/run/secrets/mafit_jwt_token
   ```

### 方式四：使用Docker环境变量文件

1. **创建环境变量文件**：
   ```bash
   # production.env
   AWS_ACCESS_KEY_ID=your-production-access-key
   AWS_SECRET_ACCESS_KEY=your-production-secret-key
   AWS_S3_BUCKET=your-production-bucket
   CDN_BASE_URL=https://your-production-cdn.com
   MAFIT_JWT_TOKEN=your-production-jwt-token
   ```

2. **在docker-compose.yml中引用**：
   ```yaml
   services:
     screenshot-server:
       env_file:
         - production.env
   ```

## 🔐 生产环境安全最佳实践

### 1. 密钥管理
- ✅ 使用IAM角色而不是访问密钥（如果可能）
- ✅ 定期轮换密钥
- ✅ 使用最小权限原则
- ✅ 启用CloudTrail监控

### 2. 网络安全
```yaml
services:
  screenshot-server:
    networks:
      - internal-network
    ports:
      - "127.0.0.1:8080:8080"  # 只允许本地访问
```

### 3. 资源限制
```yaml
services:
  screenshot-server:
    deploy:
      resources:
        limits:
          memory: 1G
          cpus: '0.5'
        reservations:
          memory: 512M
          cpus: '0.25'
```

### 4. 健康检查和重启策略
```yaml
services:
  screenshot-server:
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

## 🐳 Docker Swarm部署

### 1. 初始化Swarm
```bash
docker swarm init
```

### 2. 创建secrets
```bash
echo "your-production-access-key" | docker secret create aws_access_key_id -
echo "your-production-secret-key" | docker secret create aws_secret_access_key -
echo "your-production-jwt-token" | docker secret create mafit_jwt_token -
```

### 3. 部署服务
```bash
docker stack deploy -c docker-compose.yml screenshot-stack
```

## ☁️ 云平台部署

### AWS ECS部署
```yaml
# task-definition.json
{
  "family": "screenshot-server",
  "containerDefinitions": [
    {
      "name": "screenshot-server",
      "image": "your-registry/screenshot-server:latest",
      "environment": [
        {
          "name": "AWS_ACCESS_KEY_ID",
          "value": "your-access-key"
        },
        {
          "name": "AWS_SECRET_ACCESS_KEY",
          "value": "your-secret-key"
        }
      ],
      "secrets": [
        {
          "name": "MAFIT_JWT_TOKEN",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:mafit-jwt-token"
        }
      ]
    }
  ]
}
```

### Kubernetes部署
```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: screenshot-server
spec:
  replicas: 2
  selector:
    matchLabels:
      app: screenshot-server
  template:
    metadata:
      labels:
        app: screenshot-server
    spec:
      containers:
      - name: screenshot-server
        image: your-registry/screenshot-server:latest
        env:
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: aws-secrets
              key: access-key-id
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: aws-secrets
              key: secret-access-key
        - name: MAFIT_JWT_TOKEN
          valueFrom:
            secretKeyRef:
              name: mafit-secrets
              key: jwt-token
```

## 📊 监控和日志

### 1. 日志配置
```yaml
services:
  screenshot-server:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### 2. 监控指标
- 服务响应时间
- 错误率
- 内存和CPU使用率
- 截图成功率

## 🔄 部署流程

### 1. 构建镜像
```bash
docker build -t your-registry/screenshot-server:latest .
docker push your-registry/screenshot-server:latest
```

### 2. 更新环境变量
```bash
# 编辑.env文件或更新secrets
nano .env
```

### 3. 部署服务
```bash
docker compose down
docker compose up -d
```

### 4. 验证部署
```bash
# 检查服务状态
docker compose ps

# 测试健康检查
curl http://localhost:8080/health

# 查看日志
docker compose logs -f
```
