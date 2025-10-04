package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github-reports/internal/config"
)

// DeepSeekClient 为 DeepSeek (兼容 OpenAI API) 实现 Client
type DeepSeekClient struct {
	config config.LLMConfig
	client *http.Client
}

// NewDeepSeekClient 创建一个新的 DeepSeek 客户端
func NewDeepSeekClient(cfg config.LLMConfig) *DeepSeekClient {
	return &DeepSeekClient{
		config: cfg,
		client: &http.Client{
			Timeout: 60 * time.Second, // 设置 60 秒超时
		},
	}
}

type deepseekRequest struct {
	Model    string            `json:"model"`
	Messages []deepseekMessage `json:"messages"`
}

type deepseekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepseekResponse struct {
	Choices []struct {
		Message deepseekMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// GenerateReport 使用 DeepSeek 生成报告
func (c *DeepSeekClient) GenerateReport(ctx context.Context, activityData string) (string, error) {
	// 直接将活动数据作为用户输入，使用固定的系统提示词
	systemPrompt := `你是一名「个人 GitHub 技术动态总结助手」。请根据以下 GitHub 活动数据，生成一份**简洁、技术性强、周报风格**的总结。

### 输出要求

1. **按项目分组**

   * 每个项目仅用 **2–3 行**概述本人的主要技术进展。
   * 突出成果、优化点或解决的技术难题。
   * 不要逐条列出 commit/PR 详情，只提最核心的工作。

2. **总体技术分析**

   * 总结近期的主要技术方向（如新功能开发、性能优化、架构演进）。
   * 简要点出代表性的难题及解决方式。
   * 用数据支撑（commit 数、代码增删行数），但不要展开逐条解释。

3. **风格要求**

   * 输出要**高度凝练**，像周会上口头汇报一样简明。
   * 重点在「做了什么」和「技术价值」，而不是「做了多少」。
   * 避免冗长的 commit 描述，保持报告感。

### 输出模板

---

# [{username}](https://github.com/{username}) 的 GitHub 一周动态分析

## 项目A
- 核心进展 1（简要）
- 核心进展 2（简要）

## 项目B
- 核心进展 1（简要）
- 核心进展 2（简要）

## 总体技术分析
近期主要精力集中在 {方向概述}。  
代表性难题：{难题简述 + 解决方式}。  
本期共提交 {C} 个 commits，新增 {A} 行代码，删除 {D} 行代码。

---

### 输入数据
---
GitHub 活动数据：
{activity_data}
---
`

	userPrompt := activityData

	return c.completeWithSystem(ctx, systemPrompt, userPrompt)
}

// ExtractGitHubUsername 使用 LLM 从内容中提取 GitHub 用户名
func (c *DeepSeekClient) ExtractGitHubUsername(ctx context.Context, content string) (string, error) {
	prompt := fmt.Sprintf(`从以下内容中提取 GitHub 用户名。只返回用户名，不要有任何其他文字。

内容：
%s

GitHub 用户名：`, content)

	return c.complete(ctx, prompt)
}

// completeWithSystem 使用系统提示词和用户提示词发起补全请求
func (c *DeepSeekClient) completeWithSystem(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	baseURL := c.config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}

	reqBody := deepseekRequest{
		Model: c.config.Model,
		Messages: []deepseekMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}

	return c.doRequest(ctx, baseURL, reqBody)
}

// complete 是用于发起补全请求的辅助方法
func (c *DeepSeekClient) complete(ctx context.Context, prompt string) (string, error) {
	baseURL := c.config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}

	reqBody := deepseekRequest{
		Model: c.config.Model,
		Messages: []deepseekMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	return c.doRequest(ctx, baseURL, reqBody)
}

// doRequest 执行实际的 HTTP 请求
func (c *DeepSeekClient) doRequest(ctx context.Context, baseURL string, reqBody deepseekRequest) (string, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var deepseekResp deepseekResponse
	if err := json.Unmarshal(body, &deepseekResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if deepseekResp.Error != nil {
		return "", fmt.Errorf("DeepSeek API error: %s", deepseekResp.Error.Message)
	}

	if len(deepseekResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return deepseekResp.Choices[0].Message.Content, nil
}
