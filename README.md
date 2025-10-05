# GitHub Reports AI Agent

基于飞书 Webhook 触发的 GitHub 技术动态分析 AI Agent。通过自然语言指令，自动拉取 GitHub 用户活动数据，使用 DeepSeek LLM 生成技术总结报告，并推送到飞书。

## 核心特性

- **智能解析**：通过 LLM 从自然语言中提取 GitHub 用户名
- **数据采集**：自动拉取 Commits、PR、Issues、Code Reviews 等活动数据
- **AI 总结**：使用 DeepSeek 生成技术深度分析报告
- **飞书集成**：Webhook 触发 + 自动推送结果到飞书
- **异步处理**：立即响应请求，后台处理，避免超时重试
- **Token 认证**：Webhook 接口受 token 保护
- **错误推送**：处理失败时自动将错误信息发送到飞书

## 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/minorcell/github-reports.git
cd github-reports
```

### 2. 配置文件

复制配置模板并修改：

```bash
cp config.example.yaml config.yaml
```

编辑 `config.yaml`：

```yaml
server:
  port: 8080

webhook:
  token: "your_webhook_token" # Webhook 认证 token

github:
  tokens:
    - token: "ghp_your_github_token_here"

llm:
  provider: "deepseek"
  api_key: "sk-your-deepseek-api-key"
  model: "deepseek-chat"
  base_url: "https://api.deepseek.com/v1"

notifiers:
  feishu:
    enabled: true
    webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/your-hook-here"
```

### 3. 运行服务

```bash
# 安装依赖
go mod download

# 编译并运行
go build -o github-reports cmd/server/main.go
./github-reports --config=./config.yaml
```

或者直接运行：

```bash
go run cmd/server/main.go --config=./config.yaml
```

## 使用方式

### 飞书机器人触发（主要使用方式）

#### 1. 配置飞书机器人

在飞书群中添加自定义机器人，配置 Webhook 转发到你的服务：

```
POST https://your-domain.com/api/v1/webhook
```

#### 2. 在飞书中发送消息

用户在飞书群中 @ 机器人并发送：

```
@机器人 帮我分析一下 minorcell 的 GitHub 动态
```

或者：

```
@机器人 生成 github.com/minorcell 的技术总结
```

#### 3. 自动流程

1. 飞书发送请求到 Webhook
2. 服务立即返回 200 响应（避免超时）
3. 后台异步处理：
   - LLM 提取用户名 `minorcell`
   - 拉取 GitHub 最近 7 天的活动数据
   - LLM 生成技术分析报告
   - 推送报告到飞书群

#### 4. 接收报告

用户在飞书中收到 Markdown 格式的技术分析：

```markdown
# [minorcell](https://github.com/minorcell) 的 GitHub 技术动态分析

## github-reports

- 实现了基于 Webhook 的智能周报生成，采用 LLM 提取用户信息
- 重构 LLM 模块，统一使用 DeepSeek API
- 优化错误处理，失败时自动推送错误到飞书

## 总体技术分析

近期主要精力集中在新功能开发。
解决的关键难题包括：LLM 集成、GitHub API 数据聚合。
共提交 15 个 commits，新增 2000 行代码，删除 500 行代码。
```

### 直接 API 调用（测试用）

```bash
curl -X POST http://localhost:8080/api/v1/webhook \
  -H "Authorization: YOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "帮我分析一下 minorcell 的 GitHub 动态"
  }'
```

## API 接口

### POST /api/v1/webhook

智能 Webhook 接口，接收自然语言指令。

**认证**：需要 Authorization Header

**请求示例**：

```json
{
  "content": "帮我分析一下 minorcell 的 GitHub 动态"
}
```

**响应（立即返回）**：

```json
{
  "status": "accepted",
  "message": "Request received, processing in background"
}
```

**错误处理**：

- 解析失败时立即返回 400/500 错误
- 处理失败时错误信息会自动发送到飞书

## 工作流程

```
用户在飞书 @ 机器人
    ↓
飞书机器人转发到你的 Webhook
    ↓
验证 Token
    ↓
立即返回 200 响应（避免飞书超时重试）
    ↓
后台异步处理：
  - LLM 提取 GitHub 用户名
  - 拉取 GitHub 活动数据 (最近 7 天)
  - LLM 生成技术分析报告（60秒超时）
  - 推送报告到飞书
    ↓
用户在飞书收到报告
```

**超时控制**：
- HTTP 客户端：60 秒超时
- 异步处理：5 分钟总超时
- 飞书 Webhook：立即响应，不等待处理完成

**错误处理**：处理失败时错误信息自动发送到飞书。

## 相关链接

- [GitHub API 文档](https://docs.github.com/en/rest)
- [DeepSeek API 文档](https://platform.deepseek.com/api-docs/)
- [飞书机器人文档](https://open.feishu.cn/document/ukTMukTMukTM/ucTM5YjL3ETO24yNxkjN)
