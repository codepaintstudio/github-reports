package llm

import (
	"context"
	"fmt"

	"github-reports/internal/config"
)

// Client 是 LLM 客户端的接口
type Client interface {
	GenerateReport(ctx context.Context, activityData string) (string, error)
	ExtractGitHubUsername(ctx context.Context, content string) (string, error)
}

// NewClient 根据提供商创建一个新的 LLM 客户端
func NewClient(cfg config.LLMConfig) (Client, error) {
	if cfg.Provider != "deepseek" {
		return nil, fmt.Errorf("only deepseek provider is supported, got: %s", cfg.Provider)
	}
	return NewDeepSeekClient(cfg), nil
}
