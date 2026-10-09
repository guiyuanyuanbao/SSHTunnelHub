<div align="center">

  <img src="docs/logo.png" alt="SSHTunnelHub Logo" width="160" height="160" style="border-radius: 24px; box-shadow: 0 8px 24px rgba(0,0,0,0.3);" />

  # SSHTunnelHub

  **现代化、高可用、工业级 SSH 隧道与端口转发管理中心**

  [![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
  [![Vue Version](https://img.shields.io/badge/Vue-3.4+-4FC08D?style=flat&logo=vuedotjs)](https://vuejs.org)
  [![Element Plus](https://img.shields.io/badge/Element%20Plus-2.8+-409EFF?style=flat&logo=element)](https://element-plus.org)
  [![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com/)
  [![GitHub Actions](https://img.shields.io/badge/CI-Docker%20Publish-2088FF?style=flat&logo=githubactions)](https://github.com)
  [![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

  <p align="center">
    <b>纯 Go 嵌入式单二进制</b> · <b>5 维工业级心跳自愈</b> · <b>正反双向多跳转发</b> · <b>实时网络日志审计</b>
  </p>

</div>

---

## 📖 简介

**SSHTunnelHub** 是专为开发者与运维工程师打造的轻量级、高可用 SSH 端口转发与隧道生命周期管理平台。

无需编写复杂的 `ssh -N -f -L/-R` 脚本或忍受 SSH 连接因网络波动而假死断开。SSHTunnelHub 提供了直观优雅的 Web 控制台、完备的密钥加密存储、正向/反向多跳代理、五层全套心跳自愈机制以及毫秒级实时网络运行日志监控。

系统采用纯 Go 嵌入式 SQLite 存储，**零外部数据库依赖，零 CGO 编译依赖**，支持以 **Docker 镜像** 或 **单文件二进制** 开箱即用。

---

## ✨ 核心特性

- 🖥️ **现代可视化控制台**
  - 基于 Vue 3 + Vite + Element Plus + Pinia 构建，全站响应式卡片流布局。
  - **严谨居中对齐**：监听源端与目标端采用 CSS Grid 3 列严格对称布局，中轴线固定通道流动箭头，视觉平衡规整。
  - **毫秒级性能指标**：WebSocket 全双工广播，实时呈现各隧道的活跃连接数、上传/下行速率及运行时长。

- 🔑 **企业级主机与凭据安全**
  - 支持 **密码认证** 与 **私钥认证**（支持 PEM、OpenSSH 格式、支持加密私钥 Passphrase、支持从本地文件一键拖拽导入）。
  - 所有主机敏感凭据在落库前经 **AES-256-GCM** 算法不可逆/强加密存储，API 接口默认脱敏下发。
  - **实时连通性探测**：保存前后均可一键探测 SSH 主机连通性，秒级反馈握手往返延迟 (RTT) 与远端服务版本。

- 🔄 **完备的双向转发拓扑**
  - **正向隧道 (-L Local Forwarding)**：本地监听端口，将流量经 SSH 加密链路透传至远端可达的主机。不仅限于远端 `127.0.0.1`，亦可代理到远端内网任意服务（如远端私有云数据库 `192.168.1.50:3306`）。
  - **反向隧道 (-R Remote Forwarding)**：在远端 SSH 服务器上暴露端口监听，将远程外部流量逆向穿透回传至本地开发机或本地内网目标服务（如本地 Webhook 调试）。

- 🛡️ **五维工业级保活与自愈机制 (The 5-Tier Resilience Suite)**
  1. **容错连续心跳探测**：15s 间隔心跳探测，引入 `ServerAliveCountMax = 3` 计数器，允许公网偶发微弱抖动，连续 3 次无响应才触发重连。
  2. **反向端口抢占释放**：反向隧道重连前主动发送 `cancel-tcpip-forward` 注销残留端口，彻底杜绝 `address already in use` 冲突死循环。
  3. **指数退避与随机抖动**：重连间隔按 2s $\rightarrow$ 4s $\rightarrow$ 8s ... 指数上升（上限 60s），叠加 $\pm 20\%$ 全随机抖动（Jitter），杜绝网络恢复时的“惊群风暴”。
  4. **目标服务端到端健康探测**：后台每 30s 独立拨号探测后端真实服务，区分 SSH 链路故障与目标业务宕机，精准亮起黄色告警而不打断控制流。
  5. **套接字保活与毫秒级协同熔断**：开启底层 TCP `SO_KEEPALIVE`（15s），通过 `sync.Once` 实现读写双向瞬间拆除，防止文件描述符与端口挂死。

- 📜 **运行与网络日志控制台**
  - 独立终端暗黑风格日志面板，结构化记录认证、心跳探测、重连倒计时与端口注销等完整生命周期。
  - 支持 INFO / SUCCESS / WARN / ERROR 级别切换、特定隧道精准过滤、关键词模糊检索、实时自动滚动追踪及一键复制。
  - 每个隧道卡片底栏均配有一键直达日志入口。

- 🐳 **Docker 原生 & 自动化发布**
  - 预置多阶段构建 `Dockerfile` 与 `docker-compose.yml`，镜像体积精炼（< 40MB）。
  - 内置 GitHub Actions CI，代码推送或打标签时自动编译发布多架构镜像（`linux/amd64`, `linux/arm64`）至 GitHub Container Registry (GHCR)。

---

## 🐳 Docker 快速部署

### 1. 使用 Docker Compose (推荐)

仓库已内置 [`docker-compose.yml`](docker-compose.yml)，执行以下命令即可启动：

```bash
# 启动服务
docker compose up -d

# 查看运行日志
docker compose logs -f
```

打开浏览器访问：**`http://localhost:9090`**

#### 默认 `docker-compose.yml` 配置示例：

```yaml
services:
  sshtunnelhub:
    image: ghcr.io/${GITHUB_REPOSITORY:-owner/sshtunnelhub}:latest
    build:
      context: .
      dockerfile: Dockerfile
    container_name: sshtunnelhub
    restart: unless-stopped
    ports:
      - "9090:9090"               # Web 管理控制台
      - "10000-10050:10000-10050" # 预留正向隧道转发端口范围
    environment:
      - PORT=9090
      - DATA_DIR=/data
      - TZ=Asia/Shanghai
    volumes:
      - ./data:/data              # 数据持久化目录 (SQLite 数据库与加密密钥)
```

> [!TIP]
> **Host 网络模式（Linux 环境推荐）**：
> 如果部署在 Linux 宿主机上，且希望隧道随意监听宿主机的任意端口，建议直接在 `docker-compose.yml` 中添加 `network_mode: "host"`。此时容器直接复用宿主机网络栈，无需提前声明端口映射范围。

---

### 2. 使用 Docker CLI 运行

```bash
docker run -d \
  --name sshtunnelhub \
  --restart unless-stopped \
  -p 9090:9090 \
  -p 10000-10050:10000-10050 \
  -v $(pwd)/data:/data \
  -e TZ=Asia/Shanghai \
  ghcr.io/owner/sshtunnelhub:latest
```

---

## 🚀 二进制直接运行

SSHTunnelHub 支持编译为单一无外部依赖的二进制可执行程序。

### 运行现成二进制

直接在终端或命令行中启动：

```powershell
# Windows
.\sshtunnelhub.exe

# Linux / macOS
./sshtunnelhub
```

打开浏览器访问：`http://127.0.0.1:9090`

**常用启动命令行参数：**
- `-port 9090`：指定 Web 服务监听端口（默认 `9090`，也可通过环境变量 `PORT=9090` 指定）。
- `-data-dir ./data`：指定数据库与主密钥存储目录（默认 `./data`，也可通过环境变量 `DATA_DIR=/data` 指定）。

---

## ⚙️ GitHub Actions CI 自动发布

本项目包含完整的自动化 CI/CD 工作流：[`.github/workflows/docker-publish.yml`](.github/workflows/docker-publish.yml)。

### 触发条件与发布机制：
1. **推送至 `main` 分支**：自动执行单元测试、编译多架构镜像，并推送到 GHCR，标记为 `latest` 及分支名。
2. **发布 Release 标签 (`v*.*.*`)**：自动打标签为语义化版本号（如 `v1.0.0`、`1.0`、`latest`）。
3. **多架构支持**：利用 QEMU 与 Docker Buildx 自动化并行构建并输出 `linux/amd64` 与 `linux/arm64` 双架构镜像。

---

## 📖 隧道转发模型与应用场景

### 1. 正向隧道 (-L Local Forwarding)

```text
[本地/外部客户端] ───> [Hub 本地监听端口] ───(SSH通道)───> [远程SSH主机] ───> [目标服务:目标端口]
```

- **典型场景**：本地想要访问远程内网的数据库、Redis 或微服务集群。
- **配置示例**：
  - 监听地址与端口：`127.0.0.1:13306`（本地应用连接此端口）
  - 目标地址与端口：`192.168.1.100:3306`（远程私有云内网数据库）

### 2. 反向隧道 (-R Remote Forwarding)

```text
[外部访客] ───> [远程SSH主机监听端口] ───(SSH通道)───> [Hub 本地] ───> [内网目标服务:目标端口]
```

- **典型场景**：在没有公网 IP 的本地开发机上调试第三方回调（如微信/支付宝支付回调、GitHub Webhook）。
- **远端服务端配置提醒**：
  若希望远程监听端口对外开放公网访问，需确保远程主机 `/etc/ssh/sshd_config` 中配置了：
  ```ini
  GatewayPorts yes
  ```
  修改后执行 `sudo systemctl restart sshd` 生效。

---

## 🛠️ 本地开发与全量构建

### 前置要求
- Go 1.22+
- Node.js 20+ & pnpm 9+

### 一键构建与打包 (Windows PowerShell)

```powershell
.\build.ps1
```
脚本将自动构建前端静态资源并将其复制到后端目录，通过 Go `embed` 编译生成集成的 `sshtunnelhub.exe`。

### 前后端分离热重载开发模式

1. **后端启动 (Go)**：
   ```bash
   cd backend
   go run ./cmd/server
   ```
2. **前端启动 (Vue 3)**：
   ```bash
   cd frontend
   pnpm install
   pnpm run dev
   ```
   前端服务将启动在 `http://127.0.0.1:5173`，通过 Vite 代理自动直通后端 API 与 WebSocket。

---

## 📁 目录结构

```text
SSHTunnelHub/
├── .github/
│   └── workflows/
│       └── docker-publish.yml   # GitHub Actions 自动化多架构 Docker 发布工作流
├── backend/                     # Go 后端工程
│   ├── cmd/server/main.go       # 服务入口与优雅停机
│   ├── internal/
│   │   ├── crypto/              # AES-256-GCM 凭据加密
│   │   ├── db/                  # SQLite 纯 Go 存储 (无 CGO 依赖)
│   │   ├── handler/             # RESTful API、WebSocket 广播与日志接口
│   │   ├── logger/              # 环形内存日志缓冲区 (RingLogger)
│   │   ├── model/               # Host、Tunnel 数据模型与健康状态
│   │   ├── service/             # 业务服务层
│   │   ├── sshutil/             # SSH 握手、密钥解析与网络延迟测试
│   │   ├── tunnel/              # 隧道引擎：Worker、Forward、Reverse、KeepAlive、Metrics
│   │   └── webui/               # go:embed 静态资源集成与 SPA 回退
│   └── go.mod
├── frontend/                    # Vue 3 前端工程
│   ├── src/
│   │   ├── api/                 # Axios 接口封装
│   │   ├── components/          # 对话框与状态组件
│   │   ├── store/               # Pinia 状态管理与实时推送流
│   │   ├── views/               # Dashboard 概览、Hosts 主机管理、Tunnels 隧道管理、Logs 日志
│   │   └── App.vue
│   └── package.json
├── docs/                        # 项目资产与图标
│   ├── logo.png                 # 高清项目 Logo
│   └── logo.svg                 # 矢量徽标
├── docker-compose.yml           # Docker Compose 一键启动编排
├── Dockerfile                   # 多阶段极简生产镜像构建定义
├── build.ps1                    # PowerShell 一键构建打包脚本
└── README.md                    # 项目文档
```

---

## 📄 开源许可证

本项目基于 [MIT License](LICENSE) 开源。欢迎提交 Issue 与 Pull Request！
