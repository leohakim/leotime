package vcs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxResponseBytes = 256 << 10
const maxFactsPerKind = 12

type GiteaProvider struct {
	baseURL    string
	httpClient *http.Client
}

func NewGiteaProvider(baseURL string, client *http.Client) *GiteaProvider {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &GiteaProvider{baseURL: strings.TrimSuffix(strings.TrimSpace(baseURL), "/"), httpClient: client}
}

func (p *GiteaProvider) TestConnection(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return &ProviderError{Kind: ErrorUnauthorized}
	}
	var user struct {
		ID int `json:"id"`
	}
	if err := p.getJSON(ctx, "/api/v1/user", token, &user); err != nil {
		return err
	}
	if user.ID == 0 {
		return &ProviderError{Kind: ErrorInvalid}
	}
	return nil
}

func (p *GiteaProvider) CollectDayContext(ctx context.Context, request CollectionRequest) (Context, error) {
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(request.Date)); err != nil {
		return Context{}, &ProviderError{Kind: ErrorInvalid}
	}
	if strings.TrimSpace(request.Token) == "" {
		return Context{}, &ProviderError{Kind: ErrorUnauthorized}
	}

	result := Context{}
	for _, repository := range request.Repositories {
		owner := strings.Trim(strings.TrimSpace(repository.Owner), "/")
		name := strings.Trim(strings.TrimSpace(repository.Name), "/")
		if owner == "" || name == "" {
			return Context{}, &ProviderError{Kind: ErrorInvalid}
		}
		label := owner + "/" + name
		commits, err := p.collectCommits(ctx, request, owner, name, label)
		if err != nil {
			return Context{}, err
		}
		result.Commits = append(result.Commits, commits...)
		pulls, err := p.collectPulls(ctx, request, owner, name, label)
		if err != nil {
			return Context{}, err
		}
		result.PullRequests = append(result.PullRequests, pulls...)
		for _, pull := range pulls {
			reviews, err := p.collectReviews(ctx, request, owner, name, label, pull.Number)
			if err != nil {
				return Context{}, err
			}
			result.Reviews = append(result.Reviews, reviews...)
		}
		issues, err := p.collectIssues(ctx, request, owner, name, label)
		if err != nil {
			return Context{}, err
		}
		result.Issues = append(result.Issues, issues...)
	}
	result.Commits = uniqueCommits(result.Commits)
	result.PullRequests = capPullRequests(result.PullRequests)
	result.Reviews = capReviews(result.Reviews)
	result.Issues = capIssues(result.Issues)
	return result, nil
}

func (p *GiteaProvider) collectCommits(ctx context.Context, request CollectionRequest, owner, name, label string) ([]Commit, error) {
	var response []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name string `json:"name"`
				Date string `json:"date"`
			}
		}
	}
	if err := p.getJSON(ctx, "/api/v1/repos/"+owner+"/"+name+"/commits", request.Token, &response); err != nil {
		return nil, err
	}
	commits := make([]Commit, 0, len(response))
	for _, item := range response {
		if !matchesIdentity(request.OwnerIdentity, item.Commit.Author.Name) || !isOnDate(request.Date, item.Commit.Author.Date) {
			continue
		}
		subject := firstLine(item.Commit.Message)
		if strings.TrimSpace(item.SHA) == "" || subject == "" {
			continue
		}
		commits = append(commits, Commit{
			Repository: label, Hash: shortHash(item.SHA), Subject: truncate(subject, 180),
			Author: truncate(item.Commit.Author.Name, 80), OccurredAt: item.Commit.Author.Date,
		})
		if len(commits) == maxFactsPerKind {
			break
		}
	}
	return commits, nil
}

func (p *GiteaProvider) collectPulls(ctx context.Context, request CollectionRequest, owner, name, label string) ([]PullRequest, error) {
	var response []struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		State     string `json:"state"`
		Merged    bool   `json:"merged"`
		UpdatedAt string `json:"updated_at"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := p.getJSON(ctx, "/api/v1/repos/"+owner+"/"+name+"/pulls?state=all", request.Token, &response); err != nil {
		return nil, err
	}
	pulls := make([]PullRequest, 0, len(response))
	for _, item := range response {
		if item.Number <= 0 || strings.TrimSpace(item.Title) == "" || !isOnDate(request.Date, item.UpdatedAt) {
			continue
		}
		labels := make([]string, 0, len(item.Labels))
		for _, labelItem := range item.Labels {
			if value := strings.TrimSpace(labelItem.Name); value != "" {
				labels = append(labels, truncate(value, 40))
			}
		}
		pulls = append(pulls, PullRequest{
			Repository: label, Number: item.Number, Title: truncate(item.Title, 180),
			State: truncate(item.State, 24), Author: truncate(item.User.Login, 80),
			Labels: labels, Merged: item.Merged,
		})
		if len(pulls) == maxFactsPerKind {
			break
		}
	}
	return pulls, nil
}

func (p *GiteaProvider) collectReviews(ctx context.Context, request CollectionRequest, owner, name, label string, pullNumber int) ([]Review, error) {
	var response []struct {
		ID          int    `json:"id"`
		State       string `json:"state"`
		SubmittedAt string `json:"submitted_at"`
		User        struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if err := p.getJSON(ctx, fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d/reviews", owner, name, pullNumber), request.Token, &response); err != nil {
		return nil, err
	}
	reviews := make([]Review, 0, len(response))
	for _, item := range response {
		if item.ID == 0 || strings.TrimSpace(item.State) == "" || !isOnDate(request.Date, item.SubmittedAt) {
			continue
		}
		reviews = append(reviews, Review{
			Repository: label, PullRequestNumber: pullNumber,
			Reviewer: truncate(item.User.Login, 80), State: truncate(item.State, 24),
		})
		if len(reviews) == maxFactsPerKind {
			break
		}
	}
	return reviews, nil
}

func (p *GiteaProvider) collectIssues(ctx context.Context, request CollectionRequest, owner, name, label string) ([]Issue, error) {
	var response []struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		State     string `json:"state"`
		UpdatedAt string `json:"updated_at"`
	}
	if err := p.getJSON(ctx, "/api/v1/repos/"+owner+"/"+name+"/issues?state=all", request.Token, &response); err != nil {
		return nil, err
	}
	issues := make([]Issue, 0, len(response))
	for _, item := range response {
		if item.Number <= 0 || strings.TrimSpace(item.Title) == "" || !isOnDate(request.Date, item.UpdatedAt) {
			continue
		}
		issues = append(issues, Issue{
			Repository: label, Number: item.Number, Title: truncate(item.Title, 180), State: truncate(item.State, 24),
		})
		if len(issues) == maxFactsPerKind {
			break
		}
	}
	return issues, nil
}

func (p *GiteaProvider) getJSON(ctx context.Context, path, token string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+path, nil)
	if err != nil {
		return &ProviderError{Kind: ErrorInvalid}
	}
	request.Header.Set("Authorization", "token "+token)
	response, err := p.httpClient.Do(request)
	if err != nil {
		return &ProviderError{Kind: ErrorUnavailable}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes {
		return &ProviderError{Kind: ErrorInvalid}
	}
	switch response.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &ProviderError{Kind: ErrorUnauthorized}
	case http.StatusTooManyRequests:
		return &ProviderError{Kind: ErrorRateLimited}
	case http.StatusNotFound:
		return &ProviderError{Kind: ErrorInvalid}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return &ProviderError{Kind: ErrorUnavailable}
	}
	if err := json.Unmarshal(body, target); err != nil {
		return &ProviderError{Kind: ErrorInvalid}
	}
	return nil
}

func uniqueCommits(commits []Commit) []Commit {
	seen := map[string]struct{}{}
	result := make([]Commit, 0, len(commits))
	for _, commit := range commits {
		key := commit.Repository + ":" + commit.Hash
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, commit)
		if len(result) == maxFactsPerKind {
			break
		}
	}
	return result
}

func capPullRequests(items []PullRequest) []PullRequest {
	if len(items) > maxFactsPerKind {
		return items[:maxFactsPerKind]
	}
	return items
}

func capReviews(items []Review) []Review {
	if len(items) > maxFactsPerKind {
		return items[:maxFactsPerKind]
	}
	return items
}

func capIssues(items []Issue) []Issue {
	if len(items) > maxFactsPerKind {
		return items[:maxFactsPerKind]
	}
	return items
}

func matchesIdentity(expected, actual string) bool {
	expected = strings.ToLower(strings.TrimSpace(expected))
	actual = strings.ToLower(strings.TrimSpace(actual))
	return expected == "" || actual == "" || expected == actual || strings.Contains(expected, actual) || strings.Contains(actual, expected)
}

func isOnDate(day, timestamp string) bool {
	value, err := time.Parse(time.RFC3339, strings.TrimSpace(timestamp))
	return err == nil && value.UTC().Format("2006-01-02") == strings.TrimSpace(day)
}

func shortHash(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 7 {
		return value[:7]
	}
	return value
}

func firstLine(value string) string {
	if index := strings.IndexByte(value, '\n'); index >= 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value)
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max-1] + "…"
}
