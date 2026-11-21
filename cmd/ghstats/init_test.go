package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/render"
)

const initToken = "ghp_s3cretTokenValue"

type fakeFile struct {
	content string
	sha     string
}

type fakeCall struct {
	args  []string
	stdin string
}

type fakeGH struct {
	login   string
	scopes  string
	missing bool
	repos   map[string]string
	files   map[string]fakeFile
	secrets map[string]string
	runs    []string
	calls   []fakeCall
	puts    map[string][]map[string]string
	shaSeq  int
}

func newFakeGH(login string) *fakeGH {
	return &fakeGH{
		login:   login,
		scopes:  "gist, read:org, repo, workflow",
		repos:   map[string]string{},
		files:   map[string]fakeFile{},
		secrets: map[string]string{},
		puts:    map[string][]map[string]string{},
	}
}

func (g *fakeGH) seedRepo(name, visibility, readme string) {
	g.repos[name] = visibility
	if readme != "" {
		g.shaSeq++
		g.files[name+"/README.md"] = fakeFile{content: readme, sha: fmt.Sprintf("sha%d", g.shaSeq)}
	}
}

func ghFail(cmd, stderr string) error {
	return &ghError{Cmd: cmd, Stderr: stderr, Err: errors.New("exit status 1")}
}

func (g *fakeGH) exec(_ context.Context, in []byte, args ...string) ([]byte, error) {
	g.calls = append(g.calls, fakeCall{args: args, stdin: string(in)})
	if g.missing {
		return nil, osexec.ErrNotFound
	}
	switch {
	case len(args) == 3 && args[0] == "api" && args[1] == "-i" && args[2] == "user":
		return []byte("HTTP/2.0 200 OK\r\nContent-Type: application/json\r\nX-Oauth-Scopes: " + g.scopes + "\r\n\r\n{\"login\":\"" + g.login + "\"}"), nil
	case len(args) == 4 && args[0] == "api" && args[1] == "user" && args[2] == "--jq":
		return []byte(g.login + "\n"), nil
	case len(args) == 6 && args[0] == "api" && args[1] == "-X" && args[2] == "PUT":
		return nil, g.put(strings.TrimPrefix(args[3], "repos/"), in)
	case len(args) == 2 && args[0] == "api":
		return g.get(strings.TrimPrefix(args[1], "repos/"))
	case len(args) == 5 && args[0] == "repo" && args[1] == "view":
		vis, ok := g.repos[args[2]]
		if !ok {
			return nil, ghFail("repo view", "GraphQL: Could not resolve to a Repository with the name '"+args[2]+"'. (repository)")
		}
		return []byte(`{"visibility":"` + vis + `"}`), nil
	case len(args) == 5 && args[0] == "repo" && args[1] == "create":
		g.repos[args[2]] = map[string]string{"--public": "PUBLIC", "--private": "PRIVATE"}[args[3]]
		g.shaSeq++
		g.files[args[2]+"/README.md"] = fakeFile{content: "# " + args[2] + "\n", sha: fmt.Sprintf("sha%d", g.shaSeq)}
		return nil, nil
	case len(args) == 6 && args[0] == "secret" && args[1] == "list":
		var names []map[string]string
		for k := range g.secrets {
			if repo, name, _ := strings.Cut(k, "|"); repo == args[3] {
				names = append(names, map[string]string{"name": name})
			}
		}
		out, _ := json.Marshal(names)
		return out, nil
	case len(args) == 5 && args[0] == "secret" && args[1] == "set":
		g.secrets[args[4]+"|"+args[2]] = string(in)
		return nil, nil
	case len(args) == 5 && args[0] == "workflow" && args[1] == "run":
		g.runs = append(g.runs, args[4]+"/"+args[2])
		return nil, nil
	}
	return nil, fmt.Errorf("fake gh: unexpected call %v", args)
}

func (g *fakeGH) get(path string) ([]byte, error) {
	f, ok := g.files[strings.Replace(path, "/contents/", "/", 1)]
	if !ok {
		return nil, ghFail("api", "gh: Not Found (HTTP 404)")
	}
	out, _ := json.Marshal(map[string]string{"sha": f.sha, "content": base64.StdEncoding.EncodeToString([]byte(f.content))})
	return out, nil
}

func (g *fakeGH) put(path string, in []byte) error {
	key := strings.Replace(path, "/contents/", "/", 1)
	var body map[string]string
	if err := json.Unmarshal(in, &body); err != nil {
		return err
	}
	raw, err := base64.StdEncoding.DecodeString(body["content"])
	if err != nil {
		return err
	}
	if old, ok := g.files[key]; ok && body["sha"] != old.sha {
		return ghFail("api", "gh: sha does not match (HTTP 409)")
	}
	g.shaSeq++
	g.files[key] = fakeFile{content: string(raw), sha: fmt.Sprintf("sha%d", g.shaSeq)}
	g.puts[key] = append(g.puts[key], body)
	return nil
}

func (g *fakeGH) writes() []string {
	var out []string
	for _, c := range g.calls {
		joined := strings.Join(c.args, " ")
		if strings.HasPrefix(joined, "repo create") || strings.HasPrefix(joined, "api -X PUT") || strings.HasPrefix(joined, "secret set") {
			out = append(out, joined)
		}
	}
	return out
}

func initWith(t *testing.T, g *fakeGH, in string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	oldExec, oldIn := ghExec, stdin
	ghExec = g.exec
	stdin = strings.NewReader(in)
	t.Cleanup(func() { ghExec, stdin = oldExec, oldIn })
	return exec(append([]string{"init"}, args...)...)
}

func setupInit(t *testing.T) {
	t.Helper()
	setup(t)
	t.Setenv("GHSTATS_TOKEN", initToken)
}

func TestInitFreshProfile(t *testing.T) {
	setupInit(t)
	g := newFakeGH("me")
	code, stdout, stderr := initWith(t, g, "", "--yes")
	if code != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if g.repos["me/me"] != "PUBLIC" || g.repos["me/ghstats-data"] != "PRIVATE" {
		t.Fatalf("repos = %v", g.repos)
	}
	wf := g.files["me/me/.github/workflows/ghstats.yml"].content
	if !strings.Contains(wf, "repository: me/ghstats-data") || strings.Contains(wf, "<user>") {
		t.Fatalf("workflow = %q", wf)
	}
	if _, err := config.Parse(strings.NewReader(g.files["me/me/ghstats.yaml"].content)); err != nil {
		t.Fatal(err)
	}
	readme := g.files["me/me/README.md"].content
	if !strings.Contains(readme, readmeStart) || !strings.Contains(readme, readmeEnd) || !strings.Contains(readme, strings.TrimRight(render.Snippet(config.Default()), "\n")) {
		t.Fatalf("README = %q", readme)
	}
	if got := g.secrets["me/me|GHSTATS_TOKEN"]; got != initToken {
		t.Fatalf("secret = %q", got)
	}
	for _, c := range g.calls {
		if strings.Contains(strings.Join(c.args, " "), initToken) {
			t.Fatalf("token passed in arguments: %v", c.args)
		}
	}
	if len(g.runs) != 1 || g.runs[0] != "me/me/ghstats.yml" {
		t.Fatalf("runs = %v", g.runs)
	}
	if !strings.Contains(stdout, "https://github.com/me/me/actions") {
		t.Fatalf("stdout = %q", stdout)
	}
	if strings.Contains(stdout+stderr, initToken) {
		t.Fatal("token leaked in the output")
	}
}

func TestInitSecondRunWritesNothing(t *testing.T) {
	setupInit(t)
	g := newFakeGH("me")
	if code, _, stderr := initWith(t, g, "", "--user", "me", "--yes"); code != 0 {
		t.Fatalf("first run = %d, %q", code, stderr)
	}
	t.Setenv("GHSTATS_TOKEN", "")
	g.calls = nil
	code, stdout, stderr := initWith(t, g, "", "--user", "me")
	if code != 0 {
		t.Fatalf("second run = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if w := g.writes(); len(w) != 0 {
		t.Fatalf("second run wrote: %v", w)
	}
	for _, line := range strings.Split(stdout, "\n") {
		if (strings.HasPrefix(line, "created") || strings.HasPrefix(line, "updated")) && !strings.Contains(line, "first run") {
			t.Fatalf("unexpected change line %q in %q", line, stdout)
		}
	}
}

func TestInitForceRewritesWithSha(t *testing.T) {
	setupInit(t)
	g := newFakeGH("me")
	if code, _, stderr := initWith(t, g, "", "--user", "me", "--yes", "--no-run"); code != 0 {
		t.Fatalf("first run = %d, %q", code, stderr)
	}
	if len(g.runs) != 0 {
		t.Fatalf("--no-run started a run: %v", g.runs)
	}
	before := map[string]string{}
	for _, p := range []string{".github/workflows/ghstats.yml", "ghstats.yaml"} {
		before[p] = g.files["me/me/"+p].sha
	}
	g.puts = map[string][]map[string]string{}
	if code, _, stderr := initWith(t, g, "", "--user", "me", "--yes", "--no-run", "--force"); code != 0 {
		t.Fatalf("forced run = %d, %q", code, stderr)
	}
	for p, sha := range before {
		puts := g.puts["me/me/"+p]
		if len(puts) != 1 || puts[0]["sha"] != sha {
			t.Fatalf("%s puts = %v, want one with sha %s", p, puts, sha)
		}
	}
}

func TestInitReadmeKeepsTextAroundMarkers(t *testing.T) {
	setupInit(t)
	g := newFakeGH("me")
	g.seedRepo("me/me", "PUBLIC", "intro\n\n"+readmeStart+"\nold\n"+readmeEnd+"\n\noutro\n")
	if code, _, stderr := initWith(t, g, "", "--user", "me", "--yes", "--no-run"); code != 0 {
		t.Fatalf("code = %d, %q", code, stderr)
	}
	readme := g.files["me/me/README.md"].content
	want := "intro\n\n" + readmeBlock(render.Snippet(config.Default())) + "\n\noutro\n"
	if readme != want {
		t.Fatalf("README = %q, want %q", readme, want)
	}
}

func TestInitReadmeAppendsAfterBlankLine(t *testing.T) {
	setupInit(t)
	g := newFakeGH("me")
	g.seedRepo("me/me", "PUBLIC", "# Hi\n")
	if code, _, stderr := initWith(t, g, "", "--user", "me", "--yes", "--no-run"); code != 0 {
		t.Fatalf("code = %d, %q", code, stderr)
	}
	want := "# Hi\n\n" + readmeBlock(render.Snippet(config.Default())) + "\n"
	if got := g.files["me/me/README.md"].content; got != want {
		t.Fatalf("README = %q, want %q", got, want)
	}
}

func TestInitReadmeDeclined(t *testing.T) {
	setupInit(t)
	g := newFakeGH("me")
	g.seedRepo("me/me", "PUBLIC", "hello\n")
	code, stdout, stderr := initWith(t, g, "n\n", "--user", "me", "--no-run")
	if code != 0 {
		t.Fatalf("code = %d, %q", code, stderr)
	}
	if got := g.files["me/me/README.md"].content; got != "hello\n" {
		t.Fatalf("README was written: %q", got)
	}
	if len(g.puts["me/me/README.md"]) != 0 {
		t.Fatal("README was put")
	}
	if !strings.Contains(stdout, "Write it to README.md? [y/N]") || !strings.Contains(stdout, "skipped") || !strings.Contains(stdout, "<picture>") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestInitTokenFromStdin(t *testing.T) {
	setup(t)
	t.Setenv("GHSTATS_TOKEN", "")
	g := newFakeGH("me")
	code, stdout, stderr := initWith(t, g, initToken+"\n", "--user", "me", "--yes", "--no-run")
	if code != 0 {
		t.Fatalf("code = %d, %q", code, stderr)
	}
	if got := g.secrets["me/me|GHSTATS_TOKEN"]; got != initToken {
		t.Fatalf("secret = %q", got)
	}
	if strings.Contains(stdout+stderr, initToken) {
		t.Fatal("token leaked in the output")
	}
}

func TestInitNoTokenFails(t *testing.T) {
	setup(t)
	t.Setenv("GHSTATS_TOKEN", "")
	g := newFakeGH("me")
	code, _, stderr := initWith(t, g, "", "--user", "me", "--yes", "--no-run")
	if code != 1 || !strings.Contains(stderr, "no token") {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
}

func TestInitMissingWorkflowScope(t *testing.T) {
	setupInit(t)
	g := newFakeGH("me")
	g.scopes = "gist, read:org, repo"
	code, _, stderr := initWith(t, g, "", "--yes")
	if code != 1 || !strings.Contains(stderr, "gh auth refresh -h github.com -s workflow") {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	if w := g.writes(); len(w) != 0 {
		t.Fatalf("wrote: %v", w)
	}
}

func TestInitPublicDataRepoStops(t *testing.T) {
	setupInit(t)
	for _, extra := range [][]string{nil, {"--force"}} {
		g := newFakeGH("me")
		g.seedRepo("me/me", "PUBLIC", "hello\n")
		g.seedRepo("me/ghstats-data", "PUBLIC", "")
		code, _, stderr := initWith(t, g, "", append([]string{"--user", "me", "--yes"}, extra...)...)
		if code != 1 || !strings.Contains(stderr, "public") {
			t.Fatalf("code = %d, stderr = %q", code, stderr)
		}
		if w := g.writes(); len(w) != 0 {
			t.Fatalf("wrote: %v", w)
		}
	}
}

func TestInitTokenOfAnotherLogin(t *testing.T) {
	setupInit(t)
	g := newFakeGH("other")
	g.seedRepo("other/other", "PUBLIC", "hello\n")
	g.seedRepo("other/ghstats-data", "PRIVATE", "")
	code, stdout, stderr := initWith(t, g, "", "--yes", "--no-run")
	if code != 1 || !strings.Contains(stderr, "token") {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	if strings.Contains(stdout+stderr, initToken) {
		t.Fatal("token leaked in the output")
	}
	if len(g.secrets) != 0 {
		t.Fatalf("secrets = %v", g.secrets)
	}
}

func TestInitGhMissing(t *testing.T) {
	setupInit(t)
	g := newFakeGH("me")
	g.missing = true
	code, _, stderr := initWith(t, g, "", "--user", "me")
	if code != 1 || !strings.Contains(stderr, "https://cli.github.com") {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
}

func TestInitProfileCreationRefused(t *testing.T) {
	setupInit(t)
	g := newFakeGH("me")
	code, _, stderr := initWith(t, g, "n\n", "--user", "me")
	if code != 1 || !strings.Contains(stderr, "me/me") {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	if w := g.writes(); len(w) != 0 {
		t.Fatalf("wrote: %v", w)
	}
}

func docYAMLBlock(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "docs", "en", "guide", "getting-started.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, rest, ok := strings.Cut(strings.ReplaceAll(string(b), "\r\n", "\n"), "```yaml\n")
	if !ok {
		t.Fatal("no yaml block in getting-started.md")
	}
	block, _, ok := strings.Cut(rest, "```")
	if !ok {
		t.Fatal("unterminated yaml block")
	}
	return block
}

func TestInitWorkflowTemplateMatchesDocs(t *testing.T) {
	want := docYAMLBlock(t)
	if got := strings.ReplaceAll(workflowTemplate, "\r\n", "\n"); got != want {
		t.Fatalf("cmd/ghstats/workflow.yml differs from the workflow of docs/en/guide/getting-started.md\n--- template\n%s\n--- docs\n%s", got, want)
	}
}
