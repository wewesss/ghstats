// Package history keeps traffic day counts beyond the 14-day window GitHub exposes.
package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/wewesss/ghstats/snapshot"
)

const SchemaVersion = 1

type History struct {
	SchemaVersion int                    `json:"schema_version"`
	Repos         map[string]RepoHistory `json:"repos"`
}

type RepoHistory struct {
	NameWithOwner string              `json:"name_with_owner"`
	Private       bool                `json:"private"`
	Views         []snapshot.DayCount `json:"views"`
	Clones        []snapshot.DayCount `json:"clones"`
}

func Merge(h History, s snapshot.Snapshot) History {
	out := History{SchemaVersion: SchemaVersion, Repos: make(map[string]RepoHistory, len(h.Repos)+len(s.Repos))}
	for id, rh := range h.Repos {
		out.Repos[id] = RepoHistory{
			NameWithOwner: rh.NameWithOwner,
			Private:       rh.Private,
			Views:         slices.Clone(rh.Views),
			Clones:        slices.Clone(rh.Clones),
		}
	}
	for _, r := range s.Repos {
		if r.Traffic == nil {
			continue
		}
		rh := out.Repos[r.ID]
		rh.NameWithOwner = r.NameWithOwner
		rh.Private = r.Private
		rh.Views = mergeDays(rh.Views, r.Traffic.Views)
		rh.Clones = mergeDays(rh.Clones, r.Traffic.Clones)
		out.Repos[r.ID] = rh
	}
	return out
}

func mergeDays(old, fresh []snapshot.DayCount) []snapshot.DayCount {
	byDay := make(map[time.Time]snapshot.DayCount, len(old)+len(fresh))
	for _, group := range [][]snapshot.DayCount{old, fresh} {
		for _, d := range group {
			d.Day = d.Day.UTC().Truncate(24 * time.Hour)
			byDay[d.Day] = d
		}
	}
	out := slices.Collect(maps.Values(byDay))
	sortDays(out)
	return out
}

func sortDays(days []snapshot.DayCount) {
	slices.SortFunc(days, func(a, b snapshot.DayCount) int { return a.Day.Compare(b.Day) })
}

func Write(w io.Writer, h History) error {
	h.SchemaVersion = SchemaVersion
	repos := make(map[string]RepoHistory, len(h.Repos))
	for id, rh := range h.Repos {
		rh.Views = slices.Clone(rh.Views)
		rh.Clones = slices.Clone(rh.Clones)
		sortDays(rh.Views)
		sortDays(rh.Clones)
		if rh.Views == nil {
			rh.Views = []snapshot.DayCount{}
		}
		if rh.Clones == nil {
			rh.Clones = []snapshot.DayCount{}
		}
		repos[id] = rh
	}
	h.Repos = repos
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(h); err != nil {
		return fmt.Errorf("history: encode: %w", err)
	}
	return nil
}

func Read(r io.Reader) (History, error) {
	var h History
	if err := json.NewDecoder(r).Decode(&h); err != nil {
		return History{}, fmt.Errorf("history: decode: %w", err)
	}
	if h.SchemaVersion != SchemaVersion {
		return History{}, fmt.Errorf("history: unsupported schema_version %d (want %d)", h.SchemaVersion, SchemaVersion)
	}
	if h.Repos == nil {
		h.Repos = map[string]RepoHistory{}
	}
	return h, nil
}

func WriteFile(path string, h History) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), ".history-*.tmp")
	if err != nil {
		return fmt.Errorf("history: create temp file: %w", err)
	}
	tmp := f.Name()
	defer func() {
		if err != nil {
			os.Remove(tmp)
		}
	}()
	if err = Write(f, h); err != nil {
		return errors.Join(err, f.Close())
	}
	if err = f.Sync(); err != nil {
		return errors.Join(fmt.Errorf("history: sync: %w", err), f.Close())
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("history: close: %w", err)
	}
	if err = os.Rename(tmp, path); err != nil {
		return fmt.Errorf("history: rename: %w", err)
	}
	return nil
}

func ReadFile(path string) (History, error) {
	f, err := os.Open(path)
	if err != nil {
		return History{}, fmt.Errorf("history: open: %w", err)
	}
	defer f.Close()
	return Read(f)
}
