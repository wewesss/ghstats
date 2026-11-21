package compute_test

import (
	"fmt"
	"time"

	"github.com/wewesss/ghstats/compute"
	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/history"
	"github.com/wewesss/ghstats/snapshot"
)

func ExampleCompute() {
	now := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	snap := snapshot.Snapshot{
		SchemaVersion: snapshot.SchemaVersion,
		CollectedAt:   now,
		Viewer:        snapshot.Viewer{Login: "octocat", Followers: 3},
		Repos: []snapshot.Repo{{
			ID:            "1",
			NameWithOwner: "octocat/hello",
			Owner:         "octocat",
			Stars:         12,
			Languages:     []snapshot.LanguageSize{{Name: "Go", Bytes: 900}, {Name: "Shell", Bytes: 100}},
		}},
	}
	rep, err := compute.Compute(snap, history.History{}, config.Default(), now)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(rep.Login, rep.Totals.Repos, rep.Totals.Stars)
	for _, l := range rep.Languages.Display {
		fmt.Printf("%s %.0f%%\n", l.Name, l.Share*100)
	}
	// Output:
	// octocat 1 12
	// Go 90%
	// Shell 10%
}
