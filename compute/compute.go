// Package compute turns a snapshot, a traffic history and a configuration into a report.
package compute

import (
	"cmp"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/history"
	"github.com/wewesss/ghstats/report"
	"github.com/wewesss/ghstats/snapshot"
)

const (
	trafficWindowDays = 14
	dayLayout         = "2006-01-02"
)

func Compute(s snapshot.Snapshot, h history.History, cfg config.Config, now time.Time) (report.Report, error) {
	if err := cfg.Validate(); err != nil {
		return report.Report{}, err
	}
	repos := scoped(s.Repos, cfg.Scope)

	r := report.Report{
		SchemaVersion: report.SchemaVersion,
		GeneratedAt:   now,
		CollectedAt:   s.CollectedAt,
		Login:         s.Viewer.Login,
		Totals:        totals(repos, s.Viewer.Followers),
		Warnings:      []string{},
	}
	r.Languages = languages(repos, cfg, &r.Warnings)
	r.Distribution = distribution(repos, history.Merge(h, s), cfg.Distribution, now)
	r.Warnings = append(r.Warnings, collectionWarnings(s.Errors, repos, cfg.Distribution)...)
	return r, nil
}

func scoped(all []snapshot.Repo, sc config.Scope) []snapshot.Repo {
	owners := make(map[string]bool, len(sc.Owners))
	for _, o := range sc.Owners {
		owners[strings.ToLower(o)] = true
	}
	patterns := make([]string, len(sc.ExcludeRepos))
	for i, p := range sc.ExcludeRepos {
		patterns[i] = strings.ToLower(p)
	}
	var out []snapshot.Repo
	for _, r := range all {
		switch {
		case len(owners) > 0 && !owners[strings.ToLower(r.Owner)]:
		case r.Fork && !sc.IncludeForks:
		case r.Archived && !sc.IncludeArchived:
		case r.Private && !sc.IncludePrivate:
		case excluded(patterns, strings.ToLower(r.NameWithOwner)):
		default:
			out = append(out, r)
		}
	}
	return out
}

func excluded(patterns []string, name string) bool {
	return slices.ContainsFunc(patterns, func(p string) bool {
		ok, _ := path.Match(p, name)
		return ok
	})
}

func totals(repos []snapshot.Repo, followers int) report.Totals {
	t := report.Totals{Repos: len(repos), Followers: followers}
	for _, r := range repos {
		if r.Private {
			t.PrivateRepos++
		} else {
			t.PublicRepos++
		}
		t.Stars += r.Stars
		t.Forks += r.Forks
	}
	return t
}

func languages(repos []snapshot.Repo, cfg config.Config, warnings *[]string) report.Languages {
	lc := cfg.Languages
	out := report.Languages{
		Mode:        lc.Mode,
		Multipliers: map[string]float64{},
		HiddenShare: lc.HiddenShare,
	}

	bytesByLang := map[string]int64{}
	for _, r := range repos {
		for _, l := range r.Languages {
			bytesByLang[l.Name] += l.Bytes
		}
	}
	rawValues := make(map[string]float64, len(bytesByLang))
	for name, b := range bytesByLang {
		rawValues[name] = float64(b)
	}
	out.Raw = shares(rawValues, bytesByLang)

	langWeights := foldKeys(lc.Weights)
	for _, name := range lc.Exclude {
		langWeights[strings.ToLower(name)] = 0
	}
	repoWeights := foldKeys(cfg.Scope.RepoWeights)
	weighted := map[string]float64{}
	for _, r := range repos {
		rw := weightOr1(repoWeights, r.NameWithOwner)
		for _, l := range r.Languages {
			weighted[l.Name] += float64(l.Bytes) * rw * weightOr1(langWeights, l.Name)
		}
	}
	out.Adjusted = shares(weighted, bytesByLang)

	if lc.Mode == config.ModeAdjusted {
		for name := range bytesByLang {
			if w := weightOr1(langWeights, name); w != 1 {
				out.Multipliers[name] = w
			}
		}
		if len(out.Adjusted) == 0 {
			*warnings = append(*warnings, "adjusted language shares are unavailable: the weighted total is zero")
		}
	}

	source := out.Raw
	if lc.Mode == config.ModeAdjusted {
		source = out.Adjusted
	}
	out.Display = display(source, lc)
	return out
}

func shares(values map[string]float64, bytesByLang map[string]int64) []report.LangShare {
	var total float64
	for _, v := range values {
		total += v
	}
	out := []report.LangShare{}
	if total <= 0 {
		return out
	}
	for name, v := range values {
		if v <= 0 {
			continue
		}
		out = append(out, report.LangShare{Name: name, Bytes: bytesByLang[name], Weighted: v, Share: v / total})
	}
	slices.SortFunc(out, func(a, b report.LangShare) int {
		return cmp.Or(cmp.Compare(b.Share, a.Share), cmp.Compare(a.Name, b.Name))
	})
	return out
}

func display(source []report.LangShare, lc config.Languages) []report.LangShare {
	hidden := map[string]bool{}
	for _, name := range lc.Hide {
		hidden[strings.ToLower(name)] = true
	}
	visible := []report.LangShare{}
	other := report.LangShare{Name: lc.OtherLabel}
	for _, l := range source {
		if !hidden[strings.ToLower(l.Name)] {
			visible = append(visible, l)
		} else if lc.HiddenShare == config.HiddenShareOther {
			other = fold(other, l)
		}
	}

	if lc.HiddenShare == config.HiddenShareRenormalize {
		var sum float64
		for _, l := range visible {
			sum += l.Share
		}
		for i := range visible {
			visible[i].Share /= sum
		}
	}

	if lc.MaxVisible > 0 && len(visible) > lc.MaxVisible {
		for _, l := range visible[lc.MaxVisible:] {
			other = fold(other, l)
		}
		visible = visible[:lc.MaxVisible]
	}
	if other.Share > 0 {
		visible = append(visible, other)
	}
	return visible
}

func fold(into, l report.LangShare) report.LangShare {
	into.Bytes += l.Bytes
	into.Weighted += l.Weighted
	into.Share += l.Share
	return into
}

func foldKeys(m map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(m))
	for k, v := range m {
		out[strings.ToLower(k)] = v
	}
	return out
}

func weightOr1(weights map[string]float64, name string) float64 {
	if w, ok := weights[strings.ToLower(name)]; ok {
		return w
	}
	return 1
}

func distribution(repos []snapshot.Repo, h history.History, dc config.Distribution, now time.Time) []report.Metric {
	var out []report.Metric
	if dc.ReleaseAssets {
		var total int64
		for _, r := range repos {
			for _, rel := range r.Releases {
				for _, a := range rel.Assets {
					total += a.DownloadCount
				}
			}
		}
		out = append(out, report.Metric{
			Channel:     report.ChannelReleaseAssets,
			Count:       &total,
			Window:      report.WindowAllTime,
			Reliability: report.ReliabilityObserved,
			Note:        "GitHub Release assets only",
		})
	}
	if dc.Clones {
		out = append(out, trafficMetrics(report.ChannelClones, repos, h, now, func(rh history.RepoHistory) []snapshot.DayCount { return rh.Clones })...)
	}
	if dc.Views {
		out = append(out, trafficMetrics(report.ChannelViews, repos, h, now, func(rh history.RepoHistory) []snapshot.DayCount { return rh.Views })...)
	}
	out = append(out, report.Metric{
		Channel:     report.ChannelSourceArchives,
		Window:      report.WindowAllTime,
		Reliability: report.ReliabilityUnavailable,
		Note:        "GitHub exposes no download counter for source archives",
	})
	return out
}

func trafficMetrics(channel string, repos []snapshot.Repo, h history.History, now time.Time, series func(history.RepoHistory) []snapshot.DayCount) []report.Metric {
	today := now.UTC().Truncate(24 * time.Hour)
	windowStart := today.AddDate(0, 0, -(trafficWindowDays - 1))

	var recent, all int64
	var first time.Time
	hasData, missing := false, false
	for _, r := range repos {
		if r.Traffic == nil {
			missing = true
		}
		rh, ok := h.Repos[r.ID]
		if !ok {
			continue
		}
		for _, d := range series(rh) {
			hasData = true
			all += int64(d.Count)
			if first.IsZero() || d.Day.Before(first) {
				first = d.Day
			}
			if !d.Day.Before(windowStart) && !d.Day.After(today) {
				recent += int64(d.Count)
			}
		}
	}

	if !hasData {
		note := "no traffic data collected"
		return []report.Metric{
			{Channel: channel, Window: report.WindowLast14Days, Reliability: report.ReliabilityUnavailable, Note: note},
			{Channel: channel, Window: report.WindowAllTime, Reliability: report.ReliabilityUnavailable, Note: note},
		}
	}

	reliability, note := report.ReliabilityObserved, ""
	if missing {
		reliability, note = report.ReliabilityPartial, "some repositories have no traffic data"
	}
	return []report.Metric{
		{Channel: channel, Count: &recent, Window: report.WindowLast14Days, Reliability: report.ReliabilityObserved, Note: "summed across repositories; unique visitors are not summed"},
		{Channel: channel, Count: &all, Window: report.WindowSincePrefix + first.UTC().Format(dayLayout), Reliability: reliability, Note: note},
	}
}

func collectionWarnings(errs []snapshot.RepoError, repos []snapshot.Repo, dc config.Distribution) []string {
	names := make(map[string]bool, len(repos))
	for _, r := range repos {
		names[strings.ToLower(r.NameWithOwner)] = true
	}
	byStage := map[string]int{}
	for _, e := range errs {
		if names[strings.ToLower(e.Repo)] {
			byStage[e.Stage]++
		}
	}
	var out []string
	for _, stage := range slices.Sorted(maps.Keys(byStage)) {
		n := byStage[stage]
		out = append(out, fmt.Sprintf("%d %s collection %s", n, stage, plural(n, "error", "errors")))
	}
	if dc.Clones || dc.Views {
		missing := 0
		for _, r := range repos {
			if r.Traffic == nil {
				missing++
			}
		}
		if missing > 0 {
			out = append(out, fmt.Sprintf("%d of %d repositories have no traffic data in this snapshot", missing, len(repos)))
		}
	}
	return out
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
