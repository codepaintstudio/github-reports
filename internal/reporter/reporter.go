package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github-reports/internal/github"
	"github-reports/internal/llm"
)

// Reporter generates reports from GitHub activities
type Reporter struct {
	githubClient *github.Client
	fetcher      *github.Fetcher
	llmClient    llm.Client
}

// NewReporter creates a new Reporter
func NewReporter(githubClient *github.Client, llmClient llm.Client) *Reporter {
	return &Reporter{
		githubClient: githubClient,
		fetcher:      github.NewFetcher(githubClient),
		llmClient:    llmClient,
	}
}

// GenerateReport generates a weekly report for a user
func (r *Reporter) GenerateReport(ctx context.Context, username string, since, until time.Time) (string, error) {
	// Fetch GitHub activities
	println("[Reporter]", username, "- 正在拉取 GitHub 活动数据...")
	activity, err := r.fetcher.FetchActivities(ctx, username, since, until)
	if err != nil {
		println("[Reporter]", username, "- 拉取活动数据失败:", err.Error())
		return "", fmt.Errorf("failed to fetch activities: %w", err)
	}

	stats := activity.Statistics()
	println("[Reporter]", username, "- 数据统计: Commits:", stats["total_commits"], "PRs:", stats["total_prs"], "Issues:", stats["total_issues"], "Reviews:", stats["total_reviews"])

	// Format activity data for LLM
	println("[Reporter]", username, "- 正在格式化活动数据...")
	activityData, err := r.formatActivityData(activity)
	if err != nil {
		println("[Reporter]", username, "- 格式化数据失败:", err.Error())
		return "", fmt.Errorf("failed to format activity data: %w", err)
	}

	// Generate report using LLM
	println("[Reporter]", username, "- 正在调用 LLM 生成周报...")
	report, err := r.llmClient.GenerateReport(ctx, activityData)
	if err != nil {
		println("[Reporter]", username, "- LLM 生成失败:", err.Error())
		return "", fmt.Errorf("failed to generate report: %w", err)
	}

	println("[Reporter]", username, "- LLM 生成完成，报告长度:", len(report), "字符")

	return report, nil
}

// formatActivityData formats activity data as a structured string for LLM
func (r *Reporter) formatActivityData(activity *github.UserActivity) (string, error) {
	data := map[string]interface{}{
		"username":      activity.Username,
		"time_range":    fmt.Sprintf("%s ~ %s", activity.Since.Format("2006-01-02"), activity.Until.Format("2006-01-02")),
		"statistics":    activity.Statistics(),
		"commits":       r.formatCommits(activity.Commits),
		"pull_requests": r.formatPullRequests(activity.PullRequests),
		"issues":        r.formatIssues(activity.Issues),
		"reviews":       r.formatReviews(activity.Reviews),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

func (r *Reporter) formatCommits(commits []github.CommitInfo) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(commits))
	for _, c := range commits {
		result = append(result, map[string]interface{}{
			"sha":       c.SHA[:7], // Short SHA
			"message":   c.Message,
			"repo":      c.Repo,
			"author":    c.Author,
			"date":      c.Date.Format("2006-01-02 15:04"),
			"additions": c.Additions,
			"deletions": c.Deletions,
		})
	}
	return result
}

func (r *Reporter) formatPullRequests(prs []github.PullRequestInfo) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(prs))
	for _, pr := range prs {
		prData := map[string]interface{}{
			"number":    pr.Number,
			"title":     pr.Title,
			"repo":      pr.Repo,
			"state":     pr.State,
			"url":       pr.URL,
			"created":   pr.CreatedAt.Format("2006-01-02"),
			"additions": pr.Additions,
			"deletions": pr.Deletions,
			"comments":  pr.Comments,
		}
		if pr.MergedAt != nil {
			prData["merged"] = pr.MergedAt.Format("2006-01-02")
		}
		result = append(result, prData)
	}
	return result
}

func (r *Reporter) formatIssues(issues []github.IssueInfo) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(issues))
	for _, issue := range issues {
		issueData := map[string]interface{}{
			"number":   issue.Number,
			"title":    issue.Title,
			"repo":     issue.Repo,
			"state":    issue.State,
			"url":      issue.URL,
			"created":  issue.CreatedAt.Format("2006-01-02"),
			"comments": issue.Comments,
		}
		if issue.ClosedAt != nil {
			issueData["closed"] = issue.ClosedAt.Format("2006-01-02")
		}
		result = append(result, issueData)
	}
	return result
}

func (r *Reporter) formatReviews(reviews []github.ReviewInfo) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(reviews))
	for _, review := range reviews {
		result = append(result, map[string]interface{}{
			"pr_number": review.PRNumber,
			"pr_title":  review.PRTitle,
			"repo":      review.Repo,
			"state":     review.State,
			"url":       review.URL,
			"created":   review.CreatedAt.Format("2006-01-02"),
		})
	}
	return result
}
