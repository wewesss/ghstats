// Command ghstats collects GitHub statistics, computes a report and renders it as JSON and SVG.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wewesss/ghstats/collect"
	"github.com/wewesss/ghstats/compute"
	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/history"
	"github.com/wewesss/ghstats/render"
	"github.com/wewesss/ghstats/report"
	"github.com/wewesss/ghstats/snapshot"
)

var (
	version = "dev"
	clock   = time.Now
)

const (
	exitOK      = 0
	exitRuntime = 1
	exitUsage   = 2

	snapshotNameLayout = "2006-01-02T150405Z"
)

const usageText = `Usage: ghstats <command> [flags]

Commands:
  collect   fetch repositories, languages, releases and traffic into a snapshot
  compute   turn a snapshot and a history into report.json
  render    write stats.json, the SVG cards and README.snippet.md from a report
  version   print the version

Run "ghstats <command> -h" for the flags of a command.
`

type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usageText)
		return exitUsage
	}
	cmd, rest := args[0], args[1:]
	var err error
	switch cmd {
	case "collect":
		err = runCollect(rest, stdout, stderr)
	case "compute":
		err = runCompute(rest, stderr)
	case "render":
		err = runRender(rest, stderr)
	case "version":
		fmt.Fprintln(stdout, version)
		return exitOK
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usageText)
		return exitOK
	default:
		fmt.Fprintf(stderr, "ghstats: unknown command %q\n\n%s", cmd, usageText)
		return exitUsage
	}

	var ue usageError
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		return exitOK
	case errors.As(err, &ue):
		fmt.Fprintf(stderr, "ghstats %s: %s\n", cmd, ue.msg)
		return exitUsage
	default:
		fmt.Fprintf(stderr, "ghstats %s: %v\n", cmd, err)
		return exitRuntime
	}
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet("ghstats "+name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func parseFlags(fs *flag.FlagSet, args []string, required ...string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return usageError{"invalid flags"}
	}
	if fs.NArg() > 0 {
		return usageError{fmt.Sprintf("unexpected argument %q", fs.Arg(0))}
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	var missing []string
	for _, name := range required {
		if !set[name] || fs.Lookup(name).Value.String() == "" {
			missing = append(missing, "--"+name)
		}
	}
	if len(missing) > 0 {
		fs.Usage()
		return usageError{"missing required flag " + strings.Join(missing, ", ")}
	}
	return nil
}

func runCollect(args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("collect", stderr)
	out := fs.String("out", "data/snapshots", "directory receiving the snapshot file")
	histPath := fs.String("history", "data/history.json", "traffic history file, updated in place")
	noTraffic := fs.Bool("no-traffic", false, "skip the traffic API calls")
	concurrency := fs.Int("concurrency", 0, "parallel API requests (0 uses the default)")
	keep := fs.Int("keep", 0, "snapshots kept in --out (0 keeps them all)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if *concurrency < 0 {
		return usageError{"--concurrency must be >= 0"}
	}
	if *keep < 0 {
		return usageError{"--keep must be >= 0"}
	}

	token := os.Getenv("GHSTATS_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return errors.New("no token: set GHSTATS_TOKEN or GITHUB_TOKEN")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	snap, err := collect.Collect(ctx, collect.Options{
		Token:       token,
		GraphQLURL:  os.Getenv("GHSTATS_GRAPHQL_URL"),
		RESTURL:     os.Getenv("GHSTATS_REST_URL"),
		Traffic:     !*noTraffic,
		Concurrency: *concurrency,
		Now:         clock,
	})
	if err != nil {
		return err
	}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", *out, err)
	}
	snapPath := filepath.Join(*out, "snapshot-"+snap.CollectedAt.UTC().Format(snapshotNameLayout)+".json")
	if err := snapshot.WriteFile(snapPath, snap); err != nil {
		return err
	}
	if *keep > 0 {
		if err := pruneSnapshots(*out, snapPath, *keep); err != nil {
			return err
		}
	}

	h, err := readHistory(*histPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*histPath), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(*histPath), err)
	}
	if err := history.WriteFile(*histPath, history.Merge(h, snap)); err != nil {
		return err
	}
	if trafficForbidden(snap) {
		fmt.Fprintln(stderr, trafficHint)
	}
	fmt.Fprintln(stdout, snapPath)
	return nil
}

func pruneSnapshots(dir, current string, keep int) error {
	names, err := filepath.Glob(filepath.Join(dir, "snapshot-*.json"))
	if err != nil {
		return err
	}
	sort.Strings(names)
	others := names[:0]
	for _, n := range names {
		if n != current {
			others = append(others, n)
		}
	}
	for _, n := range others[:max(0, len(others)-(keep-1))] {
		if err := os.Remove(n); err != nil {
			return fmt.Errorf("remove %s: %w", n, err)
		}
	}
	return nil
}

const trafficHint = "hint: traffic needs push access with a token that covers each owner; a fine-grained token covers a single owner (use one token per organization or a classic token with the repo scope)"

func trafficForbidden(s snapshot.Snapshot) bool {
	for _, e := range s.Errors {
		if e.Stage == "traffic" && strings.Contains(e.Message, "403") {
			return true
		}
	}
	return false
}

func readHistory(path string) (history.History, error) {
	h, err := history.ReadFile(path)
	if errors.Is(err, iofs.ErrNotExist) {
		return history.History{Repos: map[string]history.RepoHistory{}}, nil
	}
	return h, err
}

func runCompute(args []string, stderr io.Writer) error {
	fs := newFlagSet("compute", stderr)
	snapPath := fs.String("snapshot", "", "snapshot file")
	histPath := fs.String("history", "", "traffic history file (a missing file counts as empty)")
	cfgPath := fs.String("config", "", "configuration file (defaults when omitted)")
	out := fs.String("out", "", "report file to write")
	if err := parseFlags(fs, args, "snapshot", "history", "out"); err != nil {
		return err
	}

	snap, err := snapshot.ReadFile(*snapPath)
	if err != nil {
		return err
	}
	h, err := readHistory(*histPath)
	if err != nil {
		return err
	}
	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		return err
	}
	r, err := compute.Compute(snap, h, cfg, clock().UTC())
	if err != nil {
		return err
	}
	for _, w := range r.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}
	var buf bytes.Buffer
	if err := render.JSON(&buf, r); err != nil {
		return err
	}
	return writeFile(*out, buf.Bytes())
}

func runRender(args []string, stderr io.Writer) error {
	fs := newFlagSet("render", stderr)
	reportPath := fs.String("report", "", "report file")
	cfgPath := fs.String("config", "", "configuration file (defaults when omitted)")
	outDir := fs.String("out-dir", "", "directory receiving stats.json and the SVG cards")
	if err := parseFlags(fs, args, "report", "out-dir"); err != nil {
		return err
	}

	f, err := os.Open(*reportPath)
	if err != nil {
		return fmt.Errorf("open report: %w", err)
	}
	defer f.Close()
	var r report.Report
	if err := json.NewDecoder(f).Decode(&r); err != nil {
		return fmt.Errorf("decode report: %w", err)
	}
	if r.SchemaVersion != report.SchemaVersion {
		return fmt.Errorf("unsupported report schema_version %d (want %d)", r.SchemaVersion, report.SchemaVersion)
	}
	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		return err
	}

	var jsonBuf bytes.Buffer
	if err := render.JSON(&jsonBuf, r); err != nil {
		return err
	}
	files := map[string][]byte{"stats.json": jsonBuf.Bytes()}
	for _, mode := range []render.Mode{render.Dark, render.Light} {
		var buf bytes.Buffer
		if err := render.SVG(&buf, r, cfg.Render, mode); err != nil {
			return err
		}
		files["stats-"+string(mode)+".svg"] = buf.Bytes()
		if cfg.Profile.HasIdentity() {
			var ib bytes.Buffer
			if err := render.IdentitySVG(&ib, r, cfg, mode); err != nil {
				return err
			}
			files["identity-"+string(mode)+".svg"] = ib.Bytes()
		}
		for _, l := range cfg.Profile.ResolvedLinks() {
			var lb bytes.Buffer
			if err := render.LinkBadgeSVG(&lb, l, cfg.Render.Theme, mode); err != nil {
				return err
			}
			files["links/"+l.Kind+"-"+string(mode)+".svg"] = lb.Bytes()
		}
	}
	files["README.snippet.md"] = []byte(render.Snippet(cfg))
	for name, data := range files {
		if err := writeFile(filepath.Join(*outDir, filepath.FromSlash(name)), data); err != nil {
			return err
		}
	}
	return nil
}

func loadConfig(path string) (config.Config, error) {
	if path == "" {
		return config.Default(), nil
	}
	return config.Load(path)
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
