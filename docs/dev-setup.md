# VoxMesh — 开发环境启动指南

## 概述

VoxMesh 包含 8 个 Go 微服务 + React 前端。启动方式有三种：

| 方式 | 基础设施 | 服务启动 | 优势 |
|------|----------|---------|------|
| **一键本地启动 (推荐)** | 原生 PG + Redis | `make dev-local` | 一条命令，自动检查/启动/健康检测 |
| **Docker** | Docker 容器 | `make up` | 完整环境，端口自动隔离 |
| **手动终端** | 原生安装 | `go run ./cmd/main.go` | 调试方便，可单步启动 |

---

## 方式 A: 一键本地启动 (推荐, 无 Docker)

一条命令启动所有必要服务（Auth + Channel + WS Gateway + 前端），自动检测并启动 PostgreSQL 和 Redis。

```bash
cd ~/projects/VoxMesh
make dev-local
```

脚本自动完成:
1. 清理旧进程
2. 启动 PostgreSQL + Redis（如未运行）
3. 检查/生成 JWT 密钥
4. 启动 auth (8081) → channel (8082) → ws-gateway (8085)
5. 启动 Vite 前端 (5173)
6. 打印本地和 LAN 访问地址

停止所有服务:
```bash
make dev-stop
```

查看各服务日志:
```bash
tail -f logs/*.log
```

---

## 方式 B: Docker 启动 (需要 Docker)

```bash
# 1. 确保 Docker 运行中
docker ps

# 2. 启动全部服务
cd ~/projects/VoxMesh
make up

# 3. 单独启动基础设施（仅 PG + Redis + EMQX + MinIO，不含 Go 服务）
make dev-infra

# 4. 停止并清理
make down       # 停止容器
make clean      # 停止容器 + 删除所有数据卷
```

服务端口映射（Docker 模式）:

| 服务 | 对外端口 | 容器内端口 |
|------|---------|-----------|
| ws-gateway | 8085 | 8080 |
| auth | 8081 | 8080 |
| channel | 8082 | 8080 |
| gateway-coordinator | 8084 | 8080 |
| PostgreSQL | 5432 | 5432 |
| Redis | 6379 | 6379 |
| EMQX MQTT | 1883 | 1883 |
| EMQX Dashboard | 18083 | 18083 |

---

## 方式 C: 手动终端启动 (无 Docker, 调试用)

### 前置条件

#### 1. 安装原生 PostgreSQL 和 Redis

```bash
sudo apt update
sudo apt install -y postgresql redis-server

# 启动服务
sudo service postgresql start
sudo service redis-server start

# 创建数据库和用户
sudo -u postgres psql <<SQL
CREATE USER voxmesh WITH PASSWORD 'voxmesh_dev';
CREATE DATABASE voxmesh OWNER voxmesh;
GRANT ALL PRIVILEGES ON DATABASE voxmesh TO voxmesh;
SQL
```

#### 2. 安装 EMQX (MQTT Broker)

```bash
# 下载并安装 EMQX 5.x
curl -s https://assets.emqx.com/downloads/emqx-5.7.2-ubuntu22.04-amd64.deb -o /tmp/emqx.deb
sudo dpkg -i /tmp/emqx.deb
sudo service emqx start
# Dashboard: http://localhost:18083 (admin/admin123)
```

#### 3. 生成 JWT 密钥

```bash
cd ~/projects/VoxMesh
mkdir -p secrets
openssl genrsa -out secrets/jwt_private.pem 2048
openssl rsa -in secrets/jwt_private.pem -pubout -out secrets/jwt_public.pem
```

#### 4. 运行数据库迁移

```bash
# 安装 golang-migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 执行迁移
migrate -path migrations \
  -database 'postgres://voxmesh:voxmesh_dev@localhost:5432/voxmesh?sslmode=disable' \
  up
```

### 启动服务

所有服务必须从**项目根目录** `~/projects/VoxMesh` 运行（因为密钥路径使用相对路径 `./secrets/`）。

每个服务需要独立终端（或使用 `&` 后台运行）。以下命令假设从项目根目录执行。

#### 终端 1 — Auth Service (端口 8081)

```bash
cd ~/projects/VoxMesh
export DATABASE_URL=postgres://voxmesh:voxmesh_dev@localhost:5432/voxmesh?sslmode=disable
export REDIS_URL=redis://localhost:6379/0
export JWT_PRIVATE_KEY=./secrets/jwt_private.pem
export JWT_PUBLIC_KEY=./secrets/jwt_public.pem
export HTTP_PORT=8081
CORS_ORIGINS=http://<LAN_IP>:5173 go run ./services/auth/cmd/main.go
```

#### 终端 2 — Channel Service (端口 8082)

```bash
cd ~/projects/VoxMesh
export DATABASE_URL=postgres://voxmesh:voxmesh_dev@localhost:5432/voxmesh?sslmode=disable
export REDIS_URL=redis://localhost:6379/1
export MQTT_BROKER=tcp://localhost:1883
export MQTT_CLIENT_ID=svc_channel
export JWT_PRIVATE_KEY=./secrets/jwt_private.pem
export JWT_PUBLIC_KEY=./secrets/jwt_public.pem
export HTTP_PORT=8082
CORS_ORIGINS=http://<LAN_IP>:5173 go run ./services/channel/cmd/main.go
```

#### 终端 3 — WS Gateway (端口 8085 — 前端入口)

```bash
cd ~/projects/VoxMesh
export REDIS_URL=redis://localhost:6379/5
export AUTH_SERVICE_URL=http://localhost:8081
export CHANNEL_SERVICE_URL=http://localhost:8082
export JWT_PRIVATE_KEY=./secrets/jwt_private.pem
export JWT_PUBLIC_KEY=./secrets/jwt_public.pem
export HTTP_PORT=8085
CORS_ORIGINS=http://<LAN_IP>:5173 go run ./services/ws-gateway/cmd/main.go
```

#### 终端 4 — React 前端 (端口 5173)

```bash
cd ~/projects/VoxMesh/web-client
npm install   # 仅首次
npm run dev   # 打开 http://localhost:5173
```

#### 可选服务

以下服务在基础测试中不需要：

```bash
# Gateway Coordinator (端口 8084)
export DATABASE_URL=postgres://voxmesh:voxmesh_dev@localhost:5432/voxmesh?sslmode=disable
export REDIS_URL=redis://localhost:6379/4
export MQTT_BROKER=tcp://localhost:1883
export MQTT_CLIENT_ID=svc_gateway_coord
export HTTP_PORT=8084
cd services/gateway-coordinator && go run ./cmd/main.go

# Presence Manager (无 HTTP)
export REDIS_URL=redis://localhost:6379/3
export MQTT_BROKER=tcp://localhost:1883
export MQTT_CLIENT_ID=svc_presence
cd services/presence && go run ./cmd/main.go

# Audio Mixer (gRPC 端口 9000)
export REDIS_URL=redis://localhost:6379/2
export AUDIO_MIXER_ADDR=localhost:9000
export MQTT_BROKER=tcp://localhost:1883
export MQTT_CLIENT_ID=svc_audio_mixer
cd services/audio-mixer && go run ./cmd/main.go

# Command Handler (仅 MQTT)
export MQTT_BROKER=tcp://localhost:1883
export MQTT_CLIENT_ID=svc_command
cd services/command-handler && go run ./cmd/main.go

# Notification Service (仅 MQTT)
export MQTT_BROKER=tcp://localhost:1883
export MQTT_CLIENT_ID=svc_notification
cd services/notification && go run ./cmd/main.go
```

### 依赖关系矩阵

| 服务 | PostgreSQL | Redis | MQTT | 缺失行为 |
|------|-----------|-------|------|---------|
| auth | **必需** | **必需** | 不需要 | log.Fatalf 退出 |
| channel | **必需** | **必需** | 不需要 | log.Fatalf 退出 |
| ws-gateway | 不需要 | **必需** | 不需要 | log.Fatalf 退出 |
| gateway-coordinator | **必需** | **必需** | **必需** | log.Fatalf 退出 |
| presence | 不需要 | **必需** | **必需** | log.Fatalf 退出 |
| audio-mixer | 不需要 | **必需** | 可选 | Redis 缺失则 Fatal；MQTT 缺失则 Warning |
| command-handler | 不需要 | 不需要 | **必需** | log.Fatalf 退出 |
| notification | 不需要 | 不需要 | **必需** | log.Fatalf 退出 |

---

## LAN 外网访问配置

其他设备（手机、平板）通过局域网 IP 访问开发中的 VoxMesh 前端进行测试。

### WSL2 镜像网络模式 (推荐, Windows 11 22H2+)

创建 `%USERPROFILE%\.wslconfig`:
```ini
[wsl2]
networkingMode=mirrored
```
然后 `wsl --shutdown` 重启 WSL。之后 WSL2 直接共享 Windows IP，无需端口转发。

### WSL2 NAT + 端口转发 (任意 Windows)

```powershell
# 查找 IP
ipconfig | findstr IPv4    # Windows IP (如 10.23.183.132)
wsl -d Ubuntu-22.04 -- bash -c "hostname -I"  # WSL2 IP (如 172.26.114.192)

# 端口转发 (管理员 PowerShell)
netsh interface portproxy add v4tov4 listenaddress=<WINDOWS_IP> listenport=8085 connectaddress=<WSL2_IP> connectport=8085
netsh interface portproxy add v4tov4 listenaddress=<WINDOWS_IP> listenport=5173 connectaddress=<WSL2_IP> connectport=5173

# 防火墙 (管理员 PowerShell)
New-NetFirewallRule -DisplayName "VoxMesh-8085" -Direction Inbound -LocalPort 8085 -Protocol TCP -Action Allow
New-NetFirewallRule -DisplayName "VoxMesh-5173" -Direction Inbound -LocalPort 5173 -Protocol TCP -Action Allow
```

### 启动服务 (带 CORS)

```bash
make dev-local  # 脚本自动在 CORS_ORIGINS 中追加 LAN IP
```

外网设备访问 `http://<WINDOWS_IP>:5173`。

### 移动端注意事项

- **麦克风**: 手机浏览器需要 HTTPS 才能使用 `getUserMedia`。本地测试可开启 Chrome `chrome://flags/#unsafely-treat-insecure-origin-as-secure` 添加 `http://<WINDOWS_IP>:8085`
- **crypto.randomUUID**: 旧手机浏览器 (iOS <15) 不支持，前端已内置 polyfill (`src/lib/uuid.ts`)
- **AudioWorklet**: 需要 iOS 16.4+。旧版 iOS 将无法使用音频降噪功能
- **注册账号**: 手机和桌面需要用**不同账号**登录 (同一账号多设备已支持，但测试建议分开)

---

### 验证清单

#### 基础设施
- [ ] PostgreSQL 运行中: `pg_isready`
- [ ] Redis 运行中: `redis-cli ping` → PONG
- [ ] JWT 密钥已生成: `ls secrets/jwt_private.pem secrets/jwt_public.pem`
- [ ] 数据库迁移已执行: 表 users/channels/gateways 已创建

#### 服务
- [ ] Auth 服务可访问: `curl http://localhost:8081/health`
- [ ] Channel 服务可访问: `curl http://localhost:8082/health`
- [ ] WS Gateway 可访问: `curl http://localhost:8085/health`
- [ ] 前端加载正常: `http://localhost:5173`

#### 通信
- [ ] WebSocket 连接: 控制台 `[ws] open` (仅 1 次，无重连风暴)
- [ ] 注册/登录: API 返回 JWT token pair
- [ ] Token 刷新: 过期前 5 分钟自动刷新
- [ ] 频道创建/加入: 频道列表正常显示
- [ ] 多人文字聊天: 消息双方可见

#### 音频
- [ ] 麦克风授权: 弹窗允许后 `[audio] getUserMedia OK`
- [ ] RNNoise 加载: `[audio] RNNoise worklet loaded`
- [ ] WebRTC ICE 连接: `[webrtc] ICE <user>: connected`
- [ ] 远端音频: `[webrtc] remote track from user=<id>`
- [ ] 双向通话清晰无失真
- [ ] 静音/开麦正常
- [ ] 噪声门有效 (不说话时静音)

#### 多设备/外网
- [ ] 手机可访问 `http://<WINDOWS_IP>:5173`
- [ ] 手机-桌面同一频道文字互通
- [ ] 手机-桌面同一频道音频互通
- [ ] CORS 不拦截跨来源请求
