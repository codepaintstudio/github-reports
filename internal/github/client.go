package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

// Client wraps GitHub API client
type Client struct {
	client *github.Client
	token  string
}

// NewClient creates a new GitHub client
func NewClient(token string) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		client: github.NewClient(tc),
		token:  token,
	}
}

// GetAuthenticatedUser returns the authenticated user's login name
func (c *Client) GetAuthenticatedUser(ctx context.Context) (string, error) {
	user, _, err := c.client.Users.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("failed to get authenticated user: %w", err)
	}

	if user.Login == nil {
		return "", fmt.Errorf("user login is nil")
	}

	return *user.Login, nil
}
