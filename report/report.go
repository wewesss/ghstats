// Package report defines the computed output model that renderers consume.
package report

import "time"

const SchemaVersion = 1

const (
	ChannelReleaseAssets  = "release_assets"
	ChannelClones         = "clones"
	ChannelViews          = "views"
	ChannelSourceArchives = "source_archives"

	WindowAllTime     = "all_time"
	WindowLast14Days  = "last_14_days"
	WindowSincePrefix = "since:"

	ReliabilityObserved    = "observed"
	ReliabilityPartial     = "partial"
	ReliabilityUnavailable = "unavailable"
)

type Report struct {
	SchemaVersion int       `json:"schema_version"`
	GeneratedAt   time.Time `json:"generated_at"`
	CollectedAt   time.Time `json:"collected_at"`
	Login         string    `json:"login"`
	Totals        Totals    `json:"totals"`
	Languages     Languages `json:"languages"`
	Distribution  []Metric  `json:"distribution"`
	Trends        []Trend   `json:"trends"`
	Warnings      []string  `json:"warnings"`
}

type Totals struct {
	Repos        int `json:"repos"`
	PublicRepos  int `json:"public_repos"`
	PrivateRepos int `json:"private_repos"`
	Stars        int `json:"stars"`
	Forks        int `json:"forks"`
	Followers    int `json:"followers"`
}

type Languages struct {
	Mode        string             `json:"mode"`
	Raw         []LangShare        `json:"raw"`
	Adjusted    []LangShare        `json:"adjusted"`
	Display     []LangShare        `json:"display"`
	Multipliers map[string]float64 `json:"multipliers"`
	HiddenShare string             `json:"hidden_share"`
}

type LangShare struct {
	Name     string  `json:"name"`
	Bytes    int64   `json:"bytes"`
	Weighted float64 `json:"weighted"`
	Share    float64 `json:"share"`
}

type Trend struct {
	Channel string      `json:"channel"`
	Weeks   []WeekCount `json:"weeks"`
	Days    []DayCount  `json:"days"`
}

type DayCount struct {
	Day   string `json:"day"`
	Count int64  `json:"count"`
}

type WeekCount struct {
	Start string `json:"start"`
	Count int64  `json:"count"`
}

type Metric struct {
	Channel     string `json:"channel"`
	Count       *int64 `json:"count"`
	Window      string `json:"window"`
	Reliability string `json:"reliability"`
	Note        string `json:"note,omitempty"`
}
