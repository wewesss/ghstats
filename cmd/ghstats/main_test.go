package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wewesss/ghstats/history"
	"github.com/wewesss/ghstats/report"
	"github.com/wewesss/ghstats/snapshot"
)

var fixedNow = time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)

type apiFake struct {
	rest atomic.Int32
}

func (f *apiFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/graphql" {
		node := func(id, name string, private bool) map[string]any {
			owner, _, _ := strings.Cut(name, "/")
			return map[string]any{
				"id": id, "nameWithOwner": name, "owner": map[string]any{"login": owner},
				"isPrivate": private, "isFork": false, "isArchived": false,
				"stargazerCount": 3, "forkCount": 1,
				"languages": map[string]any{"totalCount": 1, "edges": []any{map[string]any{"size": 1000, "node": map[string]any{"name": "Go"}}}},
				"releases":  map[string]any{"nodes": []any{}, "pageInfo": map[string]any{"hasNextPage": false}},
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
			"viewer": map[string]any{
				"login":     "me",
				"followers": map[string]any{"totalCount": 7},
				"following": map[string]any{"totalCount": 2},
				"repositories": map[string]any{
					"pageInfo": map[string]any{"hasNextPage": false},
					"nodes":    []any{node("R_1", "me/open", false), node("R_2", "me/hidden-project", true)},
				},
			},
			"rateLimit": map[string]any{"remaining": 4000, "resetAt": "2026-11-17T11:00:00Z"},
		}})
		return
	}
	f.rest.Add(1)
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 5 || parts[3] != "traffic" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprintf(w, `{"count":5,"uniques":2,%q:[{"timestamp":"2026-11-10T00:00:00Z","count":5,"uniques":2}]}`, parts[4])
}

func setup(t *testing.T) *apiFake {
	t.Helper()
	f := &apiFake{}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	t.Setenv("GHSTATS_TOKEN", "tok")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GHSTATS_GRAPHQL_URL", srv.URL+"/graphql")
	t.Setenv("GHSTATS_REST_URL", srv.URL+"/")
	old := clock
	clock = func() time.Time { return fixedNow }
	t.Cleanup(func() { clock = old })
	return f
}

func exec(args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestUsageErrorsExitTwo(t *testing.T) {
	tests := [][]string{
		nil,
		{"bogus"},
		{"collect", "--bogus"},
		{"collect", "extra"},
		{"collect", "--concurrency", "-1"},
		{"compute"},
		{"compute", "--snapshot", "s.json", "--history", "h.json"},
		{"compute", "--out", "r.json"},
		{"render"},
		{"render", "--report", "r.json"},
		{"render", "--out-dir", "out"},
		{"render", "--report", "r.json", "--out-dir", "out", "stray"},
	}
	for _, args := range tests {
		code, _, stderr := exec(args...)
		if code != 2 {
			t.Errorf("run(%v) = %d, want 2 (stderr %q)", args, code, stderr)
		}
		if stderr == "" {
			t.Errorf("run(%v) printed nothing on stderr", args)
		}
	}
}

func TestVersionAndHelp(t *testing.T) {
	code, stdout, _ := exec("version")
	if code != 0 || strings.TrimSpace(stdout) != "dev" {
		t.Fatalf("version = %d %q", code, stdout)
	}
	if code, _, _ := exec("compute", "-h"); code != 0 {
		t.Fatalf("compute -h = %d", code)
	}
}

func TestCollectWithoutTokenExitsOne(t *testing.T) {
	t.Setenv("GHSTATS_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	code, _, stderr := exec("collect", "--out", t.TempDir(), "--history", filepath.Join(t.TempDir(), "h.json"))
	if code != 1 || !strings.Contains(stderr, "GHSTATS_TOKEN") {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
}

func TestCollectWritesSnapshotAndHistory(t *testing.T) {
	f := setup(t)
	dir := t.TempDir()
	out, histPath := filepath.Join(dir, "snapshots"), filepath.Join(dir, "nested", "history.json")

	code, stdout, stderr := exec("collect", "--out", out, "--history", histPath, "--concurrency", "2")
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	want := filepath.Join(out, "snapshot-2026-11-17T090000Z.json")
	if strings.TrimSpace(stdout) != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	s, err := snapshot.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if s.Viewer.Login != "me" || len(s.Repos) != 2 || s.Repos[0].Traffic == nil {
		t.Fatalf("snapshot = %+v", s)
	}
	h, err := history.ReadFile(histPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Repos) != 2 || len(h.Repos["R_1"].Views) != 1 || h.Repos["R_1"].Views[0].Count != 5 {
		t.Fatalf("history = %+v", h)
	}
	if f.rest.Load() == 0 {
		t.Fatal("no traffic requests were made")
	}

	if code, _, stderr := exec("collect", "--out", out, "--history", histPath); code != 0 {
		t.Fatalf("second collect = %d, %q", code, stderr)
	}
	h2, err := history.ReadFile(histPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(h2.Repos["R_1"].Views) != 1 {
		t.Fatalf("history merge duplicated days: %+v", h2.Repos["R_1"].Views)
	}
}

func TestCollectFallsBackToGitHubTokenAndSkipsTraffic(t *testing.T) {
	f := setup(t)
	t.Setenv("GHSTATS_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "tok")
	dir := t.TempDir()
	code, _, stderr := exec("collect", "--no-traffic", "--out", dir, "--history", filepath.Join(dir, "h.json"))
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	if f.rest.Load() != 0 {
		t.Fatalf("%d traffic requests with --no-traffic", f.rest.Load())
	}
}

func TestComputeThenRenderEndToEnd(t *testing.T) {
	setup(t)
	dir := t.TempDir()
	histPath := filepath.Join(dir, "history.json")
	code, stdout, stderr := exec("collect", "--out", dir, "--history", histPath)
	if code != 0 {
		t.Fatalf("collect = %d, %q", code, stderr)
	}
	snapPath := strings.TrimSpace(stdout)

	cfgPath := filepath.Join(dir, "ghstats.yaml")
	cfg := `version: 1
render:
  blocks:
    - stats: [totals, distribution]
    - languages: [bars, iso, cells, treemap, shift, terminal]
    - history: [views, clones]
  theme:
    accent: cycle
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(dir, "out", "report.json")
	code, _, stderr = exec("compute", "--snapshot", snapPath, "--history", histPath, "--config", cfgPath, "--out", reportPath)
	if code != 0 {
		t.Fatalf("compute = %d, %q", code, stderr)
	}
	var r report.Report
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	if r.Login != "me" || r.Totals.PrivateRepos != 1 || len(r.Trends) != 2 {
		t.Fatalf("report = %+v", r)
	}

	outDir := filepath.Join(dir, "cards")
	code, _, stderr = exec("render", "--report", reportPath, "--config", cfgPath, "--out-dir", outDir)
	if code != 0 {
		t.Fatalf("render = %d, %q", code, stderr)
	}
	for _, name := range []string{"stats-dark.svg", "stats-light.svg"} {
		data, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "hidden-project") {
			t.Fatalf("%s leaks a private repository name", name)
		}
		dec := xml.NewDecoder(bytes.NewReader(data))
		for {
			if _, err := dec.Token(); err == io.EOF {
				break
			} else if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
		}
	}
	stats, err := os.ReadFile(filepath.Join(outDir, "stats.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stats), "hidden-project") || !strings.HasSuffix(string(stats), "}\n") {
		t.Fatal("unexpected stats.json")
	}
}

func TestComputeWarnsAndAcceptsMissingHistory(t *testing.T) {
	setup(t)
	dir := t.TempDir()
	code, stdout, stderr := exec("collect", "--no-traffic", "--out", dir, "--history", filepath.Join(dir, "h.json"))
	if code != 0 {
		t.Fatalf("collect = %d, %q", code, stderr)
	}
	reportPath := filepath.Join(dir, "report.json")
	code, _, stderr = exec("compute", "--snapshot", strings.TrimSpace(stdout), "--history", filepath.Join(dir, "none.json"), "--out", reportPath)
	if code != 0 {
		t.Fatalf("compute = %d, %q", code, stderr)
	}
	if !strings.Contains(stderr, "warning:") {
		t.Fatalf("stderr = %q, want a warning", stderr)
	}
	if code, _, _ := exec("render", "--report", reportPath, "--out-dir", filepath.Join(dir, "cards")); code != 0 {
		t.Fatalf("render = %d", code)
	}
}

func TestRuntimeErrorsExitOne(t *testing.T) {
	dir := t.TempDir()
	tests := [][]string{
		{"compute", "--snapshot", filepath.Join(dir, "missing.json"), "--history", filepath.Join(dir, "h.json"), "--out", filepath.Join(dir, "r.json")},
		{"render", "--report", filepath.Join(dir, "missing.json"), "--out-dir", dir},
	}
	for _, args := range tests {
		if code, _, stderr := exec(args...); code != 1 || stderr == "" {
			t.Errorf("run(%v) = %d, stderr %q", args, code, stderr)
		}
	}
}
