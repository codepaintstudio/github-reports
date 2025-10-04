# GitHub Weekly Report AI Agent

自动生成 GitHub 活动周报的 AI Agent，通过 GitHub API 拉取用户活动数据，使用 LLM 生成结构化周报，并支持企业微信/飞书推送。

## 功能特性

- **GitHub 数据采集**：自动拉取 Commits、PR、Issues、Code Reviews 等活动数据
- **AI 智能总结**：支持 OpenAI/Claude 等 LLM，自动生成专业周报
- **多种触发方式**：支持定时触发（Cron）和手动触发（HTTP API）
- **多渠道通知**：支持企业微信、飞书机器人推送
- **多用户支持**：可配置多个 GitHub 账号

## 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/minorcell/github-reports.git
cd github-reports
```

### 2. 配置文件

复制配置模板并修改：

```bash
cp config.yaml.yaml config.yaml
```

编辑 `config.yaml`：

```yaml
server:
  port: 8080

github:
  tokens:
    - token: "ghp_your_github_token"
      username: "your_username"

llm:
  provider: "deepseek" # deepseek, openai, claude
  api_key: "sk-your-api-key"
  model: "deepseek-chat"
  base_url: "https://api.deepseek.com/v1"

scheduler:
  enabled: true
  cron: "0 15 * * 5" # 每周五下午 3 点

notifiers:
  wechat:
    enabled: true
    webhook_url: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx"
  feishu:
    enabled: false
    webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
```

### 3. 运行服务

```bash
# 安装依赖
go mod download

# 运行服务
go run cmd/server/main.go --config=./configs/config.yaml
```

## API 使用

### 手动触发生成报告

```bash
curl -X POST http://localhost:8080/api/v1/reports/generate \
  -H "Content-Type: application/json" \
  -d '{
    "username": "your_username",
    "since": "2024-01-01T00:00:00Z",
    "until": "2024-01-07T23:59:59Z",
    "notify": true
  }'
```

### 健康检查

```bash
curl http://localhost:8080/api/v1/health
```

## 配置说明

### GitHub Token 配置

1. 访问 [GitHub Settings - Tokens](https://github.com/settings/tokens)
2. 生成新的 Personal Access Token
3. 需要的权限：`repo`, `read:user`

### LLM 配置

#### 使用 DeepSeek（默认推荐）

```yaml
llm:
  provider: "deepseek"
  api_key: "sk-xxx"
  model: "deepseek-chat"
  base_url: "https://api.deepseek.com/v1"
```

#### 使用 Claude

```yaml
llm:
  provider: "claude"
  api_key: "sk-ant-xxx"
  model: "claude-3-5-sonnet-20241022"
```

#### 使用 OpenAI

```yaml
llm:
  provider: "openai"
  api_key: "sk-xxx"
  model: "gpt-4-turbo-preview"
```

### 定时任务配置

使用标准 Cron 表达式：

```yaml
scheduler:
  enabled: true
  cron: "0 15 * * 5" # 每周五下午 3 点
  default_since: 168h # 默认统计最近 7 天
```

常用 Cron 表达式：

- `0 15 * * 5`：每周五下午 3 点
- `0 9 * * 1`：每周一上午 9 点
- `0 18 * * *`：每天下午 6 点

### 通知配置

#### 企业微信机器人

1. 创建群聊并添加机器人
2. 获取 Webhook URL
3. 配置到 `config.yaml`

#### 飞书机器人

1. 创建群聊并添加机器人
2. 获取 Webhook URL
3. 配置到 `config.yaml`

## 项目结构

```
github-reports/
├── cmd/
│   └── server/
│       └── main.go              # 应用入口
├── internal/
│   ├── config/                  # 配置管理
│   ├── github/                  # GitHub API 集成
│   ├── llm/                     # LLM 集成
│   ├── scheduler/               # 定时任务调度
│   ├── notifier/                # 通知模块
│   ├── reporter/                # 报告生成
│   └── api/                     # HTTP API
├── configs/
│   └── config.yaml              # 配置文件
├── CLAUDE.md                    # 项目设计文档
└── README.md                    # 使用文档
```

## 开发构建

### 编译二进制

```bash
go build -o github-reports cmd/server/main.go
./github-reports --config=./configs/config.yaml
```

### 运行测试

```bash
go test ./...
```

## 使用示例

### 场景 1：个人周报

每周五自动生成个人本周的 GitHub 活动周报，并发送到企业微信。

### 场景 2：团队周报

配置多个 GitHub 账号，批量生成团队成员的周报。

### 场景 3：临时查询

通过 API 手动触发，查询特定时间段的活动情况。

## 贡献指南

欢迎提交 Issue 和 Pull Request！

## License

MIT License

## 相关链接

- [GitHub API 文档](https://docs.github.com/en/rest)
- [企业微信机器人文档](https://developer.work.weixin.qq.com/document/path/91770)
- [飞书机器人文档](https://open.feishu.cn/document/ukTMukTMukTM/ucTM5YjL3ETO24yNxkjN)
