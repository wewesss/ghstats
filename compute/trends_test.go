package compute_test

import (
	"testing"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/report"
	"github.com/wewesss/ghstats/snapshot"
)

func trendOf(t *testing.T, r report.Report, channel string) report.Trend {
	t.Helper()
	for _, tr := range r.Trends {
		if tr.Channel == channel {
			return tr
		}
	}
	t.Fatalf("no trend for %s in %+v", channel, r.Trends)
	return report.Trend{}
}

func weekOf(t *testing.T, tr report.Trend, start string) int64 {
	t.Helper()
	for _, w := range tr.Weeks {
		if w.Start == start {
			return w.Count
		}
	}
	t.Fatalf("week %s not in trend", start)
	return 0
}

func TestTrendBucketsAreMondayBasedAndCoverTwentySixWeeks(t *testing.T) {
	a := repo("me/a", nil)
	a.Traffic = &snapshot.Traffic{Views: []snapshot.DayCount{
		dc(11, 15, 1),
		dc(11, 16, 2),
		dc(11, 17, 4),
		dc(11, 9, 8),
		dc(5, 25, 16),
		dc(5, 24, 32),
	}}
	r := mustCompute(t, snap(a), config.Default())
	tr := trendOf(t, r, report.ChannelViews)

	if len(tr.Weeks) != 26 {
		t.Fatalf("weeks = %d", len(tr.Weeks))
	}
	if tr.Weeks[0].Start != "2026-05-25" || tr.Weeks[25].Start != "2026-11-16" {
		t.Fatalf("range = %s .. %s", tr.Weeks[0].Start, tr.Weeks[25].Start)
	}
	if got := weekOf(t, tr, "2026-11-16"); got != 6 {
		t.Fatalf("current week = %d, want 6 (Monday and Tuesday)", got)
	}
	if got := weekOf(t, tr, "2026-11-09"); got != 9 {
		t.Fatalf("previous week = %d, want 9 (Sunday folds into the week before)", got)
	}
	if got := weekOf(t, tr, "2026-05-25"); got != 16 {
		t.Fatalf("oldest week = %d, want 16 (the Sunday before it is out of range)", got)
	}
	if got := weekOf(t, tr, "2026-11-02"); got != 0 {
		t.Fatalf("empty week = %d", got)
	}
}

func TestTrendAppliesScopeAndUsesBothChannels(t *testing.T) {
	a := repo("me/a", nil)
	a.Traffic = &snapshot.Traffic{Views: []snapshot.DayCount{dc(11, 16, 3)}, Clones: []snapshot.DayCount{dc(11, 16, 5)}}
	b := repo("me/b", nil)
	b.Private = true
	b.Traffic = &snapshot.Traffic{Views: []snapshot.DayCount{dc(11, 16, 100)}}
	cfg := config.Default()
	cfg.Scope.IncludePrivate = false
	r := mustCompute(t, snap(a, b), cfg)

	if got := weekOf(t, trendOf(t, r, report.ChannelViews), "2026-11-16"); got != 3 {
		t.Fatalf("views = %d, want 3 (private repo out of scope)", got)
	}
	if got := weekOf(t, trendOf(t, r, report.ChannelClones), "2026-11-16"); got != 5 {
		t.Fatalf("clones = %d", got)
	}
}

func TestTrendDisabledChannelIsAbsent(t *testing.T) {
	a := repo("me/a", nil)
	a.Traffic = &snapshot.Traffic{Views: []snapshot.DayCount{dc(11, 16, 3)}, Clones: []snapshot.DayCount{dc(11, 16, 5)}}
	cfg := config.Default()
	cfg.Distribution.Clones = false
	r := mustCompute(t, snap(a), cfg)
	if len(r.Trends) != 1 || r.Trends[0].Channel != report.ChannelViews {
		t.Fatalf("trends = %+v", r.Trends)
	}
	cfg.Distribution.Views = false
	r = mustCompute(t, snap(a), cfg)
	if r.Trends == nil || len(r.Trends) != 0 {
		t.Fatalf("trends = %#v, want empty non-nil", r.Trends)
	}
}

func TestTrendDaysCoverTwentyEightDaysZeroFilled(t *testing.T) {
	a := repo("me/a", nil)
	a.Traffic = &snapshot.Traffic{Views: []snapshot.DayCount{
		dc(11, 17, 4),
		dc(11, 16, 2),
		dc(10, 21, 8),
		dc(10, 20, 16),
	}}
	b := repo("me/b", nil)
	b.Traffic = &snapshot.Traffic{Views: []snapshot.DayCount{dc(11, 17, 1)}}
	r := mustCompute(t, snap(a, b), config.Default())
	tr := trendOf(t, r, report.ChannelViews)

	if len(tr.Days) != 28 {
		t.Fatalf("days = %d", len(tr.Days))
	}
	if tr.Days[0].Day != "2026-10-21" || tr.Days[27].Day != "2026-11-17" {
		t.Fatalf("range = %s .. %s", tr.Days[0].Day, tr.Days[27].Day)
	}
	if tr.Days[27].Count != 5 {
		t.Fatalf("today = %d, want 5 (summed across repositories)", tr.Days[27].Count)
	}
	if tr.Days[26].Count != 2 {
		t.Fatalf("yesterday = %d", tr.Days[26].Count)
	}
	if tr.Days[0].Count != 8 {
		t.Fatalf("oldest day = %d, want 8 (the day before is out of range)", tr.Days[0].Count)
	}
	if tr.Days[10].Count != 0 {
		t.Fatalf("empty day = %d", tr.Days[10].Count)
	}
	var sum int64
	for _, d := range tr.Days {
		sum += d.Count
	}
	if sum != 15 {
		t.Fatalf("sum = %d, want 15", sum)
	}
}
