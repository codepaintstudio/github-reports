package github

import "time"

// UserActivity represents all GitHub activities for a user
type UserActivity struct {
	Username     string
	Since        time.Time
	Until        time.Time
	Commits      []CommitInfo
	PullRequests []PullRequestInfo
	Issues       []IssueInfo
	Reviews      []ReviewInfo
}

// CommitInfo represents a commit
type CommitInfo struct {
	SHA       string
	Message   string
	Repo      string
	URL       string
	Author    string
	Date      time.Time
	Additions int
	Deletions int
}

// PullRequestInfo represents a pull request
type PullRequestInfo struct {
	Number    int
	Title     string
	Repo      string
	URL       string
	State     string // open, closed, merged
	CreatedAt time.Time
	MergedAt  *time.Time
	Additions int
	Deletions int
	Comments  int
}

// IssueInfo represents an issue
type IssueInfo struct {
	Number    int
	Title     string
	Repo      string
	URL       string
	State     string // open, closed
	CreatedAt time.Time
	ClosedAt  *time.Time
	Comments  int
}

// ReviewInfo represents a code review
type ReviewInfo struct {
	PRNumber  int
	PRTitle   string
	Repo      string
	URL       string
	State     string // APPROVED, CHANGES_REQUESTED, COMMENTED
	CreatedAt time.Time
}

// Statistics calculates activity statistics
func (a *UserActivity) Statistics() map[string]interface{} {
	totalAdditions := 0
	totalDeletions := 0

	for _, c := range a.Commits {
		totalAdditions += c.Additions
		totalDeletions += c.Deletions
	}

	for _, pr := range a.PullRequests {
		totalAdditions += pr.Additions
		totalDeletions += pr.Deletions
	}

	mergedPRs := 0
	for _, pr := range a.PullRequests {
		if pr.MergedAt != nil {
			mergedPRs++
		}
	}

	closedIssues := 0
	for _, issue := range a.Issues {
		if issue.ClosedAt != nil {
			closedIssues++
		}
	}

	return map[string]interface{}{
		"total_commits":      len(a.Commits),
		"total_prs":          len(a.PullRequests),
		"merged_prs":         mergedPRs,
		"total_issues":       len(a.Issues),
		"closed_issues":      closedIssues,
		"total_reviews":      len(a.Reviews),
		"code_additions":     totalAdditions,
		"code_deletions":     totalDeletions,
		"net_code_changes":   totalAdditions - totalDeletions,
	}
}
