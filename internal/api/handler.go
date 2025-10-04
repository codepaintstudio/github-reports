package api

import (
	"net/http"
	"time"

	"github-reports/internal/config"
	"github-reports/internal/github"
	"github-reports/internal/llm"
	"github-reports/internal/notifier"
	"github-reports/internal/reporter"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests
type Handler struct {
	config    *config.Config
	notifiers []notifier.Notifier
}

// NewHandler creates a new API handler
func NewHandler(cfg *config.Config) *Handler {
	var notifiers []notifier.Notifier

	if cfg.Notifiers.WeChat.Enabled {
		notifiers = append(notifiers, notifier.NewWeChatNotifier(cfg.Notifiers.WeChat.WebhookURL))
	}

	if cfg.Notifiers.Feishu.Enabled {
		notifiers = append(notifiers, notifier.NewFeishuNotifier(cfg.Notifiers.Feishu.WebhookURL))
	}

	return &Handler{
		config:    cfg,
		notifiers: notifiers,
	}
}

// GenerateReportRequest represents the request body for generating a report
type GenerateReportRequest struct {
	Username string `json:"username" binding:"required"`
	Since    string `json:"since"`  // RFC3339 format
	Until    string `json:"until"`  // RFC3339 format
	Notify   bool   `json:"notify"` // Whether to send notifications
}

// GenerateReportResponse represents the response for generating a report
type GenerateReportResponse struct {
	Username string `json:"username"`
	Report   string `json:"report"`
	Since    string `json:"since"`
	Until    string `json:"until"`
}

// GenerateReport handles POST /api/v1/reports/generate
func (h *Handler) GenerateReport(c *gin.Context) {
	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log := func(message string) {
		println("[GenerateReport]", req.Username, "-", message)
	}

	log("开始生成周报")

	// Find GitHub token for the user
	var token string
	for _, t := range h.config.GitHub.Tokens {
		if t.Username == req.Username {
			token = t.Token
			break
		}
	}

	if token == "" {
		log("错误: 未找到 GitHub token")
		c.JSON(http.StatusNotFound, gin.H{"error": "GitHub token not found for user"})
		return
	}

	log("已找到 GitHub token")

	// Parse time range
	var since, until time.Time
	var err error

	if req.Since != "" {
		since, err = time.Parse(time.RFC3339, req.Since)
		if err != nil {
			log("错误: 无效的开始时间格式")
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid since time format"})
			return
		}
	} else {
		since = time.Now().Add(-7 * 24 * time.Hour) // Default to 7 days ago
	}

	if req.Until != "" {
		until, err = time.Parse(time.RFC3339, req.Until)
		if err != nil {
			log("错误: 无效的结束时间格式")
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid until time format"})
			return
		}
	} else {
		until = time.Now()
	}

	log("时间范围: " + since.Format("2006-01-02") + " ~ " + until.Format("2006-01-02"))

	// Create clients
	log("创建 GitHub 客户端...")
	githubClient := github.NewClient(token)

	log("创建 LLM 客户端...")
	llmClient, err := llm.NewClient(h.config.LLM)
	if err != nil {
		log("错误: 创建 LLM 客户端失败 - " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create LLM client"})
		return
	}

	// Create reporter and generate report
	log("开始拉取 GitHub 活动数据...")
	rep := reporter.NewReporter(githubClient, llmClient)
	report, err := rep.GenerateReport(c.Request.Context(), req.Username, since, until)
	if err != nil {
		log("错误: 生成报告失败 - " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log("周报生成成功")

	// Send notifications if requested
	if req.Notify {
		log("开始发送通知...")
		for _, n := range h.notifiers {
			if err := n.Send(c.Request.Context(), report); err != nil {
				log("警告: 发送通知失败 - " + err.Error())
				// Log error but don't fail the request
				c.JSON(http.StatusOK, gin.H{
					"warning": "failed to send notification: " + err.Error(),
				})
			} else {
				log("通知发送成功")
			}
		}
	}

	log("完成")

	c.JSON(http.StatusOK, GenerateReportResponse{
		Username: req.Username,
		Report:   report,
		Since:    since.Format(time.RFC3339),
		Until:    until.Format(time.RFC3339),
	})
}

// GenerateAllReports handles POST /api/v1/reports/generate-all
func (h *Handler) GenerateAllReports(c *gin.Context) {
	var results []map[string]interface{}

	println("[GenerateAllReports] 开始批量生成周报")

	// Default time range: last 7 days
	until := time.Now()
	since := until.Add(-7 * 24 * time.Hour)

	println("[GenerateAllReports] 时间范围:", since.Format("2006-01-02"), "~", until.Format("2006-01-02"))
	println("[GenerateAllReports] 配置的用户数量:", len(h.config.GitHub.Tokens))

	// Generate report for each configured user
	for i, tokenConfig := range h.config.GitHub.Tokens {
		username := tokenConfig.Username
		println("[GenerateAllReports] [", i+1, "/", len(h.config.GitHub.Tokens), "] 开始处理用户:", username)

		result := map[string]interface{}{
			"username": username,
		}

		// Create clients
		println("[GenerateAllReports]", username, "- 创建 GitHub 客户端...")
		githubClient := github.NewClient(tokenConfig.Token)

		println("[GenerateAllReports]", username, "- 创建 LLM 客户端...")
		llmClient, err := llm.NewClient(h.config.LLM)
		if err != nil {
			errMsg := "failed to create LLM client: " + err.Error()
			println("[GenerateAllReports]", username, "- 错误:", errMsg)
			result["error"] = errMsg
			results = append(results, result)
			continue
		}

		// Generate report
		println("[GenerateAllReports]", username, "- 开始拉取 GitHub 活动数据...")
		rep := reporter.NewReporter(githubClient, llmClient)
		report, err := rep.GenerateReport(c.Request.Context(), username, since, until)
		if err != nil {
			errMsg := err.Error()
			println("[GenerateAllReports]", username, "- 错误:", errMsg)
			result["error"] = errMsg
			results = append(results, result)
			continue
		}

		println("[GenerateAllReports]", username, "- 周报生成成功")

		result["report"] = report
		result["since"] = since.Format(time.RFC3339)
		result["until"] = until.Format(time.RFC3339)
		result["success"] = true

		// Send notifications
		if len(h.notifiers) > 0 {
			println("[GenerateAllReports]", username, "- 开始发送通知...")
			for _, n := range h.notifiers {
				if err := n.Send(c.Request.Context(), report); err != nil {
					errMsg := err.Error()
					println("[GenerateAllReports]", username, "- 通知发送失败:", errMsg)
					result["notification_error"] = errMsg
				} else {
					println("[GenerateAllReports]", username, "- 通知发送成功")
				}
			}
		}

		results = append(results, result)
		println("[GenerateAllReports]", username, "- 完成")
	}

	println("[GenerateAllReports] 全部完成，成功:", len(results), "个用户")

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(results),
	})
}

// Health handles GET /api/v1/health
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}
