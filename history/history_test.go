package history_test

import (
	"bytes"
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wewesss/ghstats/history"
	"github.com/wewesss/ghstats/snapshot"
)

func day(d int) time.Time { return time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC) }

func dc(d, count, uniques int) snapshot.DayCount {
	return snapshot.DayCount{Day: day(d), Count: count, Uniques: uniques}
}

func snap(repos ...snapshot.Repo) snapshot.Snapshot {
	return snapshot.Snapshot{SchemaVersion: snapshot.SchemaVersion, Repos: repos}
}

func repo(id, name string, traffic *snapshot.Traffic) snapshot.Repo {
	return snapshot.Repo{ID: id, NameWithOwner: name, Traffic: traffic}
}

func TestMergeSnapshotWinsOnOverlap(t *testing.T) {
	h := history.History{Repos: map[string]history.RepoHistory{
		"R_1": {NameWithOwner: "o/a", Views: []snapshot.DayCount{dc(1, 1, 1), dc(2, 2, 1), dc(3, 3, 1)}},
	}}
	s := snap(repo("R_1", "o/a", &snapshot.Traffic{
		Views:  []snapshot.DayCount{dc(3, 30, 9), dc(4, 4, 2)},
		Clones: []snapshot.DayCount{dc(4, 7, 3)},
	}))
	got := history.Merge(h, s).Repos["R_1"]
	wantViews := []snapshot.DayCount{dc(1, 1, 1), dc(2, 2, 1), dc(3, 30, 9), dc(4, 4, 2)}
	if !reflect.DeepEqual(got.Views, wantViews) {
		t.Fatalf("views = %+v, want %+v", got.Views, wantViews)
	}
	if !reflect.DeepEqual(got.Clones, []snapshot.DayCount{dc(4, 7, 3)}) {
		t.Fatalf("clones = %+v", got.Clones)
	}
}

func TestMergeKeepsOldDaysAndRepos(t *testing.T) {
	h := history.History{Repos: map[string]history.RepoHistory{
		"R_old": {NameWithOwner: "o/old", Views: []snapshot.DayCount{dc(1, 1, 1)}},
		"R_1":   {NameWithOwner: "o/a", Clones: []snapshot.DayCount{dc(1, 5, 1), dc(2, 6, 1)}},
	}}
	s := snap(
		repo("R_1", "o/a-renamed", &snapshot.Traffic{Clones: []snapshot.DayCount{dc(10, 1, 1)}}),
		repo("R_2", "o/b", nil),
	)
	got := history.Merge(h, s)
	if _, ok := got.Repos["R_old"]; !ok {
		t.Fatal("repo absent from the snapshot was dropped")
	}
	if _, ok := got.Repos["R_2"]; ok {
		t.Fatal("repo without traffic must not create an entry")
	}
	r1 := got.Repos["R_1"]
	if r1.NameWithOwner != "o/a-renamed" || len(r1.Clones) != 3 || r1.Clones[0].Count != 5 {
		t.Fatalf("R_1 = %+v", r1)
	}
	if got.SchemaVersion != history.SchemaVersion {
		t.Fatalf("schema = %d", got.SchemaVersion)
	}
}

func TestMergeSortsDays(t *testing.T) {
	s := snap(repo("R_1", "o/a", &snapshot.Traffic{Views: []snapshot.DayCount{dc(5, 1, 1), dc(2, 1, 1), dc(9, 1, 1)}}))
	got := history.Merge(history.History{}, s).Repos["R_1"].Views
	for i := 1; i < len(got); i++ {
		if !got[i-1].Day.Before(got[i].Day) {
			t.Fatalf("not sorted: %+v", got)
		}
	}
}

func TestMergeDoesNotMutateInputs(t *testing.T) {
	h := history.History{Repos: map[string]history.RepoHistory{
		"R_1": {NameWithOwner: "o/a", Views: []snapshot.DayCount{dc(1, 1, 1), dc(2, 2, 2)}},
	}}
	s := snap(repo("R_1", "o/a", &snapshot.Traffic{Views: []snapshot.DayCount{dc(2, 99, 9), dc(3, 3, 3)}}))

	var hb, sb bytes.Buffer
	if err := history.Write(&hb, h); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.Write(&sb, s); err != nil {
		t.Fatal(err)
	}
	hBefore, sBefore := hb.String(), sb.String()

	merged := history.Merge(h, s)
	merged.Repos["R_1"].Views[0].Count = 1000

	hb.Reset()
	sb.Reset()
	if err := history.Write(&hb, h); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.Write(&sb, s); err != nil {
		t.Fatal(err)
	}
	if hb.String() != hBefore || sb.String() != sBefore {
		t.Fatal("Merge mutated an input")
	}
}

func TestRoundTrip(t *testing.T) {
	h := history.Merge(history.History{}, snap(
		repo("R_1", "o/a", &snapshot.Traffic{Views: []snapshot.DayCount{dc(1, 1, 1)}, Clones: []snapshot.DayCount{dc(2, 2, 2)}}),
	))
	var buf bytes.Buffer
	if err := history.Write(&buf, h); err != nil {
		t.Fatal(err)
	}
	got, err := history.Read(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, h) {
		t.Fatalf("got %+v, want %+v", got, h)
	}

	path := filepath.Join(t.TempDir(), "history.json")
	if err := history.WriteFile(path, h); err != nil {
		t.Fatal(err)
	}
	got, err = history.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, h) {
		t.Fatalf("file: got %+v, want %+v", got, h)
	}
}

func TestReadRejectsWrongSchema(t *testing.T) {
	_, err := history.Read(strings.NewReader(`{"schema_version": 99, "repos": {}}`))
	if err == nil || !strings.Contains(err.Error(), "schema_version") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadFileMissing(t *testing.T) {
	if _, err := history.ReadFile(filepath.Join(t.TempDir(), "none.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v, want fs.ErrNotExist", err)
	}
}
