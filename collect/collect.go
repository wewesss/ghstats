// Package collect gathers repositories, languages, releases and traffic into a snapshot.
package collect

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v93/github"

	"github.com/wewesss/ghstats/internal/ghapi"
	"github.com/wewesss/ghstats/snapshot"
)

const (
	defaultGraphQLURL  = "https://api.github.com/graphql"
	defaultHTTPTimeout = 30 * time.Second
)

type Options struct {
	Token        string
	GraphQLURL   string
	RESTURL      string
	Affiliations []string
	Traffic      bool
	Concurrency  int
	HTTPClient   *http.Client
	Now          func() time.Time
}

type collector struct {
	gql         *ghapi.Client
	gh          *github.Client
	affiliation []string
	concurrency int
}

func Collect(ctx context.Context, opts Options) (snapshot.Snapshot, error) {
	if opts.GraphQLURL == "" {
		opts.GraphQLURL = defaultGraphQLURL
	}
	if len(opts.Affiliations) == 0 {
		opts.Affiliations = []string{"OWNER", "ORGANIZATION_MEMBER", "COLLABORATOR"}
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 4
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}

	c := &collector{
		gql:         &ghapi.Client{HTTP: opts.HTTPClient, Endpoint: opts.GraphQLURL, Token: opts.Token},
		affiliation: opts.Affiliations,
		concurrency: opts.Concurrency,
	}
	if opts.Traffic {
		ghOpts := []github.ClientOptionsFunc{github.WithHTTPClient(opts.HTTPClient)}
		if opts.Token != "" {
			ghOpts = append(ghOpts, github.WithAuthToken(opts.Token))
		}
		if opts.RESTURL != "" {
			ghOpts = append(ghOpts, github.WithURLs(&opts.RESTURL, nil))
		}
		gh, err := github.NewClient(ghOpts...)
		if err != nil {
			return snapshot.Snapshot{}, fmt.Errorf("collect: github client: %w", err)
		}
		c.gh = gh
	}

	snap := snapshot.Snapshot{
		SchemaVersion: snapshot.SchemaVersion,
		CollectedAt:   opts.Now().UTC(),
		Errors:        []snapshot.RepoError{},
	}

	repos, pending, err := c.fetchRepos(ctx, &snap.Viewer)
	if err != nil {
		return snapshot.Snapshot{}, err
	}

	releaseErrs, err := c.fetchReleases(ctx, repos, pending)
	if err != nil {
		return snapshot.Snapshot{}, err
	}
	snap.Errors = appendErrors(snap.Errors, releaseErrs)

	if opts.Traffic {
		trafficErrs, err := c.fetchTraffic(ctx, repos)
		if err != nil {
			return snapshot.Snapshot{}, err
		}
		snap.Errors = appendErrors(snap.Errors, trafficErrs)
	}

	snap.Repos = repos
	return snap, nil
}

func appendErrors(dst []snapshot.RepoError, src []*snapshot.RepoError) []snapshot.RepoError {
	for _, e := range src {
		if e != nil {
			dst = append(dst, *e)
		}
	}
	return dst
}

type pendingReleases struct {
	repo   int
	cursor string
}

func (c *collector) fetchRepos(ctx context.Context, viewer *snapshot.Viewer) ([]snapshot.Repo, []pendingReleases, error) {
	repos := []snapshot.Repo{}
	var pending []pendingReleases
	seen := map[string]bool{}
	cursor := ""
	for {
		vars := map[string]any{"aff": c.affiliation, "cursor": nil}
		if cursor != "" {
			vars["cursor"] = cursor
		}
		var page reposResponse
		if err := c.gql.Do(ctx, reposQuery, vars, &page); err != nil {
			return nil, nil, fmt.Errorf("collect: repositories: %w", sanitize(err))
		}
		*viewer = snapshot.Viewer{
			Login:     page.Viewer.Login,
			Followers: page.Viewer.Followers.TotalCount,
			Following: page.Viewer.Following.TotalCount,
		}
		for _, n := range page.Viewer.Repositories.Nodes {
			if seen[n.ID] {
				continue
			}
			seen[n.ID] = true
			repos = append(repos, toRepo(n))
			if n.Releases.PageInfo.HasNextPage {
				pending = append(pending, pendingReleases{repo: len(repos) - 1, cursor: n.Releases.PageInfo.EndCursor})
			}
		}

		info := page.Viewer.Repositories.PageInfo
		if !info.HasNextPage || info.EndCursor == "" || info.EndCursor == cursor {
			break
		}
		if rl := page.RateLimit; rl != nil && rl.Remaining == 0 {
			return nil, nil, fmt.Errorf("collect: rate limit exhausted: %w",
				&ghapi.RateLimitError{ResetAt: rl.ResetAt, RetryAfter: max(time.Until(rl.ResetAt), 0)})
		}
		cursor = info.EndCursor
	}

	order := make([]int, len(repos))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return strings.Compare(repos[a].NameWithOwner, repos[b].NameWithOwner)
	})
	rank := make([]int, len(repos))
	sorted := make([]snapshot.Repo, len(repos))
	for pos, i := range order {
		rank[i] = pos
		sorted[pos] = repos[i]
	}
	for i := range pending {
		pending[i].repo = rank[pending[i].repo]
	}
	return sorted, pending, nil
}

func toRepo(n apiRepo) snapshot.Repo {
	r := snapshot.Repo{
		ID:            n.ID,
		NameWithOwner: n.NameWithOwner,
		Owner:         n.Owner.Login,
		Private:       n.IsPrivate,
		Fork:          n.IsFork,
		Archived:      n.IsArchived,
		Stars:         n.StargazerCount,
		Forks:         n.ForkCount,
		Languages:     make([]snapshot.LanguageSize, 0, len(n.Languages.Edges)),
		Releases:      appendReleases(nil, n.Releases.Nodes),
	}
	for _, e := range n.Languages.Edges {
		r.Languages = append(r.Languages, snapshot.LanguageSize{Name: e.Node.Name, Bytes: e.Size})
	}
	return r
}

func appendReleases(dst []snapshot.Release, nodes []apiRelease) []snapshot.Release {
	if dst == nil {
		dst = make([]snapshot.Release, 0, len(nodes))
	}
	for _, n := range nodes {
		rel := snapshot.Release{Tag: n.TagName, Assets: make([]snapshot.Asset, 0, len(n.ReleaseAssets.Nodes))}
		if n.PublishedAt != nil {
			rel.PublishedAt = n.PublishedAt.UTC()
		}
		for _, a := range n.ReleaseAssets.Nodes {
			rel.Assets = append(rel.Assets, snapshot.Asset{Name: a.Name, DownloadCount: a.DownloadCount})
		}
		dst = append(dst, rel)
	}
	return dst
}

func (c *collector) fetchReleases(ctx context.Context, repos []snapshot.Repo, pending []pendingReleases) ([]*snapshot.RepoError, error) {
	errs := make([]*snapshot.RepoError, len(pending))
	err := forEach(ctx, len(pending), c.concurrency, func(ctx context.Context, k int) error {
		p := pending[k]
		repo := &repos[p.repo]
		cursor := p.cursor
		for {
			var out releasesResponse
			vars := map[string]any{"id": repo.ID, "cursor": cursor}
			if err := c.gql.Do(ctx, releasesQuery, vars, &out); err != nil {
				err = sanitize(err)
				if isFatal(ctx, err) {
					return err
				}
				errs[k] = &snapshot.RepoError{Repo: repo.NameWithOwner, Stage: "releases", Message: err.Error()}
				return nil
			}
			if out.Node == nil {
				return nil
			}
			repo.Releases = appendReleases(repo.Releases, out.Node.Releases.Nodes)
			info := out.Node.Releases.PageInfo
			if !info.HasNextPage || info.EndCursor == "" || info.EndCursor == cursor {
				return nil
			}
			cursor = info.EndCursor
		}
	})
	if err != nil {
		return nil, fmt.Errorf("collect: releases: %w", err)
	}
	return errs, nil
}

func (c *collector) fetchTraffic(ctx context.Context, repos []snapshot.Repo) ([]*snapshot.RepoError, error) {
	errs := make([]*snapshot.RepoError, len(repos))
	err := forEach(ctx, len(repos), c.concurrency, func(ctx context.Context, i int) error {
		repo := &repos[i]
		_, name, _ := strings.Cut(repo.NameWithOwner, "/")
		breakdown := &github.TrafficBreakdownOptions{Per: "day"}
		views, _, err := c.gh.Repositories.ListTrafficViews(ctx, repo.Owner, name, breakdown)
		var clones *github.TrafficClones
		if err == nil {
			clones, _, err = c.gh.Repositories.ListTrafficClones(ctx, repo.Owner, name, breakdown)
		}
		if err != nil {
			err = sanitize(err)
			if isFatal(ctx, err) {
				return err
			}
			errs[i] = &snapshot.RepoError{Repo: repo.NameWithOwner, Stage: "traffic", Message: err.Error()}
			return nil
		}
		repo.Traffic = &snapshot.Traffic{Views: dayCounts(views.Views), Clones: dayCounts(clones.Clones)}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("collect: traffic: %w", err)
	}
	return errs, nil
}

func dayCounts(data []*github.TrafficData) []snapshot.DayCount {
	out := make([]snapshot.DayCount, 0, len(data))
	for _, d := range data {
		if d == nil || d.Timestamp == nil {
			continue
		}
		t := d.Timestamp.UTC()
		dc := snapshot.DayCount{Day: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)}
		if d.Count != nil {
			dc.Count = *d.Count
		}
		if d.Uniques != nil {
			dc.Uniques = *d.Uniques
		}
		out = append(out, dc)
	}
	return out
}

func isFatal(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return true
	}
	var rl *ghapi.RateLimitError
	var ghRL *github.RateLimitError
	var ghAbuse *github.AbuseRateLimitError
	return errors.As(err, &rl) || errors.As(err, &ghRL) || errors.As(err, &ghAbuse)
}

func forEach(parent context.Context, n, limit int, fn func(ctx context.Context, i int) error) error {
	ctx, cancel := context.WithCancelCause(parent)
	defer cancel(nil)
	var (
		first error
		once  sync.Once
		wg    sync.WaitGroup
	)
	sem := make(chan struct{}, limit)
loop:
	for i := range n {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break loop
		}
		if ctx.Err() != nil {
			<-sem
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fn(ctx, i); err != nil {
				once.Do(func() {
					first = err
					cancel(err)
				})
			}
		}()
	}
	wg.Wait()
	if err := parent.Err(); err != nil {
		return err
	}
	return first
}
