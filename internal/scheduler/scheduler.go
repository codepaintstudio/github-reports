package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github-reports/internal/config"
	"github-reports/internal/github"
	"github-reports/internal/llm"
	"github-reports/internal/notifier"
	"github-reports/internal/reporter"

	"github.com/robfig/cron/v3"
)

// Scheduler manages scheduled report generation
type Scheduler struct {
	cron      *cron.Cron
	config    *config.Config
	notifiers []notifier.Notifier
}

// NewScheduler creates a new scheduler
func NewScheduler(cfg *config.Config) (*Scheduler, error) {
	var notifiers []notifier.Notifier

	if cfg.Notifiers.WeChat.Enabled {
		notifiers = append(notifiers, notifier.NewWeChatNotifier(cfg.Notifiers.WeChat.WebhookURL))
	}

	if cfg.Notifiers.Feishu.Enabled {
		notifiers = append(notifiers, notifier.NewFeishuNotifier(cfg.Notifiers.Feishu.WebhookURL))
	}

	return &Scheduler{
		cron:      cron.New(),
		config:    cfg,
		notifiers: notifiers,
	}, nil
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	if !s.config.Scheduler.Enabled {
		log.Println("Scheduler is disabled")
		return nil
	}

	// Add scheduled job for each GitHub token that has a username
	for _, token := range s.config.GitHub.Tokens {
		if token.Username == "" {
			log.Printf("Skipping token without username for scheduled tasks")
			continue
		}

		token := token // Capture loop variable
		_, err := s.cron.AddFunc(s.config.Scheduler.Cron, func() {
			s.runScheduledReport(token)
		})
		if err != nil {
			return fmt.Errorf("failed to add cron job: %w", err)
		}
		log.Printf("Scheduled report task added for user: %s", token.Username)
	}

	s.cron.Start()
	log.Printf("Scheduler started with cron: %s", s.config.Scheduler.Cron)
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
	log.Println("Scheduler stopped")
}

// runScheduledReport runs a scheduled report generation
func (s *Scheduler) runScheduledReport(token config.GitHubToken) {
	ctx := context.Background()

	log.Printf("Generating scheduled report for user: %s", token.Username)

	// Create clients
	githubClient := github.NewClient(token.Token)
	llmClient, err := llm.NewClient(s.config.LLM)
	if err != nil {
		log.Printf("Failed to create LLM client: %v", err)
		return
	}

	// Create reporter
	rep := reporter.NewReporter(githubClient, llmClient)

	// Calculate time range
	until := time.Now()
	since := until.Add(-s.config.Scheduler.DefaultSince)

	// Generate report
	report, err := rep.GenerateReport(ctx, token.Username, since, until)
	if err != nil {
		log.Printf("Failed to generate report for %s: %v", token.Username, err)
		return
	}

	log.Printf("Report generated for %s", token.Username)

	// Send notifications
	for _, n := range s.notifiers {
		if err := n.Send(ctx, report); err != nil {
			log.Printf("Failed to send notification: %v", err)
		} else {
			log.Printf("Notification sent successfully for %s", token.Username)
		}
	}
}
