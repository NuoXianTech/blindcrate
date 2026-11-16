# blindcrate

🚀 blindcrate —— 四人暗标竞拍对战。看不见货仓里装了什么，也看不见别人此刻出多少，靠轮廓、主持人的提示和情报道具估价，拍下之后才开箱。

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.26.10-blue.svg)](https://go.dev/)
[![Next.js](https://img.shields.io/badge/Next.js-16-black.svg)](https://nextjs.org/)
[![React](https://img.shields.io/badge/React-19-blue.svg)](https://react.dev/)

[![GitHub release](https://img.shields.io/github/v/release/NuoXianTech/blindcrate?include_prereleases)](https://github.com/NuoXianTech/blindcrate/releases)
[![GitHub stars](https://img.shields.io/github/stars/NuoXianTech/blindcrate)](https://github.com/NuoXianTech/blindcrate/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/NuoXianTech/blindcrate)](https://github.com/NuoXianTech/blindcrate/network)
[![GitHub issues](https://img.shields.io/github/issues/NuoXianTech/blindcrate)](https://github.com/NuoXianTech/blindcrate/issues)
[![GitHub pull requests](https://img.shields.io/github/issues-pr/NuoXianTech/blindcrate)](https://github.com/NuoXianTech/blindcrate/pulls)
[![GitHub contributors](https://img.shields.io/github/contributors/NuoXianTech/blindcrate)](https://github.com/NuoXianTech/blindcrate/graphs/contributors)

[![Backend Build](https://github.com/NuoXianTech/blindcrate/actions/workflows/build_backend.yml/badge.svg)](https://github.com/NuoXianTech/blindcrate/actions/workflows/build_backend.yml)
[![Frontend Build](https://github.com/NuoXianTech/blindcrate/actions/workflows/build_frontend.yml/badge.svg)](https://github.com/NuoXianTech/blindcrate/actions/workflows/build_frontend.yml)
[![CodeQL](https://github.com/NuoXianTech/blindcrate/actions/workflows/codeql.yml/badge.svg)](https://github.com/NuoXianTech/blindcrate/actions/workflows/codeql.yml)
[![ESLint](https://github.com/NuoXianTech/blindcrate/actions/workflows/eslint.yml/badge.svg)](https://github.com/NuoXianTech/blindcrate/actions/workflows/eslint.yml)

## 📖 项目简介

**blindcrate** 是一个四人暗标竞拍的网页对战游戏。每局是一个仓库，里面有一个或几个货仓依次开拍，四个座位共用各自带入的一笔资金。开拍时只能看到货仓的格子大小、公示的掉落概率和少数几件藏品的轮廓，藏品的品质、名称和总价值都不公开；每一轮所有人同时暗中出价，较早轮次的出价在该轮结束后公开，最后一轮和加轮保持暗标。成交后货仓逐件扫描开箱，拍下的藏品进入背包，可以收藏、出售或用来完成收集。真人不足时由人机补位，人机只使用真人也看得到的公开信息。

除对局外，项目包含一套完整的账号、经济和运营后台：钱包与背包、商店、任务、签到、邮件、兑换码、图鉴、排位赛季与排行榜，以及地图、物品、掉落池等全部内容的后台配置。

### ✨ 主要特性

- 🔨 **多轮暗标竞拍** - 四人同时暗中出价，可速胜，并列时加轮
- 📦 **多货仓的仓库** - 一局依次开拍多个货仓，带入资金整局共用
- 🕵️ **信息博弈** - 只公开轮廓和概率，靠主持人提示与情报道具
- 🎁 **藏品与开箱** - 品质可自定义的藏品，成交后逐件扫描揭晓
- 🤖 **人机对手** - 多种风格的人机补位，另有人机模式和练习局
- 🏆 **排位与赛季** - 赛季段位分与奖励，带防串通和挂机禁赛
- 💰 **经济系统** - 多货币钱包、背包出售和带限购的商店
- 🎯 **成长与活动** - 任务、签到、邮件、兑换码、图鉴和排行榜
- 🧩 **可配置的内容** - 货币、品质、段位、榜单、人机风格与昵称、地图、物品、掉落池均无内置数据，可配置并整包导入导出
- ⚖️ **保存时自动验算** - 模拟对局，自动推导起拍价、折让和入场费
- 🛠️ **运营后台** - 用户管理、调账、系统设置和操作日志
- 🔐 **账号与安全** - 邮箱验证、人机验证和接口限流，防刷防滥用
- 🧱 **工程化** - 前后端分离，实时对局，完整的接口文档

## 🏗️ 架构概览

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │    Backend      │    │   Database      │
│   (Next.js)     │◄──►│     (Go)        │◄──►│  (PGSQL/Redis)  │
│                 │    │                 │    │                 │
│ • React 19      │    │ • Gin Framework │    │ • PostgreSQL    │
│ • TypeScript    │    │ • WebSocket     │    │ • Redis Cache   │
│ • Tailwind CSS  │    │ • OpenTelemetry │    │ • Session Store │
│ • Shadcn UI     │    │ • Swagger API   │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 🛠️ 技术栈

### 后端

- **[Go 1.26.10](https://go.dev/doc)** - 主要开发语言
- **[Gin](https://github.com/gin-gonic/gin)** - Web 框架
- **[PostgreSQL](https://www.postgresql.org/)** - 主数据库
- **[Redis](https://github.com/redis/redis)** - 缓存和会话存储
- **[OpenTelemetry](https://opentelemetry.io)** - 可观测性
- **[Swagger](https://github.com/swaggo/swag)** - API 文档

### 前端

- **[Next.js 16](https://github.com/vercel/next.js)** - React 框架
- **[React 19](https://github.com/facebook/react)** - UI 库
- **[TypeScript](https://github.com/microsoft/TypeScript)** - 类型安全
- **[Tailwind CSS 4](https://github.com/tailwindlabs/tailwindcss)** - 样式框架
- **[Shadcn UI](https://github.com/shadcn-ui/ui)** - 组件库
- **[Lucide Icons](https://github.com/lucide-icons/lucide)** - 图标库

## 📋 环境要求

- **Go** 1.26.10
- **Node.js** 24.22.1
- **PostgreSQL** >= 18.6
- **Redis** >= 8.0
- **pnpm** >= 8.0

## 🚀 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/NuoXianTech/blindcrate.git
cd blindcrate
```

### 2. 配置环境

```bash
cp config.example.yaml config.yaml
```

编辑 `config.yaml` 文件，配置数据库连接、Redis、OAuth2 等信息。

### 3. 初始化数据库

```bash
# 创建数据库
psql -U postgres -c "CREATE DATABASE blindcrate WITH ENCODING 'UTF8';"

# 运行迁移（启动后端时会自动执行）
```

### 4. 启动后端

```bash
# 安装 Go 依赖
go mod tidy

# 生成 API 文档
make swagger

# 启动后端服务
go run main.go api
```

### 5. 启动前端

```bash
cd frontend

# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev
```

### 6. 访问应用

- **前端界面**: http://localhost:3000
- **API 文档**: http://localhost:8000/api/swagger/index.html
- **健康检查**: http://localhost:8000/api/v1/health
- **就绪检查**: http://localhost:8000/api/v1/ready

## ⚙️ 配置说明

### 主要配置项

| 配置项 | 说明 | 示例 |
|--------|------|------|
| `app.addr` | 后端服务监听地址 | `:8000` |
| `app.trusted_proxies` | 可信的反向代理, 用于识别客户端 IP 和按 IP 限流 | `["127.0.0.1", "::1"]` |
| `worker.port` | Worker 探针端口 | `8001` |
| `schedule.port` | Scheduler 探针端口 | `8002` |
| `ws.origin_patterns` | WebSocket 连接域名 | `["localhost:3000"]` |
| `database.host` | PostgreSQL 数据库地址 | `127.0.0.1` |
| `redis.host` | Redis 服务器地址 | `127.0.0.1` |

详细配置说明请参考 `config.example.yaml` 文件。

## 🔧 开发指南

### 后端开发

```bash
# 运行 API 服务器
go run main.go api

# 运行任务调度器
go run main.go scheduler

# 运行工作队列
go run main.go worker

# 生成 Swagger 文档
make swagger

# 代码格式化和检查
make tidy
```

### 前端开发

```bash
cd frontend

# 开发模式（使用 Turbopack）
pnpm dev

# 构建生产版本
pnpm build

# 启动生产服务
pnpm start

# 代码检查和格式化
pnpm lint
pnpm format
```

## 📚 API 文档

API 文档通过 Swagger 自动生成，启动后端服务后可访问：

```
http://localhost:8000/api/swagger/index.html
```

### 主要 API 端点

- `GET /api/v1/health` - 健康检查
- `GET /api/v1/ready` - 就绪检查

## 🧪 测试

```bash
# 后端测试
go test ./...

# 前端测试
cd frontend
pnpm test
```

## 🚀 部署

### Docker 部署

```bash
# 构建镜像
docker build -t blindcrate .

# 运行容器
docker run -d -p 8000:8000 blindcrate
```

### 生产环境部署

1. 构建前端资源：
   ```bash
   cd frontend && pnpm build
   ```

2. 编译后端程序：
   ```bash
   go build -o blindcrate main.go
   ```

3. 配置生产环境的 `config.yaml`

4. 启动服务：
   ```bash
   ./blindcrate api
   ```

## 🤝 贡献指南

我们欢迎社区贡献！请在提交代码前阅读：

- [贡献指南](CONTRIBUTING.md)
- [行为准则](CODE_OF_CONDUCT.md)
- [贡献者许可协议](CLA.md)

### 提交流程

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/your-feature`)
3. 提交更改 (`git commit -am 'Add your feature'`)
4. 推送到分支 (`git push origin feature/your-feature`)
5. 创建 Pull Request

提交信息遵循 `<type>(<scope>): <subject>` 格式，且**必须包含 Scope**。

## 📄 许可证

本项目基于 [MIT 许可证](LICENSE) 开源。

## 🔗 相关链接

- [问题反馈](https://github.com/NuoXianTech/blindcrate/issues)

## ❤️ 致谢

感谢所有为本项目做出贡献的开发者，以及参与玩法数值与平衡性测试的社区成员。

## 📈 项目趋势

[![Star History Chart](https://api.star-history.com/svg?repos=NuoXianTech/blindcrate&type=Date)](https://star-history.com/#NuoXianTech/blindcrate&Date)
