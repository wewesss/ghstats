# FAQ: ghstats

> _Version française : [FAQ.fr.md](./FAQ.fr.md)_

Common questions about the design and the trade-offs. If something here is wrong or unclear, open an issue; I'd rather fix it than leave a stale answer.

---

## Setup and privacy

### Why does the raw data live in a private repository?

Snapshots and the traffic history contain private repository names. The rendered files (`stats.json`, the SVG cards, the link badges, `README.snippet.md`) never do, so only the output directory is committed to the public profile repository. The workflow in [Getting started](./docs/en/guide/getting-started.md) checks out `<user>/ghstats-data` into `data-dir` and pushes there. See [Privacy](./docs/en/guide/privacy.md).

### Why does a fine-grained token not give me the traffic of my organizations?

A fine-grained token covers a single resource owner. Repositories of an organization are outside that owner, so GitHub answers 403 on their traffic endpoints; `collect` prints a hint and carries on with the other numbers. Use one token per organization, or a classic token with the `repo` scope. Traffic also needs the read-only Administration permission, which is why the token section of [Getting started](./docs/en/guide/getting-started.md#1-create-the-token) lists it.

### What does the Action cost in API calls?

One paged GraphQL query lists the repositories with their languages and recent releases (50 repositories per page), and a repository with many releases gets extra pages. Traffic adds two REST calls per repository, one for views and one for clones. A run that exhausts the rate limit stops with an error instead of writing partial data. Set `traffic: false` on the Action to skip the traffic calls.

## Numbers

### Why is there no longer an `n/a` tile for source archives?

GitHub exposes no download counter for source archives, so the number can never be filled in. The report still lists the `source_archives` channel as unavailable, with a note saying why, but the totals card now shows cumulative views instead of a tile that could only ever say `n/a`.

### What is the difference between raw and adjusted languages?

`raw` shows the bytes of code GitHub reports for each language, as they are. `adjusted` multiplies each language by its weight in `languages.weights` and each repository by its weight in `scope.repo_weights` before the shares are computed, so a mostly generated or vendored language can count for less. `stats.json` keeps both sets of shares side by side. See [Configuration](./docs/en/guide/configuration.md#raw-and-adjusted).

### Why does TypeScript go up when I exclude HTML?

Shares always sum to 100 percent. `languages.exclude` removes a language from the adjusted calculation, as a weight of 0 would, so the total gets smaller and every other language takes a larger share of it. Nothing about TypeScript changed, only the denominator. `exclude` has no effect in `raw` mode, and `languages.hide` is different: it only removes a language from the display.

### Why are my traffic numbers higher than the last 14 days?

GitHub keeps 14 days of views and clones. ghstats merges the day counts of every run into `history.json`, so the history grows beyond that window as long as the workflow keeps running. A day that no run saw before it left the 14-day window is lost for good, so run it at least every 14 days; the daily cron in the examples leaves plenty of margin.

## Operation

### How many snapshots are kept, and does pruning lose history?

The `keep-snapshots` input defaults to `30`, and `0` keeps every snapshot. Pruning only removes old files in `snapshots/`; the traffic history lives in `history.json`, which is merged at each run and never pruned.

### Why did my scheduled workflow stop running?

GitHub disables scheduled workflows in a public repository after 60 days without repository activity. The workflow commits cards only when they change, so a quiet period can leave the repository without a recent commit. If the runs stop, open the Actions tab and re-enable the workflow, and run it once with Run workflow.

## Cards

### What does the `cycle` accent do, and what about reduced motion?

`cycle` makes the accent change over time, kept in sync across all the cards. When the viewer asks for reduced motion, or when you set `render.motion: false`, every animation stops and the card keeps its first accent color. The badges are static in every case.

### Why does my profile README show an old card after the workflow ran?

GitHub proxies and caches the images it shows in a README, so a new card can take a few minutes to appear, and a browser may keep the old one. Reload the page without the cache. If it still shows the old card, check that the workflow committed the new files in the output directory.

### Can I build my own front or server?

Yes. `stats.json` is public and readable from any web page, the Go packages and the WASM renderer are documented, and a small server example is included. See [Build your own front](./docs/en/guide/extend.md).
