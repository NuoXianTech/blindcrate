# blindcrate

🚀 blindcrate —— 盲盒开箱 · 保险柜竞拍对战平台

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
[![ESLint](https://github.com/NuoXianTech/blindcrate/actions/workflows/eslint.yml/badge.svg)](https://eslint.org/)

## 📖 项目简介

**blindcrate** 是一个「开箱 + 博弈」双核驱动的箱庭类游戏平台：单人侧通过开箱抽取战利品、在保险柜网格里搜索变现；对战侧由两名或四名玩家围绕一柜未知价值的战利品展开多轮盲拍，靠信息差、技能与心理博弈争夺整柜所有权。

### ✨ 主要特性

- 🎁 **开箱核心循环** —— 免费额度 / 次数券 / 积分三档计价，权重 × 品质倍率加权抽取，7 级品质（灰·绿·蓝·紫·红·金·炫彩）
- 🔐 **保险柜网格搜索** —— 形状装箱（`WxH`）的可挖格盲盒，按品质分级搜索耗时，带「亏损保底」赔付
- 🧠 **记忆扫描（vmem）** —— 18×18 大网格翻牌，8 次搜索机会，品质漂移 / 坐标换位 / 幻影格 / 信号模糊四重干扰与三种道具反制
- ⚔️ **多人保险柜竞拍** —— 2 人 4 局 / 4 人 6 局，`confirm → reveal → bidding → revealed` 阶段机，强买（1.8×）与压哨出价
- 🤖 **拟人化人机** —— 三档信息利用率（0 / 45 % / 100 %）、三种性格、按人类习惯取整与压哨的计划出价
- 🎯 **技能博弈** —— 17 个内置技能、17 种效果类型，信息 / 干扰 / 修正 / 战术四系，双击装备 + 每局使用次数限制
- 📚 **图鉴与收集** —— 按品质集齐发奖，未收录物品名称打码防剧透，全服收录排行榜
- 💳 **月卡 / 通行证 / 任务** —— 三档月卡（可叠加折扣与免票）、30 级赛季通行证、日常与周常任务引擎
- 🎪 **运营活动** —— 签到、限时掉落、抽奖券转盘、战利品币兑换、红包实物申领与卡密池
- 🧩 **Buff 体系** —— 品质倍率、金币倍率、掉落权重、额外次数、对战免票，防叠卡的分桶取值规则
- 🛡️ **工程化底座** —— 统一响应体、错误码规范、Swagger 全量文档、无外键有索引、条件更新扣款、Redis 分布式锁

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
- **API 文档**: http://localhost:8000/swagger/index.html
- **健康检查**: http://localhost:8000/api/v1/health
- **就绪检查**: http://localhost:8000/api/v1/ready

## ⚙️ 配置说明

### 主要配置项

| 配置项 | 说明 | 示例 |
|--------|------|------|
| | | |

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
http://localhost:8000/swagger/index.html
```

### 主要 API 端点

- `GET /api/v1/health` - 健康检查
- `GET /api/v1/ready` - 就绪检查
- `GET /api/oauth2/login` - OAuth2 登录
- `GET /api/projects` - 获取项目列表
- `POST /api/projects` - 创建新项目

## 🧪 测试

```bash
# 后端测试
go test ./...

# 前端测试
cd frontend
pnpm test
```

## 🚀 部署

### 生产环境部署

1. 构建前端资源：
   ```bash
   cd frontend && pnpm build
   ```

2. 编译后端程序：
   ```bash
   go build -o cdk main.go
   ```

3. 配置生产环境的 `config.yaml`

4. 启动服务：
   ```bash
   ./cdk api
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

## ❤️ 致谢

感谢所有为本项目做出贡献的开发者，以及参与玩法数值与平衡性测试的社区成员。

## 📈 项目趋势

[![Star History Chart](https://api.star-history.com/svg?repos=NuoXianTech/blindcrate&type=Date)](https://star-history.com/#NuoXianTech/blindcrate&Date)
