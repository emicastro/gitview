package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"emicastro.com/gitview/internal/github"
	"emicastro.com/gitview/internal/render"
	"emicastro.com/gitview/internal/stats"
)

const (
	version    = "0.1.0"
	defaultAPI = "https://api.github.com"
)

type config struct {
	user         string
	top          int
	json         bool
	includeForks bool
	fresh        bool
	version      bool
}

type deps struct {
	getenv  func(string) string
	stdout  io.Writer
	stderr  io.Writer
	baseURL string
	http    *http.Client
	now     func() time.Time
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	return runWith(ctx, args, deps{
		getenv:  getenv,
		stdout:  stdout,
		stderr:  stderr,
		baseURL: defaultAPI,
		now:     time.Now,
	})
}

func runWith(ctx context.Context, args []string, d deps) int {
	if d.now == nil {
		d.now = time.Now
	}
	if d.baseURL == "" {
		d.baseURL = defaultAPI
	}

	if err := ctx.Err(); err != nil {
		fmt.Fprintln(d.stderr, err)
		return 1
	}

	cfg, err := parseArgs(args, d.stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(d.stderr, err)
		return 2
	}

	if cfg.version {
		fmt.Fprintf(d.stdout, "gitview %s\n", version)
		return 0
	}

	token := d.getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Fprintln(d.stderr, "GITHUB_TOKEN required")
		return 2
	}

	snap, err := load(ctx, cfg, token, d)
	if err != nil {
		fmt.Fprintln(d.stderr, err)
		return 1
	}

	if cfg.json {
		if err := render.JSON(d.stdout, snap); err != nil {
			fmt.Fprintln(d.stderr, err)
			return 1
		}
		return 0
	}
	if err := render.Text(d.stdout, snap); err != nil {
		fmt.Fprintln(d.stderr, err)
		return 1
	}
	return 0
}

func load(ctx context.Context, cfg config, token string, d deps) (stats.Snapshot, error) {
	c := github.New(d.baseURL, token, d.http)
	repos, err := c.Fetch(ctx, cfg.user, cfg.includeForks, d.stderr)
	if err != nil {
		return stats.Snapshot{}, err
	}
	in := make([]stats.Input, 0, len(repos))
	for _, r := range repos {
		in = append(in, stats.Input{
			Repo: stats.Repo{
				Name:      r.Name,
				Stars:     r.Stars,
				Language:  r.PrimaryLanguage,
				UpdatedAt: r.UpdatedAt.UTC().Format("2006-01-02"),
				Fork:      r.Fork,
			},
			LangBytes: r.Languages,
		})
	}
	return stats.Build(cfg.user, d.now(), false, in, cfg.top), nil
}

func parseArgs(args []string, stderr io.Writer) (config, error) {
	// create a FlagSet here; do not use the global flag set
	fs := flag.NewFlagSet("gitview", flag.ContinueOnError)
	fs.SetOutput(stderr)

	top := fs.Int("top", 8, "how many languages to show")
	jsonOut := fs.Bool("json", false, "print JSON instead of TUI")
	includeForks := fs.Bool("forks", false, "include forked repos")
	fresh := fs.Bool("fresh", false, "ignore cache and refetch")
	showVersion := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}

	cfg := config{
		top:          *top,
		json:         *jsonOut,
		includeForks: *includeForks,
		fresh:        *fresh,
		version:      *showVersion,
	}

	if fs.NArg() >= 1 {
		cfg.user = fs.Arg(0)
	}

	if cfg.version {
		return cfg, nil
	}

	if cfg.user == "" {
		return config{}, errors.New("user required")
	}

	if cfg.top < 1 {
		return config{}, errors.New("top must be >= 1")
	}

	return cfg, nil
}
