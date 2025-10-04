package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github-reports/internal/config"
	"github-reports/internal/github"
	"github-reports/internal/llm"
	"github-reports/internal/notifier"
	"github-reports/internal/reporter"

	"github.com/gin-gonic/gin"
)

// Handler 处理 HTTP 请求
type Handler struct {
	config *config.Config
}

// NewHandler 创建一个新的 API 处理器
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		config: cfg,
	}
}

// AuthMiddleware 检查 webhook 令牌
func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")

		// 支持 "Bearer TOKEN" 和 "TOKEN" 格式
		token = strings.TrimPrefix(token, "Bearer ")

		if token != h.config.Webhook.Token {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// WebhookRequest 表示 webhook 请求体
type WebhookRequest struct {
	Content string `json:"content" binding:"required"`
}

// Webhook 处理 POST /api/v1/webhook
// 工作流程：解析内容 -> 立即返回响应 -> 异步处理（提取用户名 -> 获取 GitHub 数据 -> 生成报告 -> 发送到飞书）
func (h *Handler) Webhook(c *gin.Context) {
	// 读取原始请求体
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		fmt.Printf("[Webhook] 读取请求体失败: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// 打印原始 JSON 请求体
	fmt.Printf("[Webhook] 原始请求体 JSON: %s\n", string(bodyBytes))
	fmt.Printf("[Webhook] Content-Type: %s\n", c.GetHeader("Content-Type"))

	// 重新设置请求体供后续绑定使用
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req WebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("[Webhook] 解析请求失败: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 打印解析后的请求体内容
	fmt.Printf("[Webhook] 解析后的请求体: %+v\n", req)

	log := func(message string) {
		println("[Webhook]", message)
	}

	log("收到 webhook 请求")
	log("内容: " + req.Content)

	// 检查是否已配置飞书
	if !h.config.Notifiers.Feishu.Enabled {
		log("错误: 飞书通知未启用")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Feishu notification is not enabled"})
		return
	}

	// 立即返回响应，避免飞书超时
	c.JSON(http.StatusOK, gin.H{
		"status":  "accepted",
		"message": "Request received, processing in background",
	})

	// 在后台异步处理
	go h.processWebhookAsync(req.Content)
}

// processWebhookAsync 异步处理 webhook 请求
func (h *Handler) processWebhookAsync(content string) {
	log := func(message string) {
		println("[Webhook-Async]", message)
	}

	// 使用新的 context，设置合理的超时时间（因为原请求的 context 已经结束）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 向飞书发送错误的辅助函数
	sendErrorToFeishu := func(errorMsg string) {
		if h.config.Notifiers.Feishu.Enabled {
			feishuNotifier := notifier.NewFeishuNotifier(h.config.Notifiers.Feishu.WebhookURL)
			errorReport := fmt.Sprintf("# ❌ GitHub 周报生成失败\n\n**错误信息：**\n%s", errorMsg)
			_ = feishuNotifier.Send(ctx, errorReport)
		}
	}

	// 步骤 1: 使用 LLM 提取 GitHub 用户名
	log("步骤 1: 使用 LLM 提取 GitHub 用户名...")
	llmClient, err := llm.NewClient(h.config.LLM)
	if err != nil {
		errMsg := "创建 LLM 客户端失败: " + err.Error()
		log("错误: " + errMsg)
		sendErrorToFeishu(errMsg)
		return
	}

	username, err := llmClient.ExtractGitHubUsername(ctx, content)
	if err != nil {
		errMsg := "提取 GitHub 用户名失败: " + err.Error()
		log("错误: " + errMsg)
		sendErrorToFeishu(errMsg)
		return
	}

	// 清理用户名（去除空白字符和换行符）
	username = strings.TrimSpace(username)
	log("提取到的 GitHub 用户名: " + username)

	// 步骤 2: 查找 GitHub 令牌
	log("步骤 2: 查找 GitHub token...")
	var token string
	for _, t := range h.config.GitHub.Tokens {
		if t.Username == username {
			token = t.Token
			log("找到匹配的 GitHub token (username: " + t.Username + ")")
			break
		}
	}

	if token == "" && len(h.config.GitHub.Tokens) > 0 {
		token = h.config.GitHub.Tokens[0].Token
		log("使用默认 GitHub token 查询用户 " + username)
	}

	if token == "" {
		errMsg := "未配置 GitHub token"
		log("错误: " + errMsg)
		sendErrorToFeishu(errMsg)
		return
	}

	// 步骤 3: 拉取 GitHub 活动数据
	log("步骤 3: 拉取 GitHub 活动数据...")
	until := time.Now()
	since := until.Add(-7 * 24 * time.Hour) // 默认为最近 7 天

	githubClient := github.NewClient(token)
	rep := reporter.NewReporter(githubClient, llmClient)
	report, err := rep.GenerateReport(ctx, username, since, until)
	if err != nil {
		errMsg := fmt.Sprintf("生成 %s 的周报失败: %s", username, err.Error())
		log("错误: " + errMsg)
		sendErrorToFeishu(errMsg)
		return
	}

	log("报告生成成功")

	// 步骤 4: 发送到飞书
	log("步骤 4: 发送到飞书...")
	feishuNotifier := notifier.NewFeishuNotifier(h.config.Notifiers.Feishu.WebhookURL)
	if err := feishuNotifier.Send(ctx, report); err != nil {
		errMsg := "发送飞书通知失败: " + err.Error()
		log("错误: " + errMsg)
		return
	}

	log("成功发送到飞书")
}

// Health 处理 GET /api/v1/health
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}
