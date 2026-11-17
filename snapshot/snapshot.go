// Package snapshot defines the versioned raw data model and its JSON storage.
package snapshot

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"
)

const SchemaVersion = 1

type Snapshot struct {
	SchemaVersion int         `json:"schema_version"`
	CollectedAt   time.Time   `json:"collected_at"`
	Viewer        Viewer      `json:"viewer"`
	Repos         []Repo      `json:"repos"`
	Errors        []RepoError `json:"errors"`
}

type Viewer struct {
	Login     string `json:"login"`
	Followers int    `json:"followers"`
	Following int    `json:"following"`
}

type Repo struct {
	ID            string         `json:"id"`
	NameWithOwner string         `json:"name_with_owner"`
	Owner         string         `json:"owner"`
	Private       bool           `json:"private"`
	Fork          bool           `json:"fork"`
	Archived      bool           `json:"archived"`
	Stars         int            `json:"stars"`
	Forks         int            `json:"forks"`
	Languages     []LanguageSize `json:"languages"`
	Releases      []Release      `json:"releases"`
	Traffic       *Traffic       `json:"traffic,omitempty"`
}

type LanguageSize struct {
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
}

type Release struct {
	Tag         string    `json:"tag"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

type Asset struct {
	Name          string `json:"name"`
	DownloadCount int64  `json:"download_count"`
}

type Traffic struct {
	Views  []DayCount `json:"views"`
	Clones []DayCount `json:"clones"`
}

type DayCount struct {
	Day     time.Time `json:"day"`
	Count   int       `json:"count"`
	Uniques int       `json:"uniques"`
}

type RepoError struct {
	Repo    string `json:"repo"`
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

func Write(w io.Writer, s Snapshot) error {
	s.Repos = slices.Clone(s.Repos)
	slices.SortStableFunc(s.Repos, func(a, b Repo) int {
		return cmp.Compare(a.NameWithOwner, b.NameWithOwner)
	})
	for i := range s.Repos {
		langs := slices.Clone(s.Repos[i].Languages)
		slices.SortStableFunc(langs, func(a, b LanguageSize) int {
			return cmp.Or(cmp.Compare(b.Bytes, a.Bytes), cmp.Compare(a.Name, b.Name))
		})
		s.Repos[i].Languages = langs
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		return fmt.Errorf("snapshot: encode: %w", err)
	}
	return nil
}

func Read(r io.Reader) (Snapshot, error) {
	var s Snapshot
	if err := json.NewDecoder(r).Decode(&s); err != nil {
		return Snapshot{}, fmt.Errorf("snapshot: decode: %w", err)
	}
	if s.SchemaVersion != SchemaVersion {
		return Snapshot{}, fmt.Errorf("snapshot: unsupported schema_version %d (want %d)", s.SchemaVersion, SchemaVersion)
	}
	return s, nil
}

func WriteFile(path string, s Snapshot) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), ".snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("snapshot: create temp file: %w", err)
	}
	tmp := f.Name()
	defer func() {
		if err != nil {
			os.Remove(tmp)
		}
	}()
	if err = Write(f, s); err != nil {
		return errors.Join(err, f.Close())
	}
	if err = f.Sync(); err != nil {
		return errors.Join(fmt.Errorf("snapshot: sync: %w", err), f.Close())
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("snapshot: close: %w", err)
	}
	if err = os.Rename(tmp, path); err != nil {
		return fmt.Errorf("snapshot: rename: %w", err)
	}
	return nil
}

func ReadFile(path string) (Snapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("snapshot: open: %w", err)
	}
	defer f.Close()
	return Read(f)
}
