# Sidelight Docker 部署指南

本项目提供了完整的 Docker 化解决方案，包含 sidelight 和 rawtherapee-cli，支持 Linux 环境。

## 快速开始

### 1. 使用预构建镜像

```bash
# 拉取最新镜像
docker pull registry.cn-shanghai.aliyuncs.com/linran-pub/sidelight:latest

# 运行 web server 模式（配置文件方式）
docker run -d \
  --name sidelight \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config:/app/config \
  registry.cn-shanghai.aliyuncs.com/linran-pub/sidelight:latest \
  server --port 8080

# 运行 CLI 模式处理单张图片（环境变量方式）
docker run --rm \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config:/app/config \
  -e SL_GEMINI_API_KEY="your_api_key_here" \
  registry.cn-shanghai.aliyuncs.com/linran-pub/sidelight:latest \
  grade /app/data/input.jpg --format xmp
```

### 2. 使用 Docker Compose（推荐配置文件方式）

```bash
# 创建必要的目录
mkdir -p data config

# 创建配置文件（推荐方式）
cat > config/config.json << EOF
{
  "gemini_api_key": "your_gemini_api_key_here",
  "gemini_endpoint_url": "",
  "gemini_model_name": "gemini-pro-vision"
}
EOF

# 启动 web server
docker-compose up -d

# 运行 CLI 命令
docker-compose run --rm sidelight-cli grade /app/data/input.jpg --format pp3

# 查看帮助
docker-compose run --rm sidelight-cli --help
```

**或者使用环境变量方式：**

```bash
# 设置环境变量（备用方式）
export SL_GEMINI_API_KEY="your_api_key_here"
docker-compose up -d
```

### 3. 本地构建

```bash
# 构建镜像
docker build -t sidelight:local .

# 运行本地构建的镜像
docker run --rm sidelight:local --help
```

## 配置方式

### 1. 配置文件（推荐）

创建 `config/config.json` 文件：

```json
{
  "gemini_api_key": "your_gemini_api_key_here",
  "gemini_endpoint_url": "",
  "gemini_model_name": "gemini-pro-vision"
}
```

### 2. 环境变量（备用）

| 变量名 | 描述 | 默认值 |
|--------|------|--------|
| `RT_CLI_PATH` | RawTherapee CLI 路径（已预设） | `/usr/bin/rawtherapee-cli` |
| `SL_GEMINI_API_KEY` | Gemini API 密钥 | - |
| `SL_GEMINI_ENDPOINT_URL` | Gemini API 端点 | - |
| `SL_GEMINI_MODEL_NAME` | Gemini 模型名称 | `gemini-pro-vision` |

**配置优先级：** 命令行参数 > 环境变量 > 配置文件 > 默认值

## 数据卷挂载

- `/app/data` - 图片处理目录，放入待处理的RAW文件和JPG文件
- `/app/config` - 配置文件目录，可放入 config.json

## 支持的命令

### Grade 命令（AI色彩分级）

```bash
docker run --rm \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config:/app/config \
  registry.cn-shanghai.aliyuncs.com/linran-pub/sidelight:latest \
  grade /app/data/*.jpg --style cinematic --format xmp,pp3
```

### Frame 命令（艺术相框）

```bash
docker run --rm \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config:/app/config \
  registry.cn-shanghai.aliyuncs.com/linran-pub/sidelight:latest \
  frame /app/data/*.jpg --style vintage --output /app/data/output
```

### Server 命令（Web UI）

```bash
docker run -d \
  --name sidelight-server \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config:/app/config \
  registry.cn-shanghai.aliyuncs.com/linran-pub/sidelight:latest \
  server --port 8080
```

访问 http://localhost:8080 使用 Web 界面。

## 功能特性

- ✅ 支持多种 RAW 格式（ARW, CR3, NEF）
- ✅ 支持标准格式（JPG, PNG）
- ✅ 集成 exiftool 用于元数据提取
- ✅ 集成 rawtherapee-cli 用于 PP3 处理
- ✅ Adobe Lightroom XMP 输出
- ✅ RawTherapee PP3 原生输出
- ✅ Web UI 交互界面
- ✅ 非破坏性工作流程

## 部署到生产环境

### 使用 Docker Swarm

```yaml
# swarm-stack.yml
version: '3.8'

services:
  sidelight:
    image: registry.cn-shanghai.aliyuncs.com/linran-pub/sidelight:latest
    ports:
      - "8080:8080"
    environment:
      - SL_GEMINI_API_KEY_FILE=/run/secrets/gemini_api_key
    secrets:
      - gemini_api_key
    volumes:
      - sidelight_data:/app/data
    deploy:
      replicas: 2
      restart_policy:
        condition: on-failure

secrets:
  gemini_api_key:
    external: true

volumes:
  sidelight_data:
```

部署：

```bash
# 创建密钥
echo "your_api_key" | docker secret create gemini_api_key -

# 部署 stack
docker stack deploy -c swarm-stack.yml sidelight
```

### 使用 Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sidelight
spec:
  replicas: 2
  selector:
    matchLabels:
      app: sidelight
  template:
    metadata:
      labels:
        app: sidelight
    spec:
      containers:
      - name: sidelight
        image: registry.cn-shanghai.aliyuncs.com/linran-pub/sidelight:latest
        ports:
        - containerPort: 8080
        env:
        - name: SL_GEMINI_API_KEY
          valueFrom:
            secretKeyRef:
              name: sidelight-secrets
              key: gemini-api-key
        volumeMounts:
        - name: data-volume
          mountPath: /app/data
      volumes:
      - name: data-volume
        persistentVolumeClaim:
          claimName: sidelight-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: sidelight-service
spec:
  selector:
    app: sidelight
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

## 故障排查

### 常见问题

1. **API 密钥错误**
   ```bash
   # 检查环境变量
   docker run --rm registry.cn-shanghai.aliyuncs.com/linran-pub/sidelight:latest env | grep SL_
   ```

2. **文件权限问题**
   ```bash
   # 确保挂载目录有正确的权限
   sudo chown -R 1000:1000 ./data
   ```

3. **内存不足**
   ```bash
   # 增加 Docker 内存限制
   docker run --memory="2g" ...
   ```

4. **日志查看**
   ```bash
   # 查看容器日志
   docker logs sidelight

   # 实时查看日志
   docker logs -f sidelight
   ```

## CI/CD 流程

项目配置了 GitHub Actions 自动构建和推送：

- 推送到 `main` 或 `master` 分支时自动触发构建
- 支持多架构构建（linux/amd64, linux/arm64）
- 自动推送到阿里云容器镜像服务
- 支持手动触发构建

### 配置密钥

在 GitHub 仓库的 Settings > Secrets 中添加：

- `ACR_USERNAME` - 阿里云容器镜像服务用户名
- `ACR_PASSWORD` - 阿里云容器镜像服务密码

镜像标签：
- `latest` - 最新版本
- `${{ github.sha }}` - 特定提交的版本