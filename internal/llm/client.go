package llm

import (
	"context"
	"fmt"

	"github-reports/internal/config"
)

// Client is the interface for LLM clients
type Client interface {
	GenerateReport(ctx context.Context, activityData string) (string, error)
}

// NewClient creates a new LLM client based on provider
func NewClient(cfg config.LLMConfig) (Client, error) {
	switch cfg.Provider {
	case "openai":
		return NewOpenAIClient(cfg), nil
	case "claude":
		return NewClaudeClient(cfg), nil
	case "deepseek":
		// DeepSeek 使用 OpenAI 兼容接口
		return NewOpenAIClient(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}
