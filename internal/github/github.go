package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultBaseURL = "https://api.github.com"
	userAgent      = "gitview/0.1"
	langWorkers    = 4
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string, hc *http.Client) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    hc,
	}
}

type Repo struct {
	Name            string
	Owner           string
	Fork            bool
	Stars           int
	PrimaryLanguage string
	UpdatedAt       time.Time
	Languages       map[string]int64
}

type listRepo struct {
	Name            string `json:"name"`
	Fork            bool   `json:"fork"`
	StargazersCount int    `json:"stargazers_count"`
	Language        string `json:"language"`
	UpdatedAt       string `json:"updated_at"`
	Owner           struct {
		Login string `json:"login"`
	} `json:"owner"`
}

func (c *Client) ListRepos(ctx context.Context, user string) ([]Repo, error) {
	u := c.baseURL + "/users/" + url.PathEscape(user) + "/repos?per_page=100&type=owner"
	var out []Repo
	for u != "" {
		var page []listRepo
		next, err := c.getJSON(ctx, u, user, &page)
		if err != nil {
			return nil, err
		}
		for _, r := range page {
			updated, _ := time.Parse(time.RFC3339, r.UpdatedAt)
			out = append(out, Repo{
				Name:            r.Name,
				Owner:           r.Owner.Login,
				Fork:            r.Fork,
				Stars:           r.StargazersCount,
				PrimaryLanguage: r.Language,
				UpdatedAt:       updated,
			})
		}
		u = next
	}
	return out, nil
}

func (c *Client) Languages(ctx context.Context, owner, repo string) (map[string]int64, error) {
	u := c.baseURL + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/languages"
	var langs map[string]int64
	_, err := c.getJSON(ctx, u, owner, &langs)
	if err != nil {
		return nil, err
	}
	if langs == nil {
		langs = map[string]int64{}
	}
	return langs, nil
}

// Fetch lists repos, drops forks unless includeForks, fills Languages with at
// most 4 parallel requests. A languages failure or empty map skips that repo
// and writes a warning to warn.
func (c *Client) Fetch(ctx context.Context, user string, includeForks bool, warn io.Writer) ([]Repo, error) {
	listed, err := c.ListRepos(ctx, user)
	if err != nil {
		return nil, err
	}

	var selected []Repo
	for _, r := range listed {
		if r.Fork && !includeForks {
			continue
		}
		if r.Owner == "" {
			r.Owner = user
		}
		selected = append(selected, r)
	}

	sem := make(chan struct{}, langWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var out []Repo

	for _, r := range selected {
		r := r
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			langs, err := c.Languages(ctx, r.Owner, r.Name)
			if err != nil || len(langs) == 0 {
				if warn != nil {
					mu.Lock()
					if err != nil {
						fmt.Fprintf(warn, "skip %s/%s: %v\n", r.Owner, r.Name, err)
					} else {
						fmt.Fprintf(warn, "skip %s/%s: empty languages\n", r.Owner, r.Name)
					}
					mu.Unlock()
				}
				return
			}
			r.Languages = langs
			mu.Lock()
			out = append(out, r)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out, ctx.Err()
}

func (c *Client) getJSON(ctx context.Context, rawURL, user string, dest any) (next string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, res.Body)
		return "", apiError(res, user)
	}
	if err := json.NewDecoder(res.Body).Decode(dest); err != nil {
		return "", err
	}
	return nextLink(res.Header.Get("Link")), nil
}

func apiError(res *http.Response, user string) error {
	switch res.StatusCode {
	case http.StatusNotFound:
		return NotFoundError{User: user}
	case http.StatusUnauthorized:
		return AuthError{}
	case http.StatusForbidden:
		e := RateLimitError{}
		if raw := res.Header.Get("X-RateLimit-Reset"); raw != "" {
			if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
				e.Reset = time.Unix(n, 0).UTC()
				e.HasReset = true
			}
		}
		return e
	default:
		return fmt.Errorf("github http %d", res.StatusCode)
	}
}

func nextLink(h string) string {
	for _, part := range strings.Split(h, ",") {
		part = strings.TrimSpace(part)
		if !strings.Contains(part, `rel="next"`) && !strings.Contains(part, "rel=next") {
			continue
		}
		start := strings.Index(part, "<")
		end := strings.Index(part, ">")
		if start >= 0 && end > start {
			return part[start+1 : end]
		}
	}
	return ""
}

type NotFoundError struct{ User string }

func (e NotFoundError) Error() string { return "user not found: " + e.User }

type AuthError struct{}

func (AuthError) Error() string { return "github authentication failed" }

type RateLimitError struct {
	Reset    time.Time
	HasReset bool
}

func (e RateLimitError) Error() string {
	if e.HasReset {
		return "github rate limit exceeded, resets at " + e.Reset.UTC().Format(time.RFC3339)
	}
	return "github rate limit exceeded"
}
