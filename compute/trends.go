package compute

import (
	"time"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/history"
	"github.com/wewesss/ghstats/report"
	"github.com/wewesss/ghstats/snapshot"
)

const (
	trendWeeks = 26
	trendDays  = 28
)

func trends(repos []snapshot.Repo, h history.History, dc config.Distribution, now time.Time) []report.Trend {
	out := []report.Trend{}
	if dc.Clones {
		out = append(out, trend(report.ChannelClones, repos, h, now, func(rh history.RepoHistory) []snapshot.DayCount { return rh.Clones }))
	}
	if dc.Views {
		out = append(out, trend(report.ChannelViews, repos, h, now, func(rh history.RepoHistory) []snapshot.DayCount { return rh.Views }))
	}
	return out
}

func weekStart(t time.Time) time.Time {
	day := t.UTC().Truncate(24 * time.Hour)
	return day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
}

func trend(channel string, repos []snapshot.Repo, h history.History, now time.Time, series func(history.RepoHistory) []snapshot.DayCount) report.Trend {
	last := weekStart(now)
	first := last.AddDate(0, 0, -7*(trendWeeks-1))
	weeks := make([]report.WeekCount, trendWeeks)
	for i := range weeks {
		weeks[i].Start = first.AddDate(0, 0, 7*i).Format(dayLayout)
	}
	today := now.UTC().Truncate(24 * time.Hour)
	firstDay := today.AddDate(0, 0, -(trendDays - 1))
	days := make([]report.DayCount, trendDays)
	for i := range days {
		days[i].Day = firstDay.AddDate(0, 0, i).Format(dayLayout)
	}
	for _, r := range repos {
		rh, ok := h.Repos[r.ID]
		if !ok {
			continue
		}
		for _, d := range series(rh) {
			day := d.Day.UTC().Truncate(24 * time.Hour)
			if !day.Before(firstDay) && !day.After(today) {
				days[int(day.Sub(firstDay)/(24*time.Hour))].Count += int64(d.Count)
			}
			if day.Before(first) || day.After(last.AddDate(0, 0, 6)) {
				continue
			}
			weeks[int(weekStart(day).Sub(first)/(7*24*time.Hour))].Count += int64(d.Count)
		}
	}
	return report.Trend{Channel: channel, Weeks: weeks, Days: days}
}
