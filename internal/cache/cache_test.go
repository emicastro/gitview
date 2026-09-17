package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"emicastro.com/gitview/internal/stats"
)

func TestPutGetTTLAndNoToken(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Date(2026, 9, 17, 15, 0, 0, 0, time.UTC)
	s := New(dir, func() time.Time { return now })

	snap := stats.Snapshot{
		User:       "octocat",
		FetchedAt:  now,
		ReposCount: 1,
		Stars:      10,
		Languages:  []stats.Language{{Name: "Go", Bytes: 100, Percent: 100}},
		Repos:      []stats.Repo{{Name: "hello", Stars: 10, Language: "Go", UpdatedAt: "2026-01-02"}},
	}
	if err := s.Put("octocat", snap); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "octocat.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["token"]; ok {
		t.Fatal("cache file contains token field")
	}

	got, ok, err := s.Get("octocat")
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.User != "octocat" || got.Stars != 10 {
		t.Fatalf("got %+v", got)
	}

	s.Now = func() time.Time { return now.Add(2 * time.Hour) }
	_, ok, err = s.Get("octocat")
	if err != nil || ok {
		t.Fatalf("expired: ok=%v err=%v", ok, err)
	}

	if _, _, err := s.Get("octo/../cat"); err == nil {
		t.Fatal("expected invalid user")
	}
}

func TestCorruptCache(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "octocat.json")
	if err := os.WriteFile(p, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(dir, time.Now)
	_, _, err := s.Get("octocat")
	if err == nil {
		t.Fatal("expected corrupt cache error")
	}
}
