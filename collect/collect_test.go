package collect_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-github/v93/github"

	"github.com/wewesss/ghstats/collect"
	"github.com/wewesss/ghstats/snapshot"
)

type fake struct {
	t             *testing.T
	repoPages     map[string]map[string]any
	releasePages  map[string]map[string]any
	trafficStatus map[string]int
	trafficDelay  time.Duration
	rateLimited   bool

	graphQLCalls atomic.Int32
	restCalls    atomic.Int32
	inflight     atomic.Int32
	maxInflight  atomic.Int32
}

func (f *fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if got := r.Header.Get("Authorization"); got != "Bearer tok" {
		f.t.Errorf("%s: Authorization = %q", r.URL.Path, got)
	}
	if r.URL.Path == "/graphql" {
		f.serveGraphQL(w, r)
		return
	}
	f.serveREST(w, r)
}

func (f *fake) serveGraphQL(w http.ResponseWriter, r *http.Request) {
	f.graphQLCalls.Add(1)
	var req struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.t.Errorf("decode request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cursor, _ := req.Variables["cursor"].(string)
	var data any
	switch {
	case strings.Contains(req.Query, "viewer"):
		page, ok := f.repoPages[cursor]
		if !ok {
			f.t.Errorf("unexpected repo cursor %q", cursor)
		}
		data = page
	case strings.Contains(req.Query, "node(id"):
		id, _ := req.Variables["id"].(string)
		page, ok := f.releasePages[id+"|"+cursor]
		if !ok {
			f.t.Errorf("unexpected release page %q %q", id, cursor)
		}
		data = map[string]any{"node": page}
	default:
		f.t.Errorf("unexpected query %q", req.Query)
	}
	json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func (f *fake) serveREST(w http.ResponseWriter, r *http.Request) {
	f.restCalls.Add(1)
	n := f.inflight.Add(1)
	defer f.inflight.Add(-1)
	for {
		m := f.maxInflight.Load()
		if n <= m || f.maxInflight.CompareAndSwap(m, n) {
			break
		}
	}
	time.Sleep(f.trafficDelay)

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 5 || parts[0] != "repos" || parts[3] != "traffic" {
		f.t.Errorf("unexpected REST path %q", r.URL.Path)
		http.NotFound(w, r)
		return
	}
	if got := r.URL.Query().Get("per"); got != "day" {
		f.t.Errorf("per = %q", got)
	}
	if f.rateLimited {
		w.Header().Set("X-RateLimit-Limit", "60")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10))
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"message":"API rate limit exceeded"}`)
		return
	}
	if status := f.trafficStatus[parts[1]+"/"+parts[2]]; status != 0 {
		w.WriteHeader(status)
		fmt.Fprint(w, `{"message":"Resource not accessible"}`)
		return
	}
	entry := `{"timestamp":"2026-11-01T00:00:00Z","count":5,"uniques":2}`
	fmt.Fprintf(w, `{"count":5,"uniques":2,%q:[%s]}`, parts[4], entry)
}

func repoNode(id, name string, releases map[string]any) map[string]any {
	owner, _, _ := strings.Cut(name, "/")
	if releases == nil {
		releases = releasePage(false, "")
	}
	return map[string]any{
		"id":             id,
		"nameWithOwner":  name,
		"owner":          map[string]any{"login": owner},
		"isPrivate":      false,
		"isFork":         false,
		"isArchived":     false,
		"stargazerCount": 3,
		"forkCount":      1,
		"languages": map[string]any{
			"totalCount": 1,
			"edges":      []any{map[string]any{"size": 1000, "node": map[string]any{"name": "Go"}}},
		},
		"releases": inner(releases),
	}
}

func releasePage(hasNext bool, endCursor string, tags ...string) map[string]any {
	nodes := []any{}
	for _, tag := range tags {
		nodes = append(nodes, map[string]any{
			"tagName":     tag,
			"publishedAt": "2026-10-01T12:00:00Z",
			"releaseAssets": map[string]any{
				"nodes": []any{map[string]any{"name": tag + ".zip", "downloadCount": 10}},
			},
		})
	}
	page := map[string]any{"nodes": nodes}
	pi := map[string]any{"hasNextPage": hasNext}
	if hasNext {
		pi["endCursor"] = endCursor
	}
	page["pageInfo"] = pi
	return map[string]any{"releases": page}
}

func repoPage(hasNext bool, endCursor string, nodes ...map[string]any) map[string]any {
	pi := map[string]any{"hasNextPage": hasNext}
	if hasNext {
		pi["endCursor"] = endCursor
	}
	return map[string]any{
		"viewer": map[string]any{
			"login":     "me",
			"followers": map[string]any{"totalCount": 7},
			"following": map[string]any{"totalCount": 2},
			"repositories": map[string]any{
				"pageInfo": pi,
				"nodes":    nodes,
			},
		},
		"rateLimit": map[string]any{"remaining": 4000, "resetAt": "2026-11-17T11:00:00Z"},
	}
}

func inner(releases map[string]any) map[string]any {
	return releases["releases"].(map[string]any)
}

func newFake(t *testing.T) *fake {
	return &fake{
		t: t,
		repoPages: map[string]map[string]any{
			"": repoPage(true, "c1",
				repoNode("R_3", "me/c", nil),
				repoNode("R_1", "me/a", nil),
			),
			"c1": repoPage(true, "c2",
				repoNode("R_2", "me/b", releasePage(true, "rc1", "v3", "v2")),
				repoNode("R_1", "me/a", nil),
			),
			"c2": repoPage(false, "",
				repoNode("R_4", "org/z", nil),
			),
		},
		releasePages: map[string]map[string]any{
			"R_2|rc1": releasePage(true, "rc2", "v1"),
			"R_2|rc2": releasePage(false, "", "v0"),
		},
		trafficStatus: map[string]int{},
	}
}

func run(t *testing.T, f *fake, opts collect.Options) (snapshot.Snapshot, error) {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	opts.Token = "tok"
	opts.GraphQLURL = srv.URL + "/graphql"
	opts.RESTURL = srv.URL + "/"
	opts.HTTPClient = srv.Client()
	return collect.Collect(t.Context(), opts)
}

func names(s snapshot.Snapshot) []string {
	out := make([]string, len(s.Repos))
	for i, r := range s.Repos {
		out[i] = r.NameWithOwner
	}
	return out
}

func TestCollectPagesDedupAndSort(t *testing.T) {
	f := newFake(t)
	now := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	snap, err := run(t, f, collect.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(names(snap), ","), "me/a,me/b,me/c,org/z"; got != want {
		t.Fatalf("repos = %s, want %s", got, want)
	}
	if snap.SchemaVersion != snapshot.SchemaVersion || !snap.CollectedAt.Equal(now) {
		t.Fatalf("header = %d %v", snap.SchemaVersion, snap.CollectedAt)
	}
	if snap.Viewer != (snapshot.Viewer{Login: "me", Followers: 7, Following: 2}) {
		t.Fatalf("viewer = %+v", snap.Viewer)
	}
	if len(snap.Errors) != 0 {
		t.Fatalf("errors = %+v", snap.Errors)
	}
	a := snap.Repos[0]
	if a.Stars != 3 || a.Forks != 1 || a.Owner != "me" || len(a.Languages) != 1 || a.Languages[0].Bytes != 1000 {
		t.Fatalf("repo a = %+v", a)
	}
}

func TestCollectFetchesRemainingReleasePages(t *testing.T) {
	f := newFake(t)
	snap, err := run(t, f, collect.Options{})
	if err != nil {
		t.Fatal(err)
	}
	var b snapshot.Repo
	for _, r := range snap.Repos {
		if r.NameWithOwner == "me/b" {
			b = r
		}
	}
	var tags []string
	for _, rel := range b.Releases {
		tags = append(tags, rel.Tag)
		if len(rel.Assets) != 1 || rel.Assets[0].DownloadCount != 10 {
			t.Fatalf("assets of %s = %+v", rel.Tag, rel.Assets)
		}
	}
	if got, want := strings.Join(tags, ","), "v3,v2,v1,v0"; got != want {
		t.Fatalf("tags = %s, want %s", got, want)
	}
}

func TestCollectTrafficForbiddenIsPartial(t *testing.T) {
	f := newFake(t)
	f.trafficStatus["me/b"] = http.StatusForbidden
	snap, err := run(t, f, collect.Options{Traffic: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Errors) != 1 || snap.Errors[0].Repo != "me/b" || snap.Errors[0].Stage != "traffic" {
		t.Fatalf("errors = %+v", snap.Errors)
	}
	for _, r := range snap.Repos {
		if r.NameWithOwner == "me/b" {
			if r.Traffic != nil {
				t.Fatal("me/b must have no traffic")
			}
			continue
		}
		if r.Traffic == nil || len(r.Traffic.Views) != 1 || len(r.Traffic.Clones) != 1 {
			t.Fatalf("%s traffic = %+v", r.NameWithOwner, r.Traffic)
		}
		v := r.Traffic.Views[0]
		if v.Count != 5 || v.Uniques != 2 || !v.Day.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("%s view = %+v", r.NameWithOwner, v)
		}
	}
}

func TestCollectWithoutTrafficMakesNoRESTCalls(t *testing.T) {
	f := newFake(t)
	snap, err := run(t, f, collect.Options{Traffic: false})
	if err != nil {
		t.Fatal(err)
	}
	if n := f.restCalls.Load(); n != 0 {
		t.Fatalf("REST calls = %d", n)
	}
	for _, r := range snap.Repos {
		if r.Traffic != nil {
			t.Fatalf("%s has traffic", r.NameWithOwner)
		}
	}
}

func TestCollectCancelledContext(t *testing.T) {
	f := newFake(t)
	srv := httptest.NewServer(f)
	defer srv.Close()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := collect.Collect(ctx, collect.Options{
		Token:      "tok",
		GraphQLURL: srv.URL + "/graphql",
		RESTURL:    srv.URL + "/",
		HTTPClient: srv.Client(),
		Traffic:    true,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestCollectCancelledDuringTraffic(t *testing.T) {
	f := newFake(t)
	f.trafficDelay = 200 * time.Millisecond
	srv := httptest.NewServer(f)
	defer srv.Close()
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(100*time.Millisecond, cancel)
	_, err := collect.Collect(ctx, collect.Options{
		Token:       "tok",
		GraphQLURL:  srv.URL + "/graphql",
		RESTURL:     srv.URL + "/",
		HTTPClient:  srv.Client(),
		Traffic:     true,
		Concurrency: 1,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestCollectRespectsConcurrency(t *testing.T) {
	f := newFake(t)
	f.trafficDelay = 20 * time.Millisecond
	var nodes []map[string]any
	for i := range 10 {
		nodes = append(nodes, repoNode(fmt.Sprintf("R_%02d", i), fmt.Sprintf("me/r%02d", i), nil))
	}
	f.repoPages = map[string]map[string]any{"": repoPage(false, "", nodes...)}

	const limit = 3
	snap, err := run(t, f, collect.Options{Traffic: true, Concurrency: limit})
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Repos) != 10 || len(snap.Errors) != 0 {
		t.Fatalf("repos=%d errors=%+v", len(snap.Repos), snap.Errors)
	}
	if got := f.restCalls.Load(); got != 20 {
		t.Fatalf("REST calls = %d, want 20", got)
	}
	if got := f.maxInflight.Load(); got < 1 || got > limit {
		t.Fatalf("max in-flight = %d, want within [1, %d]", got, limit)
	}
}

func TestCollectRateLimitStopsRemainingTraffic(t *testing.T) {
	f := newFake(t)
	f.rateLimited = true
	var nodes []map[string]any
	for i := range 20 {
		nodes = append(nodes, repoNode(fmt.Sprintf("R_%02d", i), fmt.Sprintf("me/r%02d", i), nil))
	}
	f.repoPages = map[string]map[string]any{"": repoPage(false, "", nodes...)}

	_, err := run(t, f, collect.Options{Traffic: true, Concurrency: 1})
	var rl *github.RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("err = %v, want *github.RateLimitError", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, must not be a bare cancellation", err)
	}
	if got := f.restCalls.Load(); got != 1 {
		t.Fatalf("REST calls = %d, want 1", got)
	}
}
