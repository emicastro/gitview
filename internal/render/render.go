package render

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"emicastro.com/gitview/internal/stats"
)

const barWidth = 20

type jsonLang struct {
	Name    string  `json:"name"`
	Bytes   int64   `json:"bytes"`
	Percent float64 `json:"percent"`
}

type jsonRepo struct {
	Name      string `json:"name"`
	Stars     int    `json:"stars"`
	Language  string `json:"language"`
	UpdatedAt string `json:"updated_at"`
	Fork      bool   `json:"fork"`
}

type jsonSnap struct {
	User       string     `json:"user"`
	FetchedAt  string     `json:"fetched_at"`
	Cached     bool       `json:"cached"`
	ReposCount int        `json:"repos_count"`
	Stars      int        `json:"stars"`
	Languages  []jsonLang `json:"languages"`
	Repos      []jsonRepo `json:"repos"`
}

func Decode(r io.Reader) (stats.Snapshot, error) {
	var in jsonSnap
	if err := json.NewDecoder(r).Decode(&in); err != nil {
		return stats.Snapshot{}, err
	}
	fetched, err := time.Parse(time.RFC3339, in.FetchedAt)
	if err != nil {
		return stats.Snapshot{}, err
	}
	s := stats.Snapshot{
		User:       in.User,
		FetchedAt:  fetched.UTC(),
		Cached:     in.Cached,
		ReposCount: in.ReposCount,
		Stars:      in.Stars,
		Languages:  make([]stats.Language, 0, len(in.Languages)),
		Repos:      make([]stats.Repo, 0, len(in.Repos)),
	}
	for _, l := range in.Languages {
		s.Languages = append(s.Languages, stats.Language{Name: l.Name, Bytes: l.Bytes, Percent: l.Percent})
	}
	for _, r := range in.Repos {
		s.Repos = append(s.Repos, stats.Repo{
			Name:      r.Name,
			Stars:     r.Stars,
			Language:  r.Language,
			UpdatedAt: r.UpdatedAt,
			Fork:      r.Fork,
		})
	}
	return s, nil
}

func JSON(w io.Writer, s stats.Snapshot) error {
	out := jsonSnap{
		User:       s.User,
		FetchedAt:  s.FetchedAt.UTC().Format(time.RFC3339),
		Cached:     s.Cached,
		ReposCount: s.ReposCount,
		Stars:      s.Stars,
		Languages:  make([]jsonLang, 0, len(s.Languages)),
		Repos:      make([]jsonRepo, 0, len(s.Repos)),
	}
	for _, l := range s.Languages {
		out.Languages = append(out.Languages, jsonLang{Name: l.Name, Bytes: l.Bytes, Percent: l.Percent})
	}
	for _, r := range s.Repos {
		out.Repos = append(out.Repos, jsonRepo{
			Name:      r.Name,
			Stars:     r.Stars,
			Language:  r.Language,
			UpdatedAt: r.UpdatedAt,
			Fork:      r.Fork,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func Text(w io.Writer, s stats.Snapshot) error {
	src := "live"
	if s.Cached {
		src = "cached"
	}
	if _, err := fmt.Fprintf(w, "@%s  repos=%d  stars=%d  %s\n", s.User, s.ReposCount, s.Stars, src); err != nil {
		return err
	}
	for _, l := range s.Languages {
		if _, err := fmt.Fprintf(w, "%s %s %5.1f%%\n", pad(l.Name, 12), bar(l.Percent), l.Percent); err != nil {
			return err
		}
	}
	for _, r := range s.Repos {
		if _, err := fmt.Fprintf(w, "%s %d %s %s\n", pad(r.Name, 16), r.Stars, r.Language, r.UpdatedAt); err != nil {
			return err
		}
	}
	return nil
}

func bar(pct float64) string {
	n := int(math.Round(pct / 100 * barWidth))
	if n < 0 {
		n = 0
	}
	if n > barWidth {
		n = barWidth
	}
	return strings.Repeat("█", n) + strings.Repeat(" ", barWidth-n)
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}
