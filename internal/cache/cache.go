package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"emicastro.com/gitview/internal/render"
	"emicastro.com/gitview/internal/stats"
)

const TTL = time.Hour

type Store struct {
	Dir string
	Now func() time.Time
	TTL time.Duration
}

func New(dir string, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{Dir: dir, Now: now, TTL: TTL}
}

func (s *Store) path(user string) (string, error) {
	if err := validUser(user); err != nil {
		return "", err
	}
	return filepath.Join(s.Dir, user+".json"), nil
}

func validUser(user string) error {
	if user == "" || strings.Contains(user, "/") || strings.Contains(user, `\`) || strings.Contains(user, "..") {
		return fmt.Errorf("invalid cache user %q", user)
	}
	if filepath.Base(user) != user {
		return fmt.Errorf("invalid cache user %q", user)
	}
	return nil
}

func (s *Store) Get(user string) (stats.Snapshot, bool, error) {
	p, err := s.path(user)
	if err != nil {
		return stats.Snapshot{}, false, err
	}
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return stats.Snapshot{}, false, nil
		}
		return stats.Snapshot{}, false, err
	}
	defer f.Close()
	snap, err := render.Decode(f)
	if err != nil {
		return stats.Snapshot{}, false, fmt.Errorf("corrupt cache: %w", err)
	}
	age := s.Now().UTC().Sub(snap.FetchedAt)
	if age > s.TTL || age < 0 {
		return stats.Snapshot{}, false, nil
	}
	return snap, true, nil
}

func (s *Store) Put(user string, snap stats.Snapshot) error {
	p, err := s.path(user)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	tmp := p + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	encErr := render.JSON(f, snap)
	closeErr := f.Close()
	if encErr != nil {
		_ = os.Remove(tmp)
		return encErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, p)
}
