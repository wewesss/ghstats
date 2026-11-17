// Package sample provides the fixed report and configurations used by the render tests, the showcase generator and the WASM builder.
package sample

import (
	"time"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/report"
)

func AllBlocks() []config.Block {
	return []config.Block{
		{Name: "stats", Variants: []string{"totals", "distribution"}},
		{Name: "languages", Variants: []string{"bars", "iso", "cells", "treemap", "shift", "terminal"}},
		{Name: "history", Variants: []string{"views", "clones"}},
	}
}

func FullRender() config.Render {
	c := config.DefaultRender()
	c.Blocks = AllBlocks()
	c.Theme.Accent = config.Accent{Name: "cycle"}
	return c
}

func WideRender() config.Render {
	c := FullRender()
	c.Layout = config.LayoutWide
	return c
}

func Profile() config.Config {
	c := config.Default()
	c.Render.Theme.Accent = config.Accent{Name: "cycle"}
	c.Profile = config.Profile{
		Name:     "wewesss",
		Tagline:  "SOFTWARE · GAMES · EXPERIMENTS",
		Location: "caen, fr",
		Note:     "no fixed stack — it depends on what I'm making",
		Focus: []config.Focus{
			{Name: "WEB & DESKTOP", Building: []string{"full-stack apps", "desktop software"}},
			{Name: "GAMES", Building: []string{"gameplay systems", "game engines"}},
			{Name: "AI & AUTOMATION", Building: []string{"coding agents", "automated workflows"}},
			{Name: "SYSTEMS", Building: []string{"runtimes & compilers", "native tooling"}},
		},
		Links: []config.Link{
			{Kind: "youtube", Value: "@wewesss14"},
			{Kind: "website", Value: "https://miralabs.tech"},
			{Kind: "github_sponsors", Value: "wewesss"},
		},
	}
	return c
}

const customIconPath = "M12 2l3 7h7l-5.5 4.5L18.5 21 12 16.8 5.5 21l2-7.5L2 9h7z"

func AllLinks() []config.Link {
	return []config.Link{
		{Kind: "youtube", Value: "@ghstats-demo"},
		{Kind: "twitch", Value: "ghstats-demo"},
		{Kind: "kick", Value: "ghstats-demo"},
		{Kind: "discord", Value: "ghstats-demo"},
		{Kind: "tiktok", Value: "@ghstats-demo"},
		{Kind: "instagram", Value: "ghstats-demo"},
		{Kind: "x", Value: "ghstats-demo"},
		{Kind: "bluesky", Value: "ghstats-demo.bsky.social"},
		{Kind: "threads", Value: "ghstats-demo"},
		{Kind: "mastodon", Value: "https://example.com/@ghstats-demo"},
		{Kind: "reddit", Value: "ghstats-demo"},
		{Kind: "linkedin", Value: "ghstats-demo"},
		{Kind: "steam", Value: "https://example.com/steam/ghstats-demo"},
		{Kind: "itchio", Value: "ghstats-demo"},
		{Kind: "website", Value: "https://example.com"},
		{Kind: "github_sponsors", Value: "ghstats-demo"},
		{Kind: "open_collective", Value: "ghstats-demo"},
		{Kind: "buy_me_a_coffee", Value: "ghstats-demo"},
		{Kind: "ko_fi", Value: "ghstats-demo"},
		{Kind: "patreon", Value: "ghstats-demo"},
		CustomLinks()["path"],
	}
}

func CustomLinks() map[string]config.Link {
	return map[string]config.Link{
		"named": {Kind: config.KindCustom, Value: "https://example.com/blog", Label: "my blog", Icon: "youtube"},
		"path":  {Kind: config.KindCustom, Value: "https://example.com/blog", Label: "my blog", Icon: customIconPath},
		"none":  {Kind: config.KindCustom, Value: "https://example.com/blog", Label: "my blog"},
	}
}

func i64(v int64) *int64 { return &v }

func lang(name string, share float64) report.LangShare {
	return report.LangShare{Name: name, Share: share}
}

func weeks(vals ...int64) []report.WeekCount {
	start := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	out := make([]report.WeekCount, len(vals))
	for i, v := range vals {
		out[i] = report.WeekCount{Start: start.AddDate(0, 0, 7*i).Format("2006-01-02"), Count: v}
	}
	return out
}

func Report() report.Report {
	return report.Report{
		SchemaVersion: report.SchemaVersion,
		GeneratedAt:   time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC),
		CollectedAt:   time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC),
		Login:         "wewesss",
		Totals:        report.Totals{Repos: 42, PublicRepos: 31, PrivateRepos: 11, Stars: 128, Forks: 17, Followers: 56},
		Languages: report.Languages{
			Mode: "adjusted",
			Raw: []report.LangShare{
				lang("Lua", .317), lang("TypeScript", .246), lang("Go", .175), lang("Makefile", .087),
				lang("Rust", .079), lang("Python", .056), lang("Shell", .04),
			},
			Display: []report.LangShare{
				lang("TypeScript", .31), lang("Go", .22), lang("Lua", .14), lang("Rust", .10),
				lang("Python", .07), lang("Shell", .05), lang("Other", .11),
			},
			Multipliers: map[string]float64{"Lua": 0.35},
		},
		Distribution: []report.Metric{
			{Channel: report.ChannelReleaseAssets, Count: i64(3412), Window: report.WindowAllTime, Reliability: report.ReliabilityObserved},
			{Channel: report.ChannelClones, Count: i64(287), Window: report.WindowLast14Days, Reliability: report.ReliabilityObserved},
			{Channel: report.ChannelClones, Count: i64(900), Window: "since:2026-01-01", Reliability: report.ReliabilityObserved},
			{Channel: report.ChannelViews, Count: i64(1905), Window: report.WindowLast14Days, Reliability: report.ReliabilityObserved},
			{Channel: report.ChannelViews, Count: i64(5000), Window: "since:2026-01-01", Reliability: report.ReliabilityObserved},
			{Channel: report.ChannelSourceArchives, Window: report.WindowAllTime, Reliability: report.ReliabilityUnavailable},
		},
		Trends: []report.Trend{
			{Channel: report.ChannelClones, Weeks: weeks(8, 12, 9, 15, 11, 7, 14, 18, 13, 10, 21, 17, 12, 19, 25, 16, 14, 22, 28, 19, 17, 24, 31, 22, 27, 35)},
			{Channel: report.ChannelViews, Weeks: weeks(42, 38, 51, 47, 60, 55, 49, 72, 66, 58, 81, 77, 69, 90, 84, 79, 102, 95, 88, 110, 97, 121, 115, 108, 134, 128)},
		},
		Warnings: []string{},
	}
}
