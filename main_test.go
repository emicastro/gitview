package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"emicastro.com/gitview/internal/cache"
	"emicastro.com/gitview/internal/stats"
	"emicastro.com/gitview/internal/ui"
)

const dummyToken = "test-github-token"

func TestParseArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    config
		wantErr string
		help    bool
	}{
		{
			name: "happy",
			args: []string{"-top", "3", "-json", "-forks", "-fresh", "octocat"},
			want: config{user: "octocat", top: 3, json: true, includeForks: true, fresh: true},
		},
		{
			name: "defaults",
			args: []string{"octocat"},
			want: config{user: "octocat", top: 8},
		},
		{
			name:    "missing user",
			args:    []string{"-json"},
			wantErr: "user required",
		},
		{
			name:    "top zero",
			args:    []string{"-top", "0", "octocat"},
			wantErr: "top must be >= 1",
		},
		{
			name:    "unknown flag",
			args:    []string{"-nope", "octocat"},
			wantErr: "flag provided but not defined: -nope",
		},
		{
			name: "flags after user are not options",
			args: []string{"octocat", "-json"},
			want: config{user: "octocat", top: 8},
		},
		{
			name: "version without user",
			args: []string{"-version"},
			want: config{top: 8, version: true},
		},
		{
			name: "help",
			args: []string{"-h"},
			help: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseArgs(tt.args, io.Discard)
			if tt.help {
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("err = %v, want flag.ErrHelp", err)
				}
				return
			}
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseArgs: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		token      string
		wantCode   int
		wantStdout string
		wantStderr string
		forbid     string
	}{
		{
			name:       "help without token",
			args:       []string{"-h"},
			wantCode:   0,
			wantStdout: "",
		},
		{
			name:       "version without token",
			args:       []string{"-version"},
			wantCode:   0,
			wantStdout: "gitview 0.1.0\n",
		},
		{
			name:       "missing user",
			args:       []string{},
			wantCode:   2,
			wantStderr: "user required\n",
		},
		{
			name:       "missing token on fetch path",
			args:       []string{"octocat"},
			wantCode:   2,
			wantStderr: "GITHUB_TOKEN required\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			getenv := func(key string) string {
				if key == "GITHUB_TOKEN" {
					return tt.token
				}
				return ""
			}
			code := run(context.Background(), tt.args, getenv, &stdout, &stderr)
			if code != tt.wantCode {
				t.Fatalf("exit %d, want %d; stderr=%q stdout=%q", code, tt.wantCode, stderr.String(), stdout.String())
			}
			if tt.wantStdout != "" && stdout.String() != tt.wantStdout {
				t.Fatalf("stdout = %q, want %q", stdout.String(), tt.wantStdout)
			}
			if tt.wantStderr != "" && stderr.String() != tt.wantStderr {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.wantStderr)
			}
			forbid := tt.forbid
			if forbid == "" {
				forbid = dummyToken
			}
			out := stdout.String() + stderr.String()
			if forbid != "" && strings.Contains(out, forbid) {
				t.Fatalf("token leaked in output %q", out)
			}
		})
	}
}

func TestRunJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+dummyToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/user" {
			_ = json.NewEncoder(w).Encode(map[string]string{"login": "other"})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/languages") {
			_ = json.NewEncoder(w).Encode(map[string]int64{"Go": 12000, "C": 1000})
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"name":             "hello",
				"fork":             false,
				"stargazers_count": 10,
				"language":         "Go",
				"updated_at":       "2026-01-02T00:00:00Z",
				"owner":            map[string]string{"login": "octocat"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	var stdout, stderr bytes.Buffer
	code := runWith(context.Background(), []string{"-json", "octocat"}, deps{
		getenv: func(key string) string {
			if key == "GITHUB_TOKEN" {
				return dummyToken
			}
			return ""
		},
		stdout:  &stdout,
		stderr:  &stderr,
		baseURL: srv.URL,
		http:    srv.Client(),
		now:     func() time.Time { return time.Date(2026, 9, 17, 15, 0, 0, 0, time.UTC) },
		startUI: func(ui.Model) int {
			t.Error("TUI started on -json")
			return 1
		},
	})
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	out := stdout.String()
	if strings.Contains(out, dummyToken) || strings.Contains(stderr.String(), dummyToken) {
		t.Fatalf("token leaked")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["user"] != "octocat" || got["cached"] != false {
		t.Fatalf("got %#v", got)
	}
	if got["fetched_at"] != "2026-09-17T15:00:00Z" {
		t.Fatalf("fetched_at = %v", got["fetched_at"])
	}
}

func TestRunCacheHitWithoutToken(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Date(2026, 9, 17, 15, 0, 0, 0, time.UTC)
	st := cache.New(dir, func() time.Time { return now })
	if err := st.Put("octocat", stats.Snapshot{
		User:       "octocat",
		FetchedAt:  now,
		ReposCount: 1,
		Stars:      10,
		Languages:  []stats.Language{{Name: "Go", Bytes: 1, Percent: 100}},
	}); err != nil {
		t.Fatal(err)
	}

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	var stdout, stderr bytes.Buffer
	code := runWith(context.Background(), []string{"-json", "octocat"}, deps{
		getenv:   func(string) string { return "" },
		stdout:   &stdout,
		stderr:   &stderr,
		baseURL:  srv.URL,
		http:     srv.Client(),
		now:      func() time.Time { return now },
		cacheDir: dir,
	})
	if code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if calls.Load() != 0 {
		t.Fatalf("network called %d times", calls.Load())
	}
	if !strings.Contains(stdout.String(), `"cached": true`) {
		t.Fatalf("stdout=%s", stdout.String())
	}

	code = runWith(context.Background(), []string{"-fresh", "-json", "octocat"}, deps{
		getenv:   func(string) string { return "" },
		stdout:   &stdout,
		stderr:   &stderr,
		baseURL:  srv.URL,
		http:     srv.Client(),
		now:      func() time.Time { return now },
		cacheDir: dir,
	})
	if code != 2 {
		t.Fatalf("fresh without token exit %d", code)
	}
}
