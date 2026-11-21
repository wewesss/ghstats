package history_test

import (
	"fmt"
	"time"

	"github.com/wewesss/ghstats/history"
	"github.com/wewesss/ghstats/snapshot"
)

func ExampleMerge() {
	day := func(d int) time.Time { return time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC) }
	repo := func(views ...snapshot.DayCount) snapshot.Snapshot {
		return snapshot.Snapshot{Repos: []snapshot.Repo{{
			ID:            "1",
			NameWithOwner: "octocat/hello",
			Traffic:       &snapshot.Traffic{Views: views},
		}}}
	}
	h := history.Merge(history.History{}, repo(
		snapshot.DayCount{Day: day(1), Count: 4},
		snapshot.DayCount{Day: day(2), Count: 5},
	))
	h = history.Merge(h, repo(
		snapshot.DayCount{Day: day(2), Count: 6},
		snapshot.DayCount{Day: day(3), Count: 7},
	))
	for _, v := range h.Repos["1"].Views {
		fmt.Println(v.Day.Format("2006-01-02"), v.Count)
	}
	// Output:
	// 2026-11-01 4
	// 2026-11-02 6
	// 2026-11-03 7
}
