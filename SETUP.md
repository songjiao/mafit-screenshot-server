# 项目设置说明

## 敏感信息配置

⚠️ **重要安全提醒**：本项目包含敏感配置信息，请务必按照以下步骤进行安全配置。

### 1. 配置文件设置

项目中的 `configs/config.yaml` 文件包含敏感信息（AWS密钥、JWT令牌等），已被添加到 `.gitignore` 中，不会被提交到Git仓库。

### 2. 配置环境变量（推荐方式）

使用环境变量管理敏感信息，这是更安全的方式：

```bash
# 运行环境变量配置脚本
chmod +x setup-env.sh
./setup-env.sh
```

然后编辑 `.env` 文件，填入你的实际配置：

```bash
# AWS配置
AWS_REGION=ap-east-1
AWS_S3_BUCKET=your-actual-bucket-name
AWS_ACCESS_KEY_ID=your-actual-access-key-id
AWS_SECRET_ACCESS_KEY=your-actual-secret-access-key

# CDN配置
CDN_BASE_URL=https://your-actual-cdn-domain.com

# Mafit配置
MAFIT_JWT_TOKEN=your-actual-jwt-token
```

### 3. 传统配置文件方式（备选）

如果你更喜欢使用配置文件，可以：

```bash
cp configs/config.yaml.template configs/config.yaml
```

然后编辑 `configs/config.yaml` 文件，填入你的实际配置。

### 3. 安全注意事项

- ✅ 配置文件 `configs/config.yaml` 已被添加到 `.gitignore`
- ✅ 只有模板文件 `configs/config.yaml.template` 会被提交到Git
- ✅ 请确保不要将真实的密钥信息提交到版本控制系统
- ✅ 建议使用环境变量或密钥管理服务来管理敏感信息

### 4. 环境变量配置（推荐）

你也可以使用环境变量来配置敏感信息，这样更安全：

```bash
export AWS_ACCESS_KEY_ID="your-access-key-id"
export AWS_SECRET_ACCESS_KEY="your-secret-access-key"
export MAFIT_JWT_TOKEN="your-jwt-token"
```

然后在配置文件中使用环境变量：

```yaml
s3:
  access_key_id: "${AWS_ACCESS_KEY_ID}"
  secret_access_key: "${AWS_SECRET_ACCESS_KEY}"

mafit:
  jwt_access_token: "${MAFIT_JWT_TOKEN}"
```

## Git仓库设置

### 初始化Git仓库

```bash
git init
git remote add origin git@github.com:songjiao/mafit-screenshot-server.git
```

### 首次提交

```bash
git add .
git commit -m "Initial commit: Screenshot server with Docker support"
git push -u origin main
```

## 部署说明

### Docker部署

```bash
# 构建并启动
docker compose up -d

# 查看状态
docker compose ps

# 查看日志
docker compose logs -f
```

### 系统服务部署

```bash
# 安装系统服务（需要sudo权限）
sudo ./install-service.sh

# 启动服务
sudo systemctl start screenshot-server.service

# 查看状态
systemctl status screenshot-server.service
```

## 安全检查清单

- [ ] 配置文件已从Git中排除
- [ ] 敏感信息已替换为占位符
- [ ] 环境变量已正确设置（如果使用）
- [ ] 本地配置文件已创建并配置
- [ ] 测试服务正常运行
