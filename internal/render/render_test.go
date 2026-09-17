package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"emicastro.com/gitview/internal/stats"
)

func TestJSONShape(t *testing.T) {
	t.Parallel()

	s := stats.Snapshot{
		User:       "octocat",
		FetchedAt:  time.Date(2026, 9, 17, 15, 0, 0, 0, time.UTC),
		Cached:     false,
		ReposCount: 12,
		Stars:      340,
		Languages: []stats.Language{
			{Name: "Go", Bytes: 12000, Percent: 61.2},
			{Name: "Other", Bytes: 1000, Percent: 5.1},
		},
		Repos: []stats.Repo{
			{Name: "hello", Stars: 10, Language: "Go", UpdatedAt: "2026-01-02", Fork: false},
		},
	}
	var buf bytes.Buffer
	if err := JSON(&buf, s); err != nil {
		t.Fatal(err)
	}

	want := `{
  "user": "octocat",
  "fetched_at": "2026-09-17T15:00:00Z",
  "cached": false,
  "repos_count": 12,
  "stars": 340,
  "languages": [
    {
      "name": "Go",
      "bytes": 12000,
      "percent": 61.2
    },
    {
      "name": "Other",
      "bytes": 1000,
      "percent": 5.1
    }
  ],
  "repos": [
    {
      "name": "hello",
      "stars": 10,
      "language": "Go",
      "updated_at": "2026-01-02",
      "fork": false
    }
  ]
}
`
	if buf.String() != want {
		t.Fatalf("json mismatch\ngot:\n%s\nwant:\n%s", buf.String(), want)
	}
	if strings.Contains(buf.String(), "token") {
		t.Fatal("json mentioned token")
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
}

func TestTextBars(t *testing.T) {
	t.Parallel()

	s := stats.Snapshot{
		User:       "octocat",
		ReposCount: 1,
		Stars:      10,
		Languages:  []stats.Language{{Name: "Go", Bytes: 100, Percent: 100}},
		Repos:      []stats.Repo{{Name: "hello", Stars: 10, Language: "Go", UpdatedAt: "2026-01-02"}},
	}
	var buf bytes.Buffer
	if err := Text(&buf, s); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "@octocat  repos=1  stars=10  live") {
		t.Fatalf("header: %q", got)
	}
	if !strings.Contains(got, strings.Repeat("█", 20)) {
		t.Fatalf("bar: %q", got)
	}
	if !strings.Contains(got, "hello") {
		t.Fatalf("repo: %q", got)
	}
}
