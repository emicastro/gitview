package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testToken = "test-github-token"

func TestListReposPaginatesAndAuth(t *testing.T) {
	t.Parallel()

	var pages int
	var ua, auth, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		auth = r.Header.Get("Authorization")
		path = r.URL.Path
		pages++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "2" {
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"name": "b", "fork": false, "stargazers_count": 1, "language": "Go", "updated_at": "2026-01-02T00:00:00Z", "owner": map[string]string{"login": "octocat"}},
			})
			return
		}
		w.Header().Set("Link", `<`+abs(r, "/users/octocat/repos?per_page=100&page=2")+`>; rel="next"`)
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"name": "a", "fork": true, "stargazers_count": 3, "language": "C", "updated_at": "2026-01-01T00:00:00Z", "owner": map[string]string{"login": "octocat"}},
		})
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, testToken, srv.Client())
	repos, err := c.ListRepos(context.Background(), "octocat")
	if err != nil {
		t.Fatal(err)
	}
	if ua != "gitview/0.1" {
		t.Fatalf("User-Agent = %q", ua)
	}
	if auth != "Bearer "+testToken {
		t.Fatalf("Authorization = %q", auth)
	}
	if path != "/users/octocat/repos" {
		t.Fatalf("path = %s", path)
	}
	if pages != 2 || len(repos) != 2 {
		t.Fatalf("pages=%d repos=%d", pages, len(repos))
	}
	if repos[0].Name != "a" || !repos[0].Fork || repos[0].Stars != 3 {
		t.Fatalf("repo0 = %+v", repos[0])
	}
}

func TestAPIErrors(t *testing.T) {
	t.Parallel()

	reset := time.Date(2026, 9, 17, 16, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		status int
		hdr    http.Header
		check  func(*testing.T, error)
	}{
		{
			name:   "404",
			status: 404,
			check: func(t *testing.T, err error) {
				var nf NotFoundError
				if !errors.As(err, &nf) || nf.Error() != "user not found: missing" {
					t.Fatalf("err = %v", err)
				}
			},
		},
		{
			name:   "401",
			status: 401,
			check: func(t *testing.T, err error) {
				var a AuthError
				if !errors.As(err, &a) || err.Error() != "github authentication failed" {
					t.Fatalf("err = %v", err)
				}
				if strings.Contains(err.Error(), testToken) {
					t.Fatalf("token leaked in %q", err)
				}
			},
		},
		{
			name:   "403 with reset",
			status: 403,
			hdr:    http.Header{"X-RateLimit-Reset": []string{strconv.FormatInt(reset.Unix(), 10)}},
			check: func(t *testing.T, err error) {
				var rl RateLimitError
				if !errors.As(err, &rl) {
					t.Fatalf("err = %v", err)
				}
				want := "github rate limit exceeded, resets at " + reset.Format(time.RFC3339)
				if err.Error() != want {
					t.Fatalf("err = %q, want %q", err, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, vs := range tt.hdr {
					w.Header()[k] = vs
				}
				w.WriteHeader(tt.status)
			}))
			t.Cleanup(srv.Close)
			c := New(srv.URL, testToken, srv.Client())
			_, err := c.ListRepos(context.Background(), "missing")
			if err == nil {
				t.Fatal("expected error")
			}
			tt.check(t, err)
		})
	}
}

func TestFetchSkipsLanguagesFailureAndEmpty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/repos") && !strings.Contains(r.URL.Path, "/languages"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				repoJSON("keep"),
				repoJSON("fail"),
				repoJSON("empty"),
			})
		case strings.Contains(r.URL.Path, "/keep/languages"):
			_ = json.NewEncoder(w).Encode(map[string]int64{"Go": 10})
		case strings.Contains(r.URL.Path, "/fail/languages"):
			w.WriteHeader(http.StatusInternalServerError)
		case strings.Contains(r.URL.Path, "/empty/languages"):
			_ = json.NewEncoder(w).Encode(map[string]int64{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	var warn bytes.Buffer
	c := New(srv.URL, testToken, srv.Client())
	repos, err := c.Fetch(context.Background(), "octocat", false, &warn)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Name != "keep" || repos[0].Languages["Go"] != 10 {
		t.Fatalf("repos = %+v", repos)
	}
	if !strings.Contains(warn.String(), "fail") || !strings.Contains(warn.String(), "empty") {
		t.Fatalf("warn = %q", warn.String())
	}
	if strings.Contains(warn.String(), testToken) {
		t.Fatalf("token leaked in warn %q", warn.String())
	}
}

func TestFetchDropsForksAndCapsConcurrency(t *testing.T) {
	t.Parallel()

	var current, max atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/repos") && !strings.Contains(r.URL.Path, "/languages") {
			var list []map[string]any
			for _, name := range []string{"r1", "r2", "r3", "r4", "r5", "forked"} {
				item := repoJSON(name)
				if name == "forked" {
					item["fork"] = true
				}
				list = append(list, item)
			}
			_ = json.NewEncoder(w).Encode(list)
			return
		}
		n := current.Add(1)
		for {
			old := max.Load()
			if n <= old || max.CompareAndSwap(old, n) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		current.Add(-1)
		_ = json.NewEncoder(w).Encode(map[string]int64{"Go": 1})
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, testToken, srv.Client())
	repos, err := c.Fetch(context.Background(), "octocat", false, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 5 {
		t.Fatalf("len=%d (fork should be dropped)", len(repos))
	}
	if got := max.Load(); got > 4 {
		t.Fatalf("max concurrency %d > 4", got)
	}
}

func repoJSON(name string) map[string]any {
	return map[string]any{
		"name":             name,
		"fork":             false,
		"stargazers_count": 1,
		"language":         "Go",
		"updated_at":       "2026-01-02T00:00:00Z",
		"owner":            map[string]string{"login": "octocat"},
	}
}

func abs(r *http.Request, path string) string {
	return "http://" + r.Host + path
}
