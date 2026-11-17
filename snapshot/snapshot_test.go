package snapshot_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wewesss/ghstats/snapshot"
)

func sample() snapshot.Snapshot {
	day := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	return snapshot.Snapshot{
		SchemaVersion: snapshot.SchemaVersion,
		CollectedAt:   time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC),
		Viewer:        snapshot.Viewer{Login: "me", Followers: 3, Following: 4},
		Repos: []snapshot.Repo{
			{
				ID:            "R_1",
				NameWithOwner: "me/a",
				Owner:         "me",
				Stars:         5,
				Languages: []snapshot.LanguageSize{
					{Name: "Go", Bytes: 900},
					{Name: "C", Bytes: 100},
					{Name: "Make", Bytes: 100},
				},
				Releases: []snapshot.Release{{
					Tag:         "v1",
					PublishedAt: day,
					Assets:      []snapshot.Asset{{Name: "bin", DownloadCount: 7}},
				}},
				Traffic: &snapshot.Traffic{
					Views:  []snapshot.DayCount{{Day: day, Count: 2, Uniques: 1}},
					Clones: []snapshot.DayCount{{Day: day, Count: 3, Uniques: 2}},
				},
			},
			{ID: "R_2", NameWithOwner: "me/b", Owner: "me", Private: true},
		},
		Errors: []snapshot.RepoError{{Repo: "me/b", Stage: "traffic", Message: "forbidden"}},
	}
}

func TestRoundTrip(t *testing.T) {
	want := sample()
	var buf bytes.Buffer
	if err := snapshot.Write(&buf, want); err != nil {
		t.Fatal(err)
	}
	got, err := snapshot.Read(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestWriteOrdersDeterministically(t *testing.T) {
	s := sample()
	s.Repos[0], s.Repos[1] = s.Repos[1], s.Repos[0]
	s.Repos[1].Languages[0], s.Repos[1].Languages[2] = s.Repos[1].Languages[2], s.Repos[1].Languages[0]
	var buf bytes.Buffer
	if err := snapshot.Write(&buf, s); err != nil {
		t.Fatal(err)
	}
	got, err := snapshot.Read(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, sample()) {
		t.Fatalf("ordering not normalised: %+v", got)
	}
	if s.Repos[0].NameWithOwner != "me/b" {
		t.Fatal("Write mutated its input")
	}
}

func TestReadRejectsUnknownSchemaVersion(t *testing.T) {
	_, err := snapshot.Read(strings.NewReader(`{"schema_version": 99}`))
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")
	want := sample()
	if err := snapshot.WriteFile(path, want); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.WriteFile(path, want); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "snap.json" {
		t.Fatalf("unexpected directory content: %v", entries)
	}
	got, err := snapshot.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("file round trip mismatch")
	}
}
