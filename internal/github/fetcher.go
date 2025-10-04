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

	// Fetch commits
	println("[Fetcher]", username, "- 正在拉取 Commits...")
	commits, err := f.fetchCommits(ctx, username, since, until)
	if err != nil {
		println("[Fetcher]", username, "- 拉取 Commits 失败:", err.Error())
		return nil, fmt.Errorf("failed to fetch commits: %w", err)
	}
	activity.Commits = commits
	println("[Fetcher]", username, "- 找到", len(commits), "个 Commits")

	// Fetch pull requests
	println("[Fetcher]", username, "- 正在拉取 Pull Requests...")
	prs, err := f.fetchPullRequests(ctx, username, since, until)
	if err != nil {
		println("[Fetcher]", username, "- 拉取 Pull Requests 失败:", err.Error())
		return nil, fmt.Errorf("failed to fetch pull requests: %w", err)
	}
	activity.PullRequests = prs
	println("[Fetcher]", username, "- 找到", len(prs), "个 Pull Requests")

	// Fetch issues
	println("[Fetcher]", username, "- 正在拉取 Issues...")
	issues, err := f.fetchIssues(ctx, username, since, until)
	if err != nil {
		println("[Fetcher]", username, "- 拉取 Issues 失败:", err.Error())
		return nil, fmt.Errorf("failed to fetch issues: %w", err)
	}
	activity.Issues = issues
	println("[Fetcher]", username, "- 找到", len(issues), "个 Issues")

	// Fetch reviews
	println("[Fetcher]", username, "- 正在拉取 Code Reviews...")
	reviews, err := f.fetchReviews(ctx, username, since, until)
	if err != nil {
		println("[Fetcher]", username, "- 拉取 Code Reviews 失败:", err.Error())
		return nil, fmt.Errorf("failed to fetch reviews: %w", err)
	}
	activity.Reviews = reviews
	println("[Fetcher]", username, "- 找到", len(reviews), "个 Code Reviews")

	return activity, nil
}

// fetchCommits fetches commits from user's events
func (f *Fetcher) fetchCommits(ctx context.Context, username string, since, until time.Time) ([]CommitInfo, error) {
	var commits []CommitInfo
	opts := &github.ListOptions{PerPage: 100}

	for {
		events, resp, err := f.client.client.Activity.ListEventsPerformedByUser(ctx, username, false, opts)
		if err != nil {
			return nil, err
		}

		for _, event := range events {
			// Check if event is within time range
			if event.CreatedAt == nil {
				continue
			}
			if event.CreatedAt.Before(since) || event.CreatedAt.After(until) {
				continue
			}

			// Process PushEvent
			if *event.Type == "PushEvent" {
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

					// Skip commits not authored by the target user
					// Check both author name and login
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
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return commits, nil
}

// fetchPullRequests fetches pull requests created by the user
func (f *Fetcher) fetchPullRequests(ctx context.Context, username string, since, until time.Time) ([]PullRequestInfo, error) {
	var prs []PullRequestInfo

	// Search for PRs created by the user
	query := fmt.Sprintf("author:%s created:%s..%s",
		username,
		since.Format("2006-01-02"),
		until.Format("2006-01-02"),
	)

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		result, resp, err := f.client.client.Search.Issues(ctx, query, opts)
		if err != nil {
			return nil, err
		}

		for _, issue := range result.Issues {
			if issue.PullRequestLinks == nil {
				continue
			}

			pr := PullRequestInfo{
				Number:    getIntValue(issue.Number),
				Title:     getStringValue(issue.Title),
				URL:       getStringValue(issue.HTMLURL),
				State:     getStringValue(issue.State),
				CreatedAt: getTimeValue(issue.CreatedAt),
				Comments:  getIntValue(issue.Comments),
			}

			if issue.Repository != nil && issue.Repository.FullName != nil {
				pr.Repo = *issue.Repository.FullName
			}

			if issue.ClosedAt != nil {
				pr.MergedAt = getTimePointer(issue.ClosedAt)
			}

			// Fetch detailed PR info for additions/deletions
			if pr.Repo != "" {
				owner, repoName := parseRepoName(pr.Repo)
				if owner != "" && repoName != "" {
					prDetail, _, err := f.client.client.PullRequests.Get(ctx, owner, repoName, pr.Number)
					if err == nil {
						pr.Additions = getIntValue(prDetail.Additions)
						pr.Deletions = getIntValue(prDetail.Deletions)
						if prDetail.MergedAt != nil {
							pr.MergedAt = getTimePointer(prDetail.MergedAt)
						}
					}
				}
			}

			prs = append(prs, pr)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return prs, nil
}

// fetchIssues fetches issues created by the user
func (f *Fetcher) fetchIssues(ctx context.Context, username string, since, until time.Time) ([]IssueInfo, error) {
	var issues []IssueInfo

	query := fmt.Sprintf("author:%s type:issue created:%s..%s",
		username,
		since.Format("2006-01-02"),
		until.Format("2006-01-02"),
	)

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		result, resp, err := f.client.client.Search.Issues(ctx, query, opts)
		if err != nil {
			return nil, err
		}

		for _, issue := range result.Issues {
			if issue.PullRequestLinks != nil {
				continue // Skip PRs
			}

			info := IssueInfo{
				Number:    getIntValue(issue.Number),
				Title:     getStringValue(issue.Title),
				URL:       getStringValue(issue.HTMLURL),
				State:     getStringValue(issue.State),
				CreatedAt: getTimeValue(issue.CreatedAt),
				Comments:  getIntValue(issue.Comments),
			}

			if issue.Repository != nil && issue.Repository.FullName != nil {
				info.Repo = *issue.Repository.FullName
			}

			if issue.ClosedAt != nil {
				info.ClosedAt = getTimePointer(issue.ClosedAt)
			}

			issues = append(issues, info)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return issues, nil
}

// fetchReviews fetches code reviews by the user
func (f *Fetcher) fetchReviews(ctx context.Context, username string, since, until time.Time) ([]ReviewInfo, error) {
	var reviews []ReviewInfo
	opts := &github.ListOptions{PerPage: 100}

	for {
		events, resp, err := f.client.client.Activity.ListEventsPerformedByUser(ctx, username, false, opts)
		if err != nil {
			return nil, err
		}

		for _, event := range events {
			if event.CreatedAt == nil {
				continue
			}
			if event.CreatedAt.Before(since) || event.CreatedAt.After(until) {
				continue
			}

			if *event.Type == "PullRequestReviewEvent" {
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

	return reviews, nil
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
