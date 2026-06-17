# VoxMesh — 服务端架构设计文档

> 版本: 2.0.0 | 日期: 2026-06-14 | 状态: 已实施 (Phase 12 WebRTC P2P)

## 目录
1. [项目概述与评估](#1-项目概述与评估)
2. [开发环境与工具链](#2-开发环境与工具链)
3. [技术栈选择与理由](#3-技术栈选择与理由)
4. [架构总览](#4-架构总览)
5. [服务模块分解](#5-服务模块分解)
6. [MQTT 主题层级设计](#6-mqtt-主题层级设计)
7. [WebSocket 协议设计](#7-websocket-协议设计)
8. [REST API 接口设计](#8-rest-api-接口设计)
9. [数据库设计](#9-数据库设计)
10. [网关协调协议](#10-网关协调协议)
11. [音频路由逻辑](#11-音频路由逻辑)
12. [安全设计](#12-安全设计)
13. [部署架构](#13-部署架构)
14. [嵌入式设备接口协议](#14-嵌入式设备接口协议)
15. [关键架构决策](#15-关键架构决策)
16. [当前实施状态](#16-当前实施状态)
17. [功能模块验证清单](#17-功能模块验证清单)
18. [未来可扩展点](#18-未来可扩展点)

---

## 1. 项目概述与评估

### 1.1 项目定位

VoxMesh 是一个跨设备（网页端 + 嵌入式终端）的语音频道通信系统，核心能力：

- **本地离线组网**：ESP-NOW 多跳 Mesh 实现无公网环境下的区域内对讲
- **云端跨地域互通**：通过 MQTT 服务器桥接不同地域的本地网络
- **多客户端接入**：Web 浏览器端 + ESP32-S3 嵌入式终端
- **语音智能处理**：终端侧实时降噪 + 离线命令词识别
- **网关高可用**：多网关自动故障切换

### 1.2 项目复杂度评估

| 维度 | 评估 | 说明 |
|---|---|---|
| **服务端架构** | 中高 | 多服务微服务架构，MQTT + WebSocket 双协议栈 |
| **实时性要求** | 高 | 音频流延迟需控制在 150ms 以内（端到端） |
| **并发规模** | 中等 | 单频道 50-100 人，总并发 1000-5000 设备 |
| **嵌入式集成** | 高 | 需要与 ESP-IDF 工具链配合，MQTT 协议精确对齐 |
| **前端复杂度** | 中高 | WebRTC P2P mesh + RNNoise WASM 降噪，响应式布局 |

### 1.3 功能模块总览

```
用户系统: 注册/登录, JWT 认证, 角色权限, 设备绑定, API 密钥
频道管理: 频道 CRUD, 层级结构, 权限控制, 临时频道, 密码保护
音频引擎: getUserMedia → RNNoise WASM 降噪 → WebRTC Opus 编码 → P2P SRTP 传输
网关管理: 网关注册, 心跳监控, 故障切换, 拓扑管理, 设备分配
实时通信: WebSocket (信令+文字), MQTT (嵌入式音频), 在线状态, 说话指示
语音智能: RNNoise 神经网络降噪(Web端), 本地降噪+命令词识别(嵌入式终端)
系统管理: 健康检查, 监控告警, 日志聚合, 版本管理
数据持久化: 用户存储, 频道存储, 状态缓存, 消息历史
```

---

## 2. 开发环境与工具链

### 2.1 整体开发环境

```
Windows 11 Home (宿主机)
├── WSL2 (Ubuntu 24.04 LTS) ← 主要开发环境
│   ├── Go 1.26.4 toolchain (go mod, go build)
│   ├── Node.js 22 LTS (前端, npm)
│   ├── PostgreSQL 16 + Redis 7 (原生安装)
│   ├── Docker Desktop (WSL2 backend, 可选)
│   └── Git (版本控制)
├── ESP-IDF v5.2+ (Windows 原生 / VS Code)
└── IDE: VS Code / GoLand
```

### 2.2 IDE 选择

| IDE | 用途 | 理由 |
|---|---|---|
| **GoLand / VS Code** | 服务端 Go 开发 | Go 支持、Docker 集成、DB 工具 |
| **VS Code** | ESP32 固件开发 | ESP-IDF 官方插件、串口监视器 |
| **VS Code** | React 前端开发 | TypeScript 支持、调试工具 |

### 2.3 开发工作流

```bash
# 一键启动 (基础设施 + 后端 + 前端)
make dev-local

# 停止
make dev-stop

# 单独启动基础设施 (Docker)
make dev-infra

# 数据库迁移
make migrate-up

# 测试
make test              # 单元测试
make test-integration  # 集成测试

# 构建
make build
```

### 2.4 开发辅助工具

| 工具 | 用途 |
|---|---|
| MQTTX | MQTT 客户端调试 (GUI) |
| Postman / Bruno | REST API 测试 |
| ffmpeg | 模拟音频流生成和验证 |
| websocat | WebSocket 命令行测试 |

---

## 3. 技术栈选择与理由

### 3.1 服务端语言: Go 1.26.4

| 对比项 | Go | Node.js | Rust |
|---|---|---|---|
| 并发模型 | goroutine (轻量) | Event Loop | async/await |
| 容器镜像 | ~10MB | ~150MB | ~5MB |
| MQTT 客户端 | paho.golang (成熟) | mqtt.js | rumqtt |
| 内存占用 | 极低 | 中 | 极低 |

**选择 Go 的核心原因**:
1. goroutine 天然适合音频流处理
2. paho.golang 完整支持 MQTT 5.0
3. 低延迟 GC
4. 单二进制部署 (~10MB)
5. 标准库强大

### 3.2 MQTT Broker: EMQX 5.x

- MQTT 5.0 完整支持 (User Properties 用于音频元数据)
- PostgreSQL ACL 插件 → 动态权限管理
- 共享订阅 → Audio Mixer 可水平扩展
- 内置 WebSocket → 备用浏览器连接方案
- Dashboard: `http://localhost:18083`

### 3.3 数据库: PostgreSQL 16 + Redis 7

**PostgreSQL** — 持久化: 用户、频道、网关、设备、消息历史、JSONB 灵活配置

**Redis** — 热数据: 在线状态 TTL、网关心跳超时、速率限制 Token Bucket、临时频道自动过期

### 3.4 前端: React 19 + TypeScript + Vite

- **WebRTC P2P Mesh**: RTCPeerConnection 浏览器引擎 Opus 编解码 + SRTP P2P 传输
- **RNNoise WASM**: `@timephy/rnnoise-wasm` (Jitsi Meet 同款) — ML 神经网络降噪
- **Zustand**: 轻量状态管理，适合高频音频状态更新
- **Tailwind CSS 4**: 响应式布局（桌面三栏 / 手机汉堡菜单侧滑）
- **AudioWorklet**: 音频处理管线在独立线程运行，不阻塞 UI

### 3.5 音频编解码: Opus (全系统唯一编解码器)

| 平台 | 实现 |
|---|---|
| Web 浏览器 | WebRTC 内置 Opus (RTCPeerConnection) — 自动 FEC/PLC/自适应码率 |
| 嵌入式 ESP32 | ESP-IDF libopus 组件 — 20ms 帧, 16kHz |
| 服务端 Audio Mixer | Go cgo 调用 libopus — 混音/转码 (嵌入式桥接用) |

Web 端**不**使用 WebAssembly libopus — WebRTC 引擎已内置完整的 Opus 编解码，效果优于手动实现。

### 3.6 安全: JWT (RS256) + TLS

- RS256 非对称签名: 公钥可分发给网关离线验证
- TLS 全链路: Web (WSS 443), MQTT (8883), API (HTTPS 443)
- MQTT ACL: EMQX PostgreSQL ACL 插件

---

## 4. 架构总览

### 4.1 系统拓扑

```
                         ┌──────────────────────────────────────┐
                         │          INTERNET / CLOUD            │
  ┌──────────┐    WSS    │  ┌──────────┐      ┌──────────┐     │
  │  Web     │◄─────────►│  │  Nginx   │      │  EMQX 5  │     │
  │  Client  │  (REST+WS) │  │(TLS+Proxy)│      │  Broker  │     │
  │ (React)  │  +SRTP P2P │  └────┬─────┘      └────┬─────┘     │
  └──────────┘           │       │                  │          │
        │ SRTP (P2P)     │  ┌────▼──────────────────▼──────┐   │
        └───────────────►│  │       VoxMesh Core           │   │
                         │  │  Auth  │Channel│Audio Mixer  │   │
                         │  │  Gateway Coord │Presence Mgr │   │
                         │  │  WS Gateway (信令中继)       │   │
                         │  └────────────┬────────────────┘   │
                         │               │                     │
                         │  ┌────────────▼────────────┐       │
                         │  │  PostgreSQL │ Redis     │       │
                         │  └─────────────────────────┘       │
                         └──────────────────────────────────────┘
                                          │ MQTT/TLS :8883
              ┌───────────────────────────┼───────────────────┐
              │     LOCAL SITE / ESP-NOW MESH                 │
  ┌───────────▼──────────┐    ┌─────────▼─────────┐          │
  │  Gateway #1          │    │  Gateway #2       │ (N冗余)  │
  │  MQTT + ESP-NOW      │    │  MQTT + ESP-NOW   │          │
  └──────────┬───────────┘    └────────┬──────────┘          │
             │ ESP-NOW                 │ ESP-NOW              │
  ┌──────────▼──────────┐    ┌─────────▼─────────┐          │
  │ ESP32 Terminal #1   │    │ ESP32 Terminal #N │          │
  │ Mic + Speaker       │    │ (纯Mesh, 无WiFi)  │          │
  └─────────────────────┘    └───────────────────┘          │
  └──────────────────────────────────────────────────────────┘
```

### 4.2 数据流总结

| 通信场景 | 数据路径 |
|---|---|
| 嵌入式↔嵌入式 (同Mesh) | ESP-NOW 直连/中继, 不经过服务器 |
| 嵌入式→Web | ESP32→ESP-NOW→网关→MQTT→Audio Mixer→WebSocket→Web |
| Web→嵌入式 | Web→WebSocket→Audio Mixer→MQTT→网关→ESP-NOW→ESP32 |
| Web↔Web (同频道) | **WebRTC P2P SRTP 直连** (不经服务器) + WebSocket 信令中继 |
| Web↔Web (跨网段) | WebRTC STUN/TURN → P2P SRTP |
| 语音指令 | ESP32 本地 ASR→MQTT transcript→Command Handler→执行 |
| 文字聊天 | Web→WebSocket→ws-gateway→BroadcastToChannel |

---

## 5. 服务模块分解

### 5.1 Auth Service
**端口**: 8081
**职责**: 注册/登录, JWT (RS256) 签发/验证, API Key 管理
**接口**: REST `/api/v1/auth/*` (通过 ws-gateway 代理)
**依赖**: PostgreSQL, Redis
**CORS**: 支持 `CORS_ORIGINS` 环境变量追加允许来源

### 5.2 Channel Service
**端口**: 8082
**职责**: 频道 CRUD (层级结构), 成员管理, 权限检查
**接口**: REST `/api/v1/channels/*` (通过 ws-gateway 代理)
**依赖**: PostgreSQL, Redis
**CORS**: 支持 `CORS_ORIGINS` 环境变量追加允许来源

### 5.3 Audio Mixer
**端口**: 9000 (gRPC)
**职责**: 多流 Opus 混音 (N-1), 抖动缓冲, VAD, 增益归一化
**接口**: MQTT Subscribe (共享订阅), gRPC stream
**依赖**: MQTT, libopus (cgo)
**当前状态**: 嵌入式路径已实现; Web P2P 音频绕行此服务

### 5.4 Presence / State Manager
**职责**: 在线追踪, 频道成员快照, 断连清理
**接口**: MQTT Subscribe/Publish presence topics
**依赖**: Redis (TTL), MQTT

### 5.5 Gateway Coordinator
**端口**: 8084
**职责**: 网关注册, 心跳监控 (5s), 故障切换编排, Mesh 拓扑追踪
**接口**: MQTT gateways topics, REST `/api/v1/gateways/*`
**依赖**: PostgreSQL, Redis (心跳 TTL), MQTT

### 5.6 WS Gateway
**端口**: 8085 (核心入口)
**职责**:
- WebSocket 端点 `/ws` — JWT 认证, 文字聊天中继, WebRTC SDP/ICE 信令中继
- REST API 代理 — 转发 `/api/v1/auth/*` → Auth, `/api/v1/channels/*` → Channel
- 多设备 Hub — 同一用户可从多个设备同时连接 (ConnID 索引)
**接口**: WebSocket (信令+文字), REST 代理
**依赖**: Auth Service, Channel Service, Redis
**CORS**: 支持 `CORS_ORIGINS` 环境变量追加允许来源

### 5.7 Command Handler
**职责**: 接收语音命令转写, 意图解析, 执行动作
**接口**: MQTT command topics
**依赖**: Channel Service API

### 5.8 Notification Service
**职责**: Web 客户端推送通知, 系统广播
**接口**: MQTT system topics
**依赖**: Redis

---

## 6. MQTT 主题层级设计

(保持原有设计不变，参见 v1.0 文档)

```
voxmesh/
├── devices/{device_id}/
│   ├── audio/tx/opus        ← 上行音频 (二进制 Opus)
│   ├── audio/rx/opus        ← 下行音频
│   ├── command/transcript   ← 语音命令转写 (JSON)
│   ├── command/result       ← 命令执行结果
│   ├── status               ← 设备状态 (Retained)
│   └── config               ← 配置下发
├── channels/{channel_id}/
│   ├── audio/device/{device_id}/opus
│   ├── audio/mixed/opus     ← 服务器混音 (嵌入式用)
│   ├── presence             ← 频道成员列表
│   ├── state                ← 频道状态变更
│   └── control              ← 客户端控制
├── gateways/{gateway_id}/
│   ├── register / heartbeat / command / topology / status
├── presence/{user_id}/status
└── system/
    ├── broadcast
    └── version
```

音频载荷: 二进制 Opus + MQTT5 User Properties (device_id, channel_id, seq, timestamp, frame_count, energy)

---

## 7. WebSocket 协议设计

### 7.1 连接
`ws://<host>:8085/ws?token=<jwt>` — token 无效关闭连接

### 7.2 消息信封
```json
{"type":"<type>","id":"<req_id>","timestamp_ms":1718234567890,"payload":{}}
```

### 7.3 消息类型

**客户端→服务器**: join_channel, leave_channel, start_speaking, stop_speaking, set_mute, set_deafen, ping, chat_message, sdp_offer, sdp_answer, ice_candidate

**服务器→客户端**: channel_joined, channel_left, presence_update, user_speaking, error, pong, chat_message, sdp_offer, sdp_answer, ice_candidate

### 7.4 WebRTC 信令类型 (Phase 12 新增)

| 类型 | 方向 | Payload | 路由 |
|------|------|---------|------|
| `sdp_offer` | C→S→C | `{sdp, sender_id}` | BroadcastToChannelExcept(sender) |
| `sdp_answer` | C→S→C | `{sdp, sender_id}` | 点对点 (GetClientsByUser) |
| `ice_candidate` | C→S→C | `{candidate, sender_id}` | BroadcastToChannelExcept(sender) |

sender_id 由服务端覆盖为 `client.UserID`，防止伪造。

### 7.5 音频二进制帧 (嵌入式路径)

| Byte 0 | 类型 | 说明 |
|---|---|---|
| 0x00 | Opus Audio | Byte 1-4 seq, Byte 5-8 timestamp_ms, Byte 9-N Opus data |
| 0x01 | Silence | 静音帧 |

此二进制帧格式**仅用于嵌入式路径**。Web 客户端通过 WebRTC SRTP 传输音频，不经过此路径。

---

## 8. REST API 接口设计

**Base URL**: `http://<host>:8085/api/v1`
**认证**: `Authorization: Bearer <jwt>` (除 auth 端点外)

### 8.1 认证 `/auth`
| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /register | 注册 |
| POST | /login | 登录 |
| POST | /refresh | 刷新 token |
| POST | /logout | 吊销 token |
| GET | /me | 当前用户信息 |

### 8.2 频道 `/channels`
GET `/channels` (列表), POST `/channels` (创建), POST `/channels/{id}/join`, POST `/channels/{id}/leave`, GET `/channels/{id}/members`

### 8.3 系统
GET `/system/health`, GET `/system/version`

### 8.4 错误格式
```json
{"error":{"code":41001,"message":"频道已满","details":{}}}
```
错误码: 40xxx(认证), 41xxx(频道), 42xxx(网关), 43xxx(限速), 50xxx(服务器)

---

## 9. 数据库设计

### 9.1 核心表

**users**: id(UUID PK), username(UNIQUE), email(UNIQUE), password_hash(bcrypt), display_name, avatar_url, is_active, created_at, updated_at, last_login_at

**channels**: id(VARCHAR PK), parent_id(FK self), name, description, sort_order, max_users, is_temporary, created_by(FK users), deleted_at(软删除)

**channel_memberships**: id(BIGSERIAL PK), user_id(FK), channel_id(FK), client_type(web/embedded), device_id, joined_at, left_at

**gateways**: id(VARCHAR PK), name, api_key_hash, status, ip_address, version, capabilities(JSONB), last_heartbeat_at

**mesh_devices**: id(VARCHAR PK), gateway_id(FK), name, firmware_version, capabilities(JSONB), last_seen_at

---

## 10. 网关协调协议

### 10.1 生命周期
OFFLINE → (MQTT CONNECT + register) → REGISTERING → ONLINE → (心跳丢失15s) → DEGRADED → (30s) → OFFLINE

### 10.2 心跳检测
- 网关每 5s PUB heartbeat
- Coordinator 设 Redis key TTL 15s
- TTL 过期 → DEGRADED + 系统广播
- 30s 后仍无 → OFFLINE + 故障切换

---

## 11. 音频路由逻辑

### 11.1 Web 客户端: WebRTC P2P Mesh (当前)

**拓扑** (2-3 人频道):
```
Browser A ←── SRTP Opus (P2P) ──→ Browser B
   ↕                                  ↕
   └──────── WebSocket 信令 ──────────┘
              (ws-gateway :8085)
```

**音频管线 (发送端)**:
```
getUserMedia (noiseSuppression, echoCancellation, autoGainControl)
  → HighPass 80 Hz (去嗡声)
  → LowPass 6 kHz (去键盘声)
  → RNNoise WASM (ML 神经网络降噪)
  → SimpleGate (不说话时静音)
  → MediaStreamDestination
  → RTCPeerConnection.addTrack()
  → WebRTC 引擎: Opus 编码 + SRTP 加密 + P2P 传输
```

**音频管线 (接收端)**:
```
pc.ontrack → MediaStreamTrack → <audio> 自动播放
  浏览器引擎: NetEQ 抖动缓冲 + Opus 解码 + PLC + AEC3
```

**ICE 配置**: `stun:stun.l.google.com:19302` (局域网无需 TURN)

**信令**: SDP offer/answer + ICE candidates 通过 WebSocket 中继

### 11.2 嵌入式设备: MQTT 路径

ESP32 → ESP-IDF libopus 编码 → MQTT `voxmesh/devices/{id}/audio/tx/opus` → Audio Mixer 混音 → 分发

### 11.3 路由矩阵

| 发送方\接收方 | Web Client | 嵌入式设备 | 跨频道 |
|---|---|---|---|
| Web Client | **P2P SRTP 直连** | MQTT rx (未来 Bridge) | 不路由 |
| 嵌入式 | MQTT→Mixer→WS | ESP-NOW 直连 / MQTT rx | 不路由 |

---

## 12. 安全设计

(保持原有设计不变)

- JWT RS256: Access Token 1h, Refresh Token 30d
- TLS 全链路: WSS 443, MQTT TLS 8883
- MQTT ACL: EMQX PostgreSQL 插件
- API 安全: 速率限制, CORS 来源白名单, 参数化查询

---

## 13. 部署架构

### 13.1 Docker Compose
postgres:16-alpine, redis:7-alpine, emqx:5.x, minio, nginx + 8 个 VoxMesh 微服务

### 13.2 本地开发 (无 Docker)
```bash
make dev-local  # 一键启动 PG+Redis+Auth+Channel+WS+Frontend
```

### 13.3 项目目录结构
```
VoxMesh/
├── go.work, Makefile, CLAUDE.md
├── docker-compose.yml, .env.example
├── docs/architecture.md, docs/dev-setup.md
├── certs/, secrets/
├── nginx/, prometheus/, emqx-acl.conf
├── logs/ (开发日志)
├── migrations/ (SQL 迁移脚本)
├── scripts/
│   ├── dev-start.sh      (一键开发启动)
│   └── setup-network.sh  (外网访问网络配置)
├── services/ (8 个微服务 + pkg/ 共享包)
├── web-client/ (React 前端)
└── embedded/mqtt-contract.md
```

---

## 14. 嵌入式设备接口协议

(参见 `embedded/mqtt-contract.md` 完整定义)

- MQTT 5.0, TCP/TLS :8883
- Client ID: {gateway_id}, Keep Alive: 30s
- 音频: Opus 编码, 20ms 帧, 16kHz
- 设备不直接连 MQTT — 网关代理

---

## 15. 关键架构决策

| 决策 | 理由 |
|---|---|
| **WebRTC P2P Mesh (Web)** | 浏览器引擎 AEC3+NetEQ+FEC/PLC 远超手动实现; ~80 行替代 ~300 行 WebCodecs |
| **RNNoise WASM 降噪** | Jitsi Meet 同款 RNN 降噪; 40KB 模型; 比手写频谱 VAD 更准确 |
| **WebSocket 信令中继** | 复用现有 WS 通道; 3 个消息类型 (sdp_offer/answer/ice_candidate) |
| **MQTT 音频总线 (嵌入式)** | ESP32 优秀 MQTT 库; MQTT 5 User Properties |
| **多设备 Hub (ConnID)** | 同一用户多设备同时在线不冲突; ConnID map + userConns 索引 |
| **全 Go 技术栈** | goroutine 混音; paho.golang MQTT; 小镜像 |
| **PostgreSQL + Redis** | PG 持久化; Redis 热数据 TTL |
| **Opus 唯一编解码** | 消除转码开销; 6-510kbps 宽范围 |
| **RS256 非对称 JWT** | 公钥可分发给网关离线验证 |

---

## 16. 当前实施状态

### 16.1 已完成 (Phase 12, 2026-06-14)

| 模块 | 状态 | 说明 |
|---|---|---|
| Auth Service | ✅ 完成 | 注册/登录, JWT RS256 |
| Channel Service | ✅ 完成 | CRUD, 层级, 成员管理 |
| WS Gateway | ✅ 完成 | WebSocket, REST 代理, WebRTC 信令, 多设备 Hub |
| WebRTC P2P Mesh | ✅ 完成 | PC per peer, SDP/ICE 信令, SRTP 音频 |
| RNNoise WASM 降噪 | ✅ 完成 | `@timephy/rnnoise-wasm` + HP/LP 带通滤波 + 能量门 |
| 文字聊天 | ✅ 完成 | WebSocket chat_message, localStorage 备份 |
| 在线状态/成员面板 | ✅ 完成 | WS presence_update, 说话指示 |
| 响应式布局 | ✅ 完成 | 桌面三栏网格; 手机汉堡菜单侧滑 |
| 一键开发启动 | ✅ 完成 | `make dev-local` (PG+Redis+Auth+Ch+WS+前端) |
| LAN 外网访问 | ✅ 完成 | WSL2 镜像网络/netsh portproxy + CORS |
| 前端自动地址检测 | ✅ 完成 | 从 window.location 推导 API/WS 地址 |
| 多设备互通 | ✅ 完成 | ConnID 索引, 同用户多设备不冲突 |
| 移动端兼容 | ✅ 完成 | crypto.randomUUID polyfill, getUserMedia 检测 |
| Gateway Coordinator | 🟡 部分 | 框架完成, 需完整测试 |
| Audio Mixer | 🟡 部分 | 框架完成, 需 libopus cgo 编译链 |
| Presence Manager | 🟡 部分 | 框架完成, 需 MQTT 集成测试 |
| Command Handler | 🟡 部分 | 框架完成 |
| Notification Service | 🟡 部分 | 框架完成 |

### 16.2 WebRTC P2P 架构详情

**废弃方案**: WebCodecs Opus (Phase 9-11) — Chrome AudioData bug, 解码器采样率不可预测, 手动 jitter buffer 无法匹敌浏览器 NetEQ。

**当前架构**: WebRTC P2P Full Mesh — 浏览器引擎处理 Opus 编解码 + AEC3 + NetEQ + FEC/PLC。

**信令实现**:
- `services/ws-gateway/internal/ws/protocol.go` — SDPSignalPayload, ICECandidatePayload
- `services/ws-gateway/internal/ws/hub.go` — handleJSON 中 3 个信令 case + ConnID 多设备支持
- `web-client/src/api/webrtc.ts` — mesh 管理器 (~230 行)

**音频管线实现**:
- `web-client/src/api/audioCapture.ts` — getUserMedia → HP/LP 带通滤波 → RNNoise WASM → SimpleGate → WebRTC
- `web-client/src/components/AudioControls.tsx` — 3 个 Effect 分离信令/捕获/清理

### 16.3 未来迁移路径

**4+ 人频道**: P2P mesh → SFU (livekit / mediasoup), 前端 API 不变

**嵌入式桥接**: MQTT-Opus Bridge 独立 Go 服务, 连接 MQTT 域和 WebRTC 域

---

## 17. 功能模块验证清单

### 17.1 基础功能

- [ ] **注册/登录**: REST API 正常返回 JWT token pair
- [ ] **Token 刷新**: Access Token 过期前 5 分钟自动刷新
- [ ] **频道列表**: 显示层级频道树
- [ ] **创建频道**: 创建新频道并出现在列表中
- [ ] **加入/离开频道**: WS join_channel → 成员列表更新
- [ ] **成员面板**: 显示在线成员, 说话指示 (绿色光晕)

### 17.2 音频功能

- [ ] **麦克风采集**: getUserMedia 弹窗授权 → 本地音频流获取
- [ ] **噪声门**: 不说话时静音 (能量自适应门限)
- [ ] **RNNoise 降噪**: 键盘声/嗡声/背景噪音被抑制
- [ ] **WebRTC P2P 连接**: ICE connected → 双方互相听到清晰声音
- [ ] **静音/开麦**: 关闭麦克风后远端不再收到音频, PC 保持连接继续收
- [ ] **闭麦联动**: 点击 Deafen → 同时关闭麦克风 + 停止远端音频

### 17.3 文字聊天

- [ ] **发送消息**: 输入框发送 → 频道内所有人收到
- [ ] **接收消息**: 显示发送者和内容
- [ ] **离线缓存**: 刷新页面后消息恢复 (localStorage)

### 17.4 多设备/网络

- [ ] **多设备互通**: 桌面 (localhost) + 手机 (LAN IP) 同一频道互通
- [ ] **外网访问**: 手机通过 `http://<WINDOWS_IP>:5173` 可访问并正常通信
- [ ] **CORS**: 不同来源的 API 请求不被拦截
- [ ] **多标签页**: 同一个浏览器多个标签页不产生重连风暴 (StrictMode guard)

### 17.5 响应式布局

- [ ] **桌面 (>768px)**: 三栏布局 — 频道列表 | 聊天+控制 | 成员面板
- [ ] **手机 (<768px)**: 汉堡菜单 → 频道列表侧滑 → 点击遮罩关闭
- [ ] **深色/浅色主题**: 切换后全局样式正确

---

## 18. 未来可扩展点

### 18.1 短期 (1-3 个月)

| 扩展点 | 说明 | 优先级 |
|---|---|---|
| **SFU 集成** | 4+ 人频道切换到 livekit/mediasoup SFU | 高 |
| **TURN 服务器** | 移动网络/对称 NAT 场景下的中继 | 高 |
| **iOS Safari AudioWorklet** | iOS < 16.4 的 fallback (ScriptProcessorNode 或 bypass) | 中 |
| **生产 HTTPS 部署** | Nginx TLS 终止 + Let's Encrypt 证书自动续期 | 高 |
| **Docker Compose 生产配置** | 完整 8 服务部署 + 健康检查 + 重启策略 | 中 |
| **音频质量诊断面板** | VAD 阈值可视化、RNNoise 降噪量显示、延迟测量 | 低 |

### 18.2 中期 (3-6 个月)

| 扩展点 | 说明 |
|---|---|
| **MQTT-Opus Bridge** | 连接嵌入式 MQTT 音频域和 Web WebRTC 域 |
| **Audio Mixer 完整实现** | cgo libopus 编译链 + 抖动缓冲 + VAD + N-1 混音 |
| **频道录音** | MinIO 存储频道音频流, Opus 格式, 可选转码为 MP3/WAV |
| **权限系统增强** | 频道密码、角色继承、ACL 白名单、踢人/禁言 |
| **端到端加密 (E2EE)** | WebRTC Insertable Streams API, 频道密钥管理 |
| **WebSocket 二进制音频回退** | 当 P2P 连接失败时, 音频通过 WS 中继 (低优先级) |

### 18.3 长期 (6-12 个月)

| 扩展点 | 说明 |
|---|---|
| **ESP32 完整集成测试** | 端到端嵌入式 → MQTT → 服务器 → Web 音频链路 |
| **联邦架构** | 多 EMQX 集群, 跨地域频道同步 |
| **Kubernetes 部署** | Helm Chart, 水平自动伸缩 (HPA) |
| **Webhook 集成** | 事件通知 (Discord/Slack/Webhook), 第三方集成 API |
| **AI 噪声抑制模型更新** | 替换/微调 RNNoise 模型, 针对特定噪声场景训练 |
| **语音转文字 (服务端)** | Whisper/DeepSpeech 服务端转写替代终端 ASR |
