package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig     `mapstructure:"server"`
	GitHub    GitHubConfig     `mapstructure:"github"`
	LLM       LLMConfig        `mapstructure:"llm"`
	Scheduler SchedulerConfig  `mapstructure:"scheduler"`
	Notifiers NotifiersConfig  `mapstructure:"notifiers"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type GitHubConfig struct {
	Tokens []GitHubToken `mapstructure:"tokens"`
}

type GitHubToken struct {
	Token    string `mapstructure:"token"`
	Username string `mapstructure:"username"`
}

type LLMConfig struct {
	Provider       string `mapstructure:"provider"` // openai, claude, custom
	APIKey         string `mapstructure:"api_key"`
	Model          string `mapstructure:"model"`
	PromptTemplate string `mapstructure:"prompt_template"`
	BaseURL        string `mapstructure:"base_url"` // optional, for custom endpoints
}

type SchedulerConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	Cron         string        `mapstructure:"cron"`
	DefaultSince time.Duration `mapstructure:"default_since"` // default: 7 days
}

type NotifiersConfig struct {
	WeChat WeChatConfig `mapstructure:"wechat"`
	Feishu FeishuConfig `mapstructure:"feishu"`
}

type WeChatConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	WebhookURL string `mapstructure:"webhook_url"`
}

type FeishuConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	WebhookURL string `mapstructure:"webhook_url"`
}

// Load loads configuration from file
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
	v.SetDefault("scheduler.enabled", false)
	v.SetDefault("scheduler.cron", "0 15 * * 5") // Friday 3PM
	v.SetDefault("scheduler.default_since", 7*24*time.Hour) // 7 days
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

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.GitHub.Tokens) == 0 {
		return fmt.Errorf("at least one GitHub token is required")
	}

	if c.LLM.APIKey == "" {
		return fmt.Errorf("LLM API key is required")
	}

	if c.LLM.Provider != "openai" && c.LLM.Provider != "claude" && c.LLM.Provider != "deepseek" && c.LLM.Provider != "custom" {
		return fmt.Errorf("invalid LLM provider: %s", c.LLM.Provider)
	}

	return nil
}
