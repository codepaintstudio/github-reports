package github

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v60/github"
)

// Fetcher fetches GitHub activities
type Fetcher struct {
	client *Client
}

// NewFetcher creates a new Fetcher
func NewFetcher(client *Client) *Fetcher {
	return &Fetcher{client: client}
}

// FetchActivities fetches all activities for a user within the time range
func (f *Fetcher) FetchActivities(ctx context.Context, username string, since, until time.Time) (*UserActivity, error) {
	activity := &UserActivity{
		Username: username,
		Since:    since,
		Until:    until,
	}

	println("[Fetcher]", username, "- 正在拉取用户活动...")

	// Fetch all events from user's timeline (single API call)
	commits, prs, issues, reviews, err := f.fetchFromEvents(ctx, username, since, until)
	if err != nil {
		println("[Fetcher]", username, "- 拉取活动失败:", err.Error())
		return nil, fmt.Errorf("failed to fetch activities: %w", err)
	}

	activity.Commits = commits
	activity.PullRequests = prs
	activity.Issues = issues
	activity.Reviews = reviews

	println("[Fetcher]", username, "- 找到", len(commits), "个 Commits,", len(prs), "个 Pull Requests,", len(issues), "个 Issues,", len(reviews), "个 Code Reviews")

	return activity, nil
}

// fetchFromEvents fetches all activities from user's event timeline in a single pass
func (f *Fetcher) fetchFromEvents(ctx context.Context, username string, since, until time.Time) ([]CommitInfo, []PullRequestInfo, []IssueInfo, []ReviewInfo, error) {
	var commits []CommitInfo
	var prs []PullRequestInfo
	var issues []IssueInfo
	var reviews []ReviewInfo

	opts := &github.ListOptions{PerPage: 100}
	prMap := make(map[string]bool)   // Track PRs to avoid duplicates
	issueMap := make(map[string]bool) // Track Issues to avoid duplicates

	for {
		events, resp, err := f.client.client.Activity.ListEventsPerformedByUser(ctx, username, false, opts)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		for _, event := range events {
			// Check if event is within time range
			if event.CreatedAt == nil {
				continue
			}
			if event.CreatedAt.Before(since) || event.CreatedAt.After(until) {
				continue
			}

			eventType := getStringValue(event.Type)

			// Process PushEvent for commits
			if eventType == "PushEvent" {
				payload, err := event.ParsePayload()
				if err != nil {
					continue
				}

				pushPayload, ok := payload.(*github.PushEvent)
				if !ok {
					continue
				}

				repo := ""
				if event.Repo != nil && event.Repo.Name != nil {
					repo = *event.Repo.Name
				}

				for _, commit := range pushPayload.Commits {
					if commit.SHA == nil || commit.Message == nil {
						continue
					}

					authorName := getStringValue(commit.Author.Name)

					// Fetch detailed commit info to verify authorship
					if repo != "" {
						owner, repoName := parseRepoName(repo)
						if owner != "" && repoName != "" {
							c, _, err := f.client.client.Repositories.GetCommit(ctx, owner, repoName, *commit.SHA, nil)
							if err == nil {
								// Verify the commit author matches the target user
								isAuthor := false
								commitAuthor := ""
								if c.Author != nil && c.Author.Login != nil {
									commitAuthor = *c.Author.Login
									if *c.Author.Login == username {
										isAuthor = true
									}
								}
								// Also check committer in case author is different
								if c.Committer != nil && c.Committer.Login != nil && *c.Committer.Login == username {
									isAuthor = true
								}

								// Skip if not authored by target user
								if !isAuthor {
									println("[Fetcher] 跳过非目标用户的 commit:", (*commit.SHA)[:7], "作者:", commitAuthor, "目标用户:", username)
									continue
								}

								// Get stats
								additions, deletions := 0, 0
								if c.Stats != nil {
									if c.Stats.Additions != nil {
										additions = *c.Stats.Additions
									}
									if c.Stats.Deletions != nil {
										deletions = *c.Stats.Deletions
									}
								}

								commits = append(commits, CommitInfo{
									SHA:       *commit.SHA,
									Message:   *commit.Message,
									Repo:      repo,
									URL:       getStringValue(commit.URL),
									Author:    authorName,
									Date:      event.CreatedAt.Time,
									Additions: additions,
									Deletions: deletions,
								})
							}
						}
					}
				}
			}

			// Process PullRequestEvent
			if eventType == "PullRequestEvent" {
				payload, err := event.ParsePayload()
				if err != nil {
					continue
				}

				prPayload, ok := payload.(*github.PullRequestEvent)
				if !ok || prPayload.PullRequest == nil {
					continue
				}

				pr := prPayload.PullRequest

				// Only process PRs created by the target user
				if pr.User == nil || pr.User.Login == nil || *pr.User.Login != username {
					continue
				}

				repo := ""
				if event.Repo != nil && event.Repo.Name != nil {
					repo = *event.Repo.Name
				}

				prKey := fmt.Sprintf("%s#%d", repo, getIntValue(pr.Number))
				if prMap[prKey] {
					continue // Skip duplicates
				}
				prMap[prKey] = true

				prInfo := PullRequestInfo{
					Number:    getIntValue(pr.Number),
					Title:     getStringValue(pr.Title),
					URL:       getStringValue(pr.HTMLURL),
					State:     getStringValue(pr.State),
					Repo:      repo,
					CreatedAt: getTimeValue(pr.CreatedAt),
					Comments:  getIntValue(pr.Comments),
					Additions: getIntValue(pr.Additions),
					Deletions: getIntValue(pr.Deletions),
				}

				if pr.MergedAt != nil {
					prInfo.MergedAt = getTimePointer(pr.MergedAt)
				}

				prs = append(prs, prInfo)
			}

			// Process IssuesEvent
			if eventType == "IssuesEvent" {
				payload, err := event.ParsePayload()
				if err != nil {
					continue
				}

				issuePayload, ok := payload.(*github.IssuesEvent)
				if !ok || issuePayload.Issue == nil {
					continue
				}

				issue := issuePayload.Issue

				// Only process issues created by the target user
				if issue.User == nil || issue.User.Login == nil || *issue.User.Login != username {
					continue
				}

				// Skip if it's actually a PR
				if issue.PullRequestLinks != nil {
					continue
				}

				repo := ""
				if event.Repo != nil && event.Repo.Name != nil {
					repo = *event.Repo.Name
				}

				issueKey := fmt.Sprintf("%s#%d", repo, getIntValue(issue.Number))
				if issueMap[issueKey] {
					continue // Skip duplicates
				}
				issueMap[issueKey] = true

				issueInfo := IssueInfo{
					Number:    getIntValue(issue.Number),
					Title:     getStringValue(issue.Title),
					URL:       getStringValue(issue.HTMLURL),
					State:     getStringValue(issue.State),
					Repo:      repo,
					CreatedAt: getTimeValue(issue.CreatedAt),
					Comments:  getIntValue(issue.Comments),
				}

				if issue.ClosedAt != nil {
					issueInfo.ClosedAt = getTimePointer(issue.ClosedAt)
				}

				issues = append(issues, issueInfo)
			}

			// Process PullRequestReviewEvent
			if eventType == "PullRequestReviewEvent" {
				payload, err := event.ParsePayload()
				if err != nil {
					continue
				}

				reviewPayload, ok := payload.(*github.PullRequestReviewEvent)
				if !ok {
					continue
				}

				// Verify the review is by the target user
				if reviewPayload.Review != nil && reviewPayload.Review.User != nil && reviewPayload.Review.User.Login != nil {
					if *reviewPayload.Review.User.Login != username {
						continue
					}
				} else {
					continue
				}

				repo := ""
				if event.Repo != nil && event.Repo.Name != nil {
					repo = *event.Repo.Name
				}

				review := ReviewInfo{
					Repo:      repo,
					CreatedAt: event.CreatedAt.Time,
				}

				if reviewPayload.Review != nil {
					review.State = getStringValue(reviewPayload.Review.State)
					review.URL = getStringValue(reviewPayload.Review.HTMLURL)
				}

				if reviewPayload.PullRequest != nil {
					review.PRNumber = getIntValue(reviewPayload.PullRequest.Number)
					review.PRTitle = getStringValue(reviewPayload.PullRequest.Title)
				}

				reviews = append(reviews, review)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return commits, prs, issues, reviews, nil
}

// Helper functions
func parseRepoName(fullName string) (owner, repo string) {
	for i, c := range fullName {
		if c == '/' {
			return fullName[:i], fullName[i+1:]
		}
	}
	return "", ""
}

func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getIntValue(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func getTimeValue(t *github.Timestamp) time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.Time
}

func getTimePointer(t *github.Timestamp) *time.Time {
	if t == nil {
		return nil
	}
	result := t.Time
	return &result
}
