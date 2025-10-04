package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	GitHub    GitHubConfig    `mapstructure:"github"`
	LLM       LLMConfig       `mapstructure:"llm"`
	Notifiers NotifiersConfig `mapstructure:"notifiers"`
	Webhook   WebhookConfig   `mapstructure:"webhook"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type WebhookConfig struct {
	Token string `mapstructure:"token"`
}

type GitHubConfig struct {
	Tokens []GitHubToken `mapstructure:"tokens"`
}

type GitHubToken struct {
	Token    string `mapstructure:"token"`
	Username string `mapstructure:"username"` // 可选：如果不指定，则允许查询任何用户
}

type LLMConfig struct {
	Provider       string `mapstructure:"provider"` // openai, claude, custom
	APIKey         string `mapstructure:"api_key"`
	Model          string `mapstructure:"model"`
	PromptTemplate string `mapstructure:"prompt_template"`
	BaseURL        string `mapstructure:"base_url"` // 可选，用于自定义端点
}

type NotifiersConfig struct {
	Feishu FeishuConfig `mapstructure:"feishu"`
}

type FeishuConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	WebhookURL string `mapstructure:"webhook_url"`
}

// Load 从文件加载配置
func Load(configPath string) (*Config, error) {
	v := viper.New()

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// Set defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("llm.provider", "deepseek")
	v.SetDefault("llm.model", "deepseek-chat")
	v.SetDefault("llm.base_url", "https://api.deepseek.com/v1")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if len(c.GitHub.Tokens) == 0 {
		return fmt.Errorf("at least one GitHub token is required")
	}

	if c.LLM.APIKey == "" {
		return fmt.Errorf("LLM API key is required")
	}

	if c.LLM.Provider != "deepseek" {
		return fmt.Errorf("only deepseek provider is supported, got: %s", c.LLM.Provider)
	}

	if c.Webhook.Token == "" {
		return fmt.Errorf("webhook token is required")
	}

	return nil
}
