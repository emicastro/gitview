package stats

import (
	"math"
	"sort"
	"time"
)

// Repo is a repository included in a snapshot (no language byte map).
type Repo struct {
	Name      string
	Stars     int
	Language  string
	UpdatedAt string
	Fork      bool
}

// Language is aggregated byte counts for one language (or the Other bucket).
type Language struct {
	Name    string
	Bytes   int64
	Percent float64
}

// Snapshot is the load result shared by JSON, text, cache, and TUI.
type Snapshot struct {
	User       string
	FetchedAt  time.Time
	Cached     bool
	ReposCount int
	Stars      int
	Languages  []Language
	Repos      []Repo
	Skipped    int `json:"-"`
}

// Input is one included repo plus its language byte map.
type Input struct {
	Repo
	LangBytes map[string]int64
}

func Build(user string, fetchedAt time.Time, cached bool, in []Input, top int) Snapshot {
	maps := make([]map[string]int64, 0, len(in))
	repos := make([]Repo, 0, len(in))
	for _, r := range in {
		maps = append(maps, r.LangBytes)
		repos = append(repos, r.Repo)
	}
	repos = SortRepos(repos)
	return Snapshot{
		User:       user,
		FetchedAt:  fetchedAt.UTC(),
		Cached:     cached,
		ReposCount: len(repos),
		Stars:      TotalStars(repos),
		Languages:  TopN(SumLanguages(maps), top),
		Repos:      repos,
	}
}

func SumLanguages(maps []map[string]int64) map[string]int64 {
	out := make(map[string]int64)
	for _, m := range maps {
		for name, n := range m {
			out[name] += n
		}
	}
	return out
}

func TotalStars(repos []Repo) int {
	var n int
	for _, r := range repos {
		n += r.Stars
	}
	return n
}

func SortRepos(repos []Repo) []Repo {
	out := append([]Repo(nil), repos...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Stars != out[j].Stars {
			return out[i].Stars > out[j].Stars
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func TopN(bytes map[string]int64, n int) []Language {
	if n < 1 {
		n = 1
	}

	type kv struct {
		name  string
		bytes int64
	}
	items := make([]kv, 0, len(bytes))
	var total int64
	for name, b := range bytes {
		total += b
		items = append(items, kv{name: name, bytes: b})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].bytes != items[j].bytes {
			return items[i].bytes > items[j].bytes
		}
		return items[i].name < items[j].name
	})

	pct := func(b int64) float64 {
		if total == 0 {
			return 0
		}
		return math.Round(float64(b)/float64(total)*1000) / 10
	}

	if len(items) <= n {
		out := make([]Language, len(items))
		for i, it := range items {
			out[i] = Language{Name: it.name, Bytes: it.bytes, Percent: pct(it.bytes)}
		}
		return out
	}

	out := make([]Language, 0, n+1)
	var rest int64
	for i, it := range items {
		if i < n {
			out = append(out, Language{Name: it.name, Bytes: it.bytes, Percent: pct(it.bytes)})
			continue
		}
		rest += it.bytes
	}
	if rest > 0 {
		out = append(out, Language{Name: "Other", Bytes: rest, Percent: pct(rest)})
	}
	return out
}
