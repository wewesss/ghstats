package compute_test

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/wewesss/ghstats/compute"
	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/history"
	"github.com/wewesss/ghstats/report"
	"github.com/wewesss/ghstats/snapshot"
)

const tol = 1e-9

var now = time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)

func near(a, b float64) bool { return math.Abs(a-b) <= tol }

func langs(pairs ...any) []snapshot.LanguageSize {
	var out []snapshot.LanguageSize
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, snapshot.LanguageSize{Name: pairs[i].(string), Bytes: int64(pairs[i+1].(int))})
	}
	return out
}

func repo(name string, l []snapshot.LanguageSize) snapshot.Repo {
	owner, _, _ := strings.Cut(name, "/")
	return snapshot.Repo{ID: "ID_" + name, NameWithOwner: name, Owner: owner, Languages: l}
}

func snap(repos ...snapshot.Repo) snapshot.Snapshot {
	return snapshot.Snapshot{
		SchemaVersion: snapshot.SchemaVersion,
		CollectedAt:   now,
		Viewer:        snapshot.Viewer{Login: "me", Followers: 4},
		Repos:         repos,
	}
}

func adjusted(mut func(*config.Config)) config.Config {
	c := config.Default()
	c.Languages.Mode = config.ModeAdjusted
	if mut != nil {
		mut(&c)
	}
	return c
}

func mustCompute(t *testing.T, s snapshot.Snapshot, cfg config.Config) report.Report {
	t.Helper()
	r, err := compute.Compute(s, history.History{}, cfg, now)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func share(t *testing.T, l []report.LangShare, name string) float64 {
	t.Helper()
	for _, e := range l {
		if e.Name == name {
			return e.Share
		}
	}
	t.Fatalf("%s not in %+v", name, l)
	return 0
}

func has(l []report.LangShare, name string) bool {
	for _, e := range l {
		if e.Name == name {
			return true
		}
	}
	return false
}

func sumShares(l []report.LangShare) float64 {
	var s float64
	for _, e := range l {
		s += e.Share
	}
	return s
}

func names(l []report.LangShare) string {
	var out []string
	for _, e := range l {
		out = append(out, e.Name)
	}
	return strings.Join(out, ",")
}

func TestReferenceCase(t *testing.T) {
	s := snap(repo("me/a", langs("Lua", 100000, "Rust", 50000, "TypeScript", 50000)))
	cfg := adjusted(func(c *config.Config) {
		c.Languages.Weights = map[string]float64{"Lua": 0.35, "TypeScript": 1.2}
	})
	r := mustCompute(t, s, cfg)

	if got := share(t, r.Languages.Raw, "Lua"); !near(got, 0.5) {
		t.Fatalf("raw Lua = %v", got)
	}
	if got, want := share(t, r.Languages.Adjusted, "Lua"), 35000.0/145000.0; !near(got, want) {
		t.Fatalf("adjusted Lua = %v, want %v", got, want)
	}
	if got, want := share(t, r.Languages.Adjusted, "TypeScript"), 60000.0/145000.0; !near(got, want) {
		t.Fatalf("adjusted TypeScript = %v, want %v", got, want)
	}
	if got := names(r.Languages.Adjusted); got != "TypeScript,Rust,Lua" {
		t.Fatalf("adjusted order = %s", got)
	}
	if got := names(r.Languages.Display); got != "TypeScript,Rust,Lua" {
		t.Fatalf("display = %s", got)
	}
	if !near(sumShares(r.Languages.Display), 1) || !near(sumShares(r.Languages.Raw), 1) {
		t.Fatalf("shares do not sum to 1")
	}
	if len(r.Languages.Multipliers) != 2 || r.Languages.Multipliers["Lua"] != 0.35 || r.Languages.Multipliers["TypeScript"] != 1.2 {
		t.Fatalf("multipliers = %v", r.Languages.Multipliers)
	}
	lua := r.Languages.Adjusted[2]
	if lua.Bytes != 100000 || !near(lua.Weighted, 35000) {
		t.Fatalf("lua = %+v", lua)
	}
}

func TestRawModeDisplaysRawAndHasNoMultipliers(t *testing.T) {
	s := snap(repo("me/a", langs("Lua", 100000, "Rust", 50000)))
	cfg := config.Default()
	cfg.Languages.Weights = map[string]float64{"Lua": 0.1}
	r := mustCompute(t, s, cfg)
	if got := share(t, r.Languages.Display, "Lua"); !near(got, 100000.0/150000.0) {
		t.Fatalf("display Lua = %v", got)
	}
	if len(r.Languages.Multipliers) != 0 {
		t.Fatalf("multipliers = %v", r.Languages.Multipliers)
	}
}

func TestLanguageNamesAreCaseInsensitive(t *testing.T) {
	s := snap(repo("me/a", langs("Lua", 100, "Rust", 100, "HTML", 100)))
	cfg := adjusted(func(c *config.Config) {
		c.Languages.Weights = map[string]float64{"lua": 0.5}
		c.Languages.Exclude = []string{"html"}
		c.Languages.Hide = []string{"RUST"}
	})
	r := mustCompute(t, s, cfg)
	if got := share(t, r.Languages.Adjusted, "Lua"); !near(got, 50.0/150.0) {
		t.Fatalf("adjusted Lua = %v", got)
	}
	if has(r.Languages.Adjusted, "HTML") {
		t.Fatal("HTML must be excluded")
	}
	if got := names(r.Languages.Display); got != "Lua,Other" {
		t.Fatalf("display = %s", got)
	}
	if r.Languages.Multipliers["Lua"] != 0.5 || r.Languages.Multipliers["HTML"] != 0 {
		t.Fatalf("multipliers = %v", r.Languages.Multipliers)
	}
}

func TestExcludeAffectsAdjustedOnly(t *testing.T) {
	s := snap(repo("me/a", langs("Go", 800, "HTML", 200)))
	cfg := adjusted(func(c *config.Config) { c.Languages.Exclude = []string{"HTML"} })
	r := mustCompute(t, s, cfg)
	if got := share(t, r.Languages.Raw, "HTML"); !near(got, 0.2) {
		t.Fatalf("raw HTML = %v", got)
	}
	if has(r.Languages.Adjusted, "HTML") || has(r.Languages.Display, "HTML") {
		t.Fatal("HTML must be absent from adjusted and display")
	}
	if got := share(t, r.Languages.Adjusted, "Go"); !near(got, 1) {
		t.Fatalf("adjusted Go = %v", got)
	}
}

func TestHiddenShare(t *testing.T) {
	s := snap(repo("me/a", langs("A", 50, "B", 30, "C", 20)))
	tests := []struct {
		name       string
		hiddenMode string
		wantNames  string
		wantShares map[string]float64
	}{
		{"other", config.HiddenShareOther, "A,B,Other", map[string]float64{"A": 0.5, "B": 0.3, "Other": 0.2}},
		{"renormalize", config.HiddenShareRenormalize, "A,B", map[string]float64{"A": 0.625, "B": 0.375}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Languages.Hide = []string{"C"}
			cfg.Languages.HiddenShare = tt.hiddenMode
			r := mustCompute(t, s, cfg)
			if got := names(r.Languages.Display); got != tt.wantNames {
				t.Fatalf("display = %s, want %s", got, tt.wantNames)
			}
			for name, want := range tt.wantShares {
				if got := share(t, r.Languages.Display, name); !near(got, want) {
					t.Fatalf("%s = %v, want %v", name, got, want)
				}
			}
			if !near(sumShares(r.Languages.Display), 1) {
				t.Fatalf("sum = %v", sumShares(r.Languages.Display))
			}
			if !near(sumShares(r.Languages.Raw), 1) || !has(r.Languages.Raw, "C") {
				t.Fatal("raw must keep hidden languages")
			}
		})
	}
}

func TestHideEverythingRenormalizeIsEmpty(t *testing.T) {
	s := snap(repo("me/a", langs("A", 50)))
	cfg := config.Default()
	cfg.Languages.Hide = []string{"A"}
	cfg.Languages.HiddenShare = config.HiddenShareRenormalize
	r := mustCompute(t, s, cfg)
	if len(r.Languages.Display) != 0 {
		t.Fatalf("display = %+v", r.Languages.Display)
	}
}

func TestMaxVisibleFoldsIntoOther(t *testing.T) {
	s := snap(repo("me/a", langs("A", 40, "B", 30, "C", 20, "D", 10)))
	tests := []struct {
		name      string
		max       int
		label     string
		wantNames string
		wantOther float64
	}{
		{"fold two", 2, "Other", "A,B,Other", 0.3},
		{"custom label", 3, "Misc", "A,B,C,Misc", 0.1},
		{"exact fit", 4, "Other", "A,B,C,D", 0},
		{"unlimited", 0, "Other", "A,B,C,D", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Languages.MaxVisible = tt.max
			cfg.Languages.OtherLabel = tt.label
			r := mustCompute(t, s, cfg)
			if got := names(r.Languages.Display); got != tt.wantNames {
				t.Fatalf("display = %s, want %s", got, tt.wantNames)
			}
			if tt.wantOther > 0 {
				last := r.Languages.Display[len(r.Languages.Display)-1]
				if last.Name != tt.label || !near(last.Share, tt.wantOther) {
					t.Fatalf("other = %+v", last)
				}
			}
			if !near(sumShares(r.Languages.Display), 1) {
				t.Fatalf("sum = %v", sumShares(r.Languages.Display))
			}
		})
	}
}

func TestHiddenAndOverflowShareOneOther(t *testing.T) {
	s := snap(repo("me/a", langs("A", 40, "B", 30, "C", 20, "D", 10)))
	cfg := config.Default()
	cfg.Languages.Hide = []string{"A"}
	cfg.Languages.MaxVisible = 1
	r := mustCompute(t, s, cfg)
	if got := names(r.Languages.Display); got != "B,Other" {
		t.Fatalf("display = %s", got)
	}
	if got := share(t, r.Languages.Display, "Other"); !near(got, 0.7) {
		t.Fatalf("other = %v", got)
	}
}

func TestRepoWeight(t *testing.T) {
	s := snap(
		repo("me/real", langs("Go", 100)),
		repo("me/generated", langs("Go", 100, "Perl", 500)),
		repo("me/Half", langs("Rust", 100)),
	)
	cfg := adjusted(func(c *config.Config) {
		c.Scope.RepoWeights = map[string]float64{"me/generated": 0, "ME/HALF": 0.5}
	})
	r := mustCompute(t, s, cfg)
	if has(r.Languages.Adjusted, "Perl") {
		t.Fatal("Perl only lives in a repo of weight 0")
	}
	if got, want := share(t, r.Languages.Adjusted, "Go"), 100.0/150.0; !near(got, want) {
		t.Fatalf("Go = %v, want %v", got, want)
	}
	if got, want := share(t, r.Languages.Adjusted, "Rust"), 50.0/150.0; !near(got, want) {
		t.Fatalf("Rust = %v, want %v", got, want)
	}
	if got := share(t, r.Languages.Raw, "Perl"); !near(got, 500.0/800.0) {
		t.Fatalf("raw Perl = %v", got)
	}
}

func TestZeroWeightedTotal(t *testing.T) {
	s := snap(repo("me/a", langs("Go", 100)))
	cfg := adjusted(func(c *config.Config) { c.Languages.Weights = map[string]float64{"Go": 0} })
	r := mustCompute(t, s, cfg)
	if len(r.Languages.Adjusted) != 0 || len(r.Languages.Display) != 0 {
		t.Fatalf("adjusted=%+v display=%+v", r.Languages.Adjusted, r.Languages.Display)
	}
	if len(r.Languages.Raw) != 1 {
		t.Fatalf("raw = %+v", r.Languages.Raw)
	}
	if len(r.Warnings) == 0 || !strings.Contains(r.Warnings[0], "weighted total is zero") {
		t.Fatalf("warnings = %v", r.Warnings)
	}
}

func TestScopeFilters(t *testing.T) {
	mk := func(name string, mut func(*snapshot.Repo)) snapshot.Repo {
		r := repo(name, langs("Go", 100))
		r.Stars, r.Forks = 2, 1
		if mut != nil {
			mut(&r)
		}
		return r
	}
	s := snap(
		mk("me/plain", nil),
		mk("me/fork", func(r *snapshot.Repo) { r.Fork = true }),
		mk("me/archived", func(r *snapshot.Repo) { r.Archived = true }),
		mk("me/private", func(r *snapshot.Repo) { r.Private = true }),
		mk("Org/team", nil),
		mk("me/legacy-old", nil),
	)
	tests := []struct {
		name        string
		mutate      func(*config.Config)
		wantRepos   int
		wantPrivate int
	}{
		{"defaults", nil, 4, 1},
		{"forks", func(c *config.Config) { c.Scope.IncludeForks = true }, 5, 1},
		{"archived", func(c *config.Config) { c.Scope.IncludeArchived = true }, 5, 1},
		{"no private", func(c *config.Config) { c.Scope.IncludePrivate = false }, 3, 0},
		{"owner filter", func(c *config.Config) { c.Scope.Owners = []string{"ORG"} }, 1, 0},
		{"exclude exact", func(c *config.Config) { c.Scope.ExcludeRepos = []string{"ME/Plain"} }, 3, 1},
		{"exclude glob", func(c *config.Config) { c.Scope.ExcludeRepos = []string{"me/*-old", "org/*"} }, 2, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}
			r := mustCompute(t, s, cfg)
			got := r.Totals
			if got.Repos != tt.wantRepos || got.PrivateRepos != tt.wantPrivate || got.PublicRepos != tt.wantRepos-tt.wantPrivate {
				t.Fatalf("totals = %+v", got)
			}
			if got.Stars != 2*tt.wantRepos || got.Forks != tt.wantRepos || got.Followers != 4 {
				t.Fatalf("totals = %+v", got)
			}
			if got := r.Languages.Raw[0].Bytes; got != int64(100*tt.wantRepos) {
				t.Fatalf("raw bytes = %d", got)
			}
		})
	}
}

func TestInvalidConfigIsRejected(t *testing.T) {
	cfg := config.Default()
	cfg.Scope.ExcludeRepos = []string{"[bad"}
	if _, err := compute.Compute(snap(), history.History{}, cfg, now); err == nil {
		t.Fatal("expected an error")
	}
}

func day(m, d int) time.Time { return time.Date(2026, time.Month(m), d, 0, 0, 0, 0, time.UTC) }

func dc(m, d, count int) snapshot.DayCount {
	return snapshot.DayCount{Day: day(m, d), Count: count, Uniques: 1}
}

func metric(t *testing.T, r report.Report, channel, window string) report.Metric {
	t.Helper()
	for _, m := range r.Distribution {
		if m.Channel == channel && (window == "" || m.Window == window || (window == "since" && strings.HasPrefix(m.Window, "since:"))) {
			return m
		}
	}
	t.Fatalf("no metric %s/%s in %+v", channel, window, r.Distribution)
	return report.Metric{}
}

func count(t *testing.T, m report.Metric) int64 {
	t.Helper()
	if m.Count == nil {
		t.Fatalf("metric %+v has no count", m)
	}
	return *m.Count
}

func withRelease(r snapshot.Repo, downloads ...int64) snapshot.Repo {
	for i, d := range downloads {
		r.Releases = append(r.Releases, snapshot.Release{
			Tag:    "v" + string(rune('0'+i)),
			Assets: []snapshot.Asset{{Name: "a.zip", DownloadCount: d}, {Name: "b.zip", DownloadCount: d * 10}},
		})
	}
	return r
}

func TestReleaseAssets(t *testing.T) {
	hidden := withRelease(repo("me/fork", nil), 1000)
	hidden.Fork = true
	s := snap(withRelease(repo("me/a", nil), 1, 2), withRelease(repo("me/b", nil), 4), hidden)
	r := mustCompute(t, s, config.Default())
	m := metric(t, r, report.ChannelReleaseAssets, "")
	if got := count(t, m); got != 3+30+4+40 {
		t.Fatalf("release assets = %d", got)
	}
	if m.Window != "all_time" || m.Reliability != "observed" || m.Note != "GitHub Release assets only" {
		t.Fatalf("metric = %+v", m)
	}
}

func TestTrafficWindows(t *testing.T) {
	a := repo("me/a", nil)
	a.Traffic = &snapshot.Traffic{
		Views:  []snapshot.DayCount{dc(11, 3, 1000), dc(11, 4, 1), dc(11, 10, 2), dc(11, 17, 4), dc(11, 18, 10000)},
		Clones: []snapshot.DayCount{dc(11, 17, 7)},
	}
	b := repo("me/b", nil)
	b.Traffic = &snapshot.Traffic{Views: []snapshot.DayCount{dc(11, 4, 8)}}
	r := mustCompute(t, snap(a, b), config.Default())

	if got := count(t, metric(t, r, report.ChannelViews, "last_14_days")); got != 1+2+4+8 {
		t.Fatalf("views 14d = %d", got)
	}
	since := metric(t, r, report.ChannelViews, "since")
	if since.Window != "since:2026-11-03" || count(t, since) != 1000+1+2+4+10000+8 || since.Reliability != "observed" {
		t.Fatalf("views since = %+v", since)
	}
	if got := count(t, metric(t, r, report.ChannelClones, "last_14_days")); got != 7 {
		t.Fatalf("clones 14d = %d", got)
	}
	if got := metric(t, r, report.ChannelClones, "since").Window; got != "since:2026-11-17" {
		t.Fatalf("clones since = %s", got)
	}
}

func TestTrafficUsesHistoryBeyondSnapshotWindow(t *testing.T) {
	a := repo("me/a", nil)
	a.Traffic = &snapshot.Traffic{Views: []snapshot.DayCount{dc(11, 16, 5)}}
	h := history.History{Repos: map[string]history.RepoHistory{
		a.ID:      {NameWithOwner: a.NameWithOwner, Views: []snapshot.DayCount{dc(9, 1, 100), dc(11, 16, 99)}},
		"ID_gone": {NameWithOwner: "me/gone", Views: []snapshot.DayCount{dc(8, 1, 7777)}},
	}}
	r, err := compute.Compute(snap(a), h, config.Default(), now)
	if err != nil {
		t.Fatal(err)
	}
	since := metric(t, r, report.ChannelViews, "since")
	if since.Window != "since:2026-09-01" || count(t, since) != 105 {
		t.Fatalf("since = %+v count=%d", since, count(t, since))
	}
	if got := count(t, metric(t, r, report.ChannelViews, "last_14_days")); got != 5 {
		t.Fatalf("14d = %d", got)
	}
}

func TestTrafficPartialWhenARepoLacksTraffic(t *testing.T) {
	a := repo("me/a", nil)
	a.Traffic = &snapshot.Traffic{Views: []snapshot.DayCount{dc(11, 16, 5)}}
	b := repo("me/b", nil)
	s := snap(a, b)
	s.Errors = []snapshot.RepoError{{Repo: "me/b", Stage: "traffic", Message: "403"}}
	r := mustCompute(t, s, config.Default())
	if got := metric(t, r, report.ChannelViews, "since").Reliability; got != "partial" {
		t.Fatalf("reliability = %s", got)
	}
	if got := metric(t, r, report.ChannelViews, "last_14_days").Reliability; got != "observed" {
		t.Fatalf("14d reliability = %s", got)
	}
	want := []string{"1 traffic collection error", "1 of 2 repositories have no traffic data in this snapshot"}
	if strings.Join(r.Warnings, "|") != strings.Join(want, "|") {
		t.Fatalf("warnings = %v", r.Warnings)
	}
}

func TestTrafficUnavailableWithoutAnyData(t *testing.T) {
	r := mustCompute(t, snap(repo("me/a", nil)), config.Default())
	for _, m := range r.Distribution {
		if (m.Channel == report.ChannelClones || m.Channel == report.ChannelViews) && (m.Count != nil || m.Reliability != "unavailable") {
			t.Fatalf("metric = %+v", m)
		}
	}
}

func TestSourceArchivesAlwaysUnavailable(t *testing.T) {
	cfg := config.Default()
	cfg.Distribution = config.Distribution{}
	r := mustCompute(t, snap(repo("me/a", nil)), cfg)
	if len(r.Distribution) != 1 {
		t.Fatalf("distribution = %+v", r.Distribution)
	}
	m := r.Distribution[0]
	if m.Channel != "source_archives" || m.Count != nil || m.Reliability != "unavailable" ||
		m.Note != "GitHub exposes no download counter for source archives" {
		t.Fatalf("metric = %+v", m)
	}
}

func TestDisabledChannelsAreAbsent(t *testing.T) {
	a := repo("me/a", nil)
	a.Traffic = &snapshot.Traffic{Views: []snapshot.DayCount{dc(11, 16, 5)}, Clones: []snapshot.DayCount{dc(11, 16, 5)}}
	s := snap(withRelease(a, 1))
	tests := []struct {
		name string
		dist config.Distribution
		want string
	}{
		{"all", config.Distribution{ReleaseAssets: true, Clones: true, Views: true}, "release_assets,clones,clones,views,views,source_archives"},
		{"no release", config.Distribution{Clones: true, Views: true}, "clones,clones,views,views,source_archives"},
		{"no clones", config.Distribution{ReleaseAssets: true, Views: true}, "release_assets,views,views,source_archives"},
		{"no views", config.Distribution{ReleaseAssets: true, Clones: true}, "release_assets,clones,clones,source_archives"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Distribution = tt.dist
			r := mustCompute(t, s, cfg)
			var got []string
			for _, m := range r.Distribution {
				got = append(got, m.Channel)
			}
			if strings.Join(got, ",") != tt.want {
				t.Fatalf("channels = %v, want %s", got, tt.want)
			}
		})
	}
}

func TestReportDoesNotLeakPrivateRepositories(t *testing.T) {
	secret := repo("secret-owner/secret-repo", langs("Zig", 5000, "Go", 100))
	secret.Private = true
	secret.Stars = 9
	secret.Releases = []snapshot.Release{{Tag: "secret-tag", Assets: []snapshot.Asset{{Name: "secret-asset.zip", DownloadCount: 3}}}}
	secret.Traffic = &snapshot.Traffic{Views: []snapshot.DayCount{dc(11, 16, 5)}}
	public := repo("me/open", langs("Go", 1000))
	s := snap(secret, public)
	s.Errors = []snapshot.RepoError{
		{Repo: "secret-owner/secret-repo", Stage: "releases", Message: "secret-owner/secret-repo: boom"},
		{Repo: "secret-owner/secret-repo", Stage: "traffic", Message: "secret-owner/secret-repo: forbidden"},
	}
	h := history.History{Repos: map[string]history.RepoHistory{
		secret.ID: {NameWithOwner: secret.NameWithOwner, Private: true, Views: []snapshot.DayCount{dc(10, 1, 1)}},
	}}

	for _, includePrivate := range []bool{true, false} {
		cfg := adjusted(func(c *config.Config) { c.Scope.IncludePrivate = includePrivate })
		r, err := compute.Compute(s, h, cfg, now)
		if err != nil {
			t.Fatal(err)
		}
		out, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		for _, banned := range []string{"secret", "open", "me/"} {
			if strings.Contains(string(out), banned) {
				t.Fatalf("include_private=%v: report contains %q: %s", includePrivate, banned, out)
			}
		}
		if !strings.Contains(string(out), `"login":"me"`) {
			t.Fatalf("login missing: %s", out)
		}
	}
}
