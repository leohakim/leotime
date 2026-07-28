package vcs

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type Repository struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
}

type CollectionRequest struct {
	Date          string       `json:"date"`
	OwnerIdentity string       `json:"ownerIdentity"`
	Token         string       `json:"-"`
	Repositories  []Repository `json:"repositories"`
}

type Commit struct {
	Repository string `json:"repository"`
	Hash       string `json:"hash"`
	Subject    string `json:"subject"`
	Author     string `json:"author"`
	OccurredAt string `json:"occurredAt"`
}

type PullRequest struct {
	Repository string   `json:"repository"`
	Number     int      `json:"number"`
	Title      string   `json:"title"`
	State      string   `json:"state"`
	Author     string   `json:"author"`
	Labels     []string `json:"labels"`
	Merged     bool     `json:"merged"`
}

type Review struct {
	Repository        string `json:"repository"`
	PullRequestNumber int    `json:"pullRequestNumber"`
	Reviewer          string `json:"reviewer"`
	State             string `json:"state"`
}

type Issue struct {
	Repository string `json:"repository"`
	Number     int    `json:"number"`
	Title      string `json:"title"`
	State      string `json:"state"`
}

type Context struct {
	Commits      []Commit      `json:"commits"`
	PullRequests []PullRequest `json:"pullRequests"`
	Reviews      []Review      `json:"reviews"`
	Issues       []Issue       `json:"issues"`
}

type DayContextCollector interface {
	CollectDayContext(context.Context, CollectionRequest) (Context, error)
}

type ConnectionTester interface {
	TestConnection(context.Context, string) error
}

type Provider interface {
	DayContextCollector
	ConnectionTester
}

func (c Context) PromptFacts() string {
	parts := make([]string, 0, len(c.Commits)+len(c.PullRequests)+len(c.Reviews)+len(c.Issues))
	for _, commit := range c.Commits {
		parts = append(parts, fmt.Sprintf("commit %s: %s", commit.Hash, commit.Subject))
	}
	for _, pull := range c.PullRequests {
		parts = append(parts, fmt.Sprintf("PR #%d: %s (%s)", pull.Number, pull.Title, pull.State))
	}
	for _, review := range c.Reviews {
		parts = append(parts, fmt.Sprintf("review PR #%d: %s (%s)", review.PullRequestNumber, review.Reviewer, review.State))
	}
	for _, issue := range c.Issues {
		parts = append(parts, fmt.Sprintf("issue #%d: %s (%s)", issue.Number, issue.Title, issue.State))
	}
	return strings.Join(parts, "\n")
}

type ErrorKind string

const (
	ErrorUnauthorized ErrorKind = "unauthorized"
	ErrorRateLimited  ErrorKind = "rate_limited"
	ErrorUnavailable  ErrorKind = "unavailable"
	ErrorInvalid      ErrorKind = "invalid_response"
)

type ProviderError struct {
	Kind ErrorKind
}

func (e *ProviderError) Error() string {
	return "vcs provider " + string(e.Kind)
}

func IsUnauthorized(err error) bool {
	var providerErr *ProviderError
	return errors.As(err, &providerErr) && providerErr.Kind == ErrorUnauthorized
}

func IsRateLimited(err error) bool {
	var providerErr *ProviderError
	return errors.As(err, &providerErr) && providerErr.Kind == ErrorRateLimited
}

func IsUnavailable(err error) bool {
	var providerErr *ProviderError
	return errors.As(err, &providerErr) && providerErr.Kind == ErrorUnavailable
}
