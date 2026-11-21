package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	osexec "os/exec"
	"os/signal"
	"slices"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/internal/ghapi"
	"github.com/wewesss/ghstats/render"
)

const (
	ghInstallURL        = "https://cli.github.com"
	tokenDocsURL        = "https://wewesss.github.io/ghstats/en/guide/getting-started/#1-create-the-token"
	defaultGraphQLURL   = "https://api.github.com/graphql"
	defaultRESTURL      = "https://api.github.com"
	readmeStart         = "<!-- ghstats:start -->"
	readmeEnd           = "<!-- ghstats:end -->"
	workflowPath        = ".github/workflows/ghstats.yml"
	configPath          = "ghstats.yaml"
	readmePath          = "README.md"
	secretName          = "GHSTATS_TOKEN"
	initHTTPTimeout     = 30 * time.Second
	profileConfigSample = `# ghstats configuration. Every key is optional; the defaults apply when a key is missing.
# Try themes, blocks and cards live, then paste the result here: https://wewesss.github.io/ghstats/en/builder/
# All keys: https://wewesss.github.io/ghstats/en/guide/configuration/
version: 1
# render:
#   blocks:
#     - stats: [totals]
#     - languages: [bars]
#     - history: [views]
#   theme:
#     preset: noir
#     accent: ice
`
)

//go:embed workflow.yml
var workflowTemplate string

var stdin io.Reader = os.Stdin

var ghExec = func(ctx context.Context, in []byte, args ...string) ([]byte, error) {
	path, err := osexec.LookPath("gh")
	if err != nil {
		return nil, err
	}
	cmd := osexec.CommandContext(ctx, path, args...)
	if in != nil {
		cmd.Stdin = bytes.NewReader(in)
	}
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return out.Bytes(), &ghError{Cmd: strings.Join(args[:min(len(args), 2)], " "), Stderr: strings.TrimSpace(errb.String()), Err: err}
	}
	return out.Bytes(), nil
}

type ghError struct {
	Cmd    string
	Stderr string
	Err    error
}

func (e *ghError) Error() string {
	msg := e.Stderr
	if msg == "" {
		msg = e.Err.Error()
	}
	return "gh " + e.Cmd + ": " + msg
}

func (e *ghError) Unwrap() error { return e.Err }

func isNotFound(err error) bool {
	var ge *ghError
	if !errors.As(err, &ge) {
		return false
	}
	s := strings.ToLower(ge.Stderr)
	return strings.Contains(s, "404") || strings.Contains(s, "not found") || strings.Contains(s, "could not resolve to a repository")
}

type initRun struct {
	ctx   context.Context
	raw   io.Reader
	in    *bufio.Reader
	out   io.Writer
	user  string
	yes   bool
	force bool
}

func runInit(args []string, in io.Reader, stdout, stderr io.Writer) error {
	fs := newFlagSet("init", stderr)
	user := fs.String("user", "", "GitHub login (defaults to the account of gh)")
	yes := fs.Bool("yes", false, "answer yes to the confirmations")
	force := fs.Bool("force", false, "rewrite the workflow and ghstats.yaml even when they exist")
	noRun := fs.Bool("no-run", false, "do not start the first run")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	r := &initRun{ctx: ctx, raw: in, in: bufio.NewReader(in), out: stdout, user: *user, yes: *yes, force: *force}

	if err := r.checkGh(); err != nil {
		return err
	}
	if r.user == "" {
		out, err := r.gh("api", "user", "--jq", ".login")
		if err != nil {
			return err
		}
		r.user = strings.TrimSpace(string(out))
		if r.user == "" {
			return errors.New("could not read the login from gh: pass --user")
		}
	}
	profile, data := r.user+"/"+r.user, r.user+"/ghstats-data"

	if err := r.ensureProfileRepo(profile); err != nil {
		return err
	}
	if err := r.ensureDataRepo(data); err != nil {
		return err
	}

	workflow := strings.ReplaceAll(workflowTemplate, "<user>", r.user)
	if _, err := r.ensureFile(profile, workflowPath, "chore: add the ghstats workflow", []byte(workflow)); err != nil {
		return err
	}
	cfgData, err := r.ensureFile(profile, configPath, "chore: add the ghstats configuration", []byte(profileConfigSample))
	if err != nil {
		return err
	}
	cfg, err := config.Parse(bytes.NewReader(cfgData))
	if err != nil {
		cfg = config.Default()
	}
	if err := r.ensureReadme(profile, cfg); err != nil {
		return err
	}
	if err := r.ensureSecret(profile); err != nil {
		return err
	}
	if *noRun {
		r.status("skipped", "first run (--no-run)")
		return nil
	}
	if _, err := r.gh("workflow", "run", "ghstats.yml", "-R", profile); err != nil {
		return fmt.Errorf("could not start the first run (start it from the Actions tab): %w", err)
	}
	r.status("created", "first run started")
	fmt.Fprintf(r.out, "https://github.com/%s/actions\n", profile)
	return nil
}

func (r *initRun) status(state, msg string) {
	fmt.Fprintf(r.out, "%-8s %s\n", state, msg)
}

func (r *initRun) gh(args ...string) ([]byte, error) {
	return ghExec(r.ctx, nil, args...)
}

func (r *initRun) confirm(question string) (bool, error) {
	if r.yes {
		return true, nil
	}
	fmt.Fprintf(r.out, "%s [y/N] ", question)
	line, err := r.in.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	}
	return false, nil
}

func (r *initRun) checkGh() error {
	out, err := r.gh("api", "-i", "user")
	if err != nil {
		if errors.Is(err, osexec.ErrNotFound) {
			return fmt.Errorf("gh not found: install the GitHub CLI from %s", ghInstallURL)
		}
		return fmt.Errorf("gh is not usable (run gh auth login): %w", err)
	}
	if !slices.Contains(oauthScopes(out), "workflow") {
		return errors.New("the gh token lacks the workflow scope, needed to write .github/workflows: run gh auth refresh -h github.com -s workflow")
	}
	r.status("ok", "gh is installed and has the workflow scope")
	return nil
}

func oauthScopes(out []byte) []string {
	var scopes []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if ok && strings.EqualFold(strings.TrimSpace(name), "x-oauth-scopes") {
			for _, s := range strings.Split(value, ",") {
				if s = strings.TrimSpace(s); s != "" {
					scopes = append(scopes, s)
				}
			}
		}
	}
	return scopes
}

func (r *initRun) visibility(repo string) (string, bool, error) {
	out, err := r.gh("repo", "view", repo, "--json", "visibility")
	if err != nil {
		if isNotFound(err) {
			return "", false, nil
		}
		return "", false, err
	}
	var v struct {
		Visibility string `json:"visibility"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return "", false, fmt.Errorf("decode gh repo view %s: %w", repo, err)
	}
	return strings.ToUpper(v.Visibility), true, nil
}

func (r *initRun) ensureProfileRepo(repo string) error {
	_, ok, err := r.visibility(repo)
	if err != nil {
		return err
	}
	if ok {
		r.status("ok", "profile repository "+repo)
		return nil
	}
	yes, err := r.confirm(fmt.Sprintf("Create the public profile repository %s?", repo))
	if err != nil {
		return err
	}
	if !yes {
		return fmt.Errorf("the profile repository %s is required", repo)
	}
	if _, err := r.gh("repo", "create", repo, "--public", "--add-readme"); err != nil {
		return err
	}
	r.status("created", "profile repository "+repo)
	return nil
}

func (r *initRun) ensureDataRepo(repo string) error {
	vis, ok, err := r.visibility(repo)
	if err != nil {
		return err
	}
	if !ok {
		if _, err := r.gh("repo", "create", repo, "--private", "--add-readme"); err != nil {
			return err
		}
		r.status("created", "data repository "+repo+" (private)")
		return nil
	}
	if vis == "PUBLIC" {
		return fmt.Errorf("the data repository %s is public but holds private repository names: make it private, then run init again", repo)
	}
	r.status("ok", "data repository "+repo)
	return nil
}

type remoteFile struct {
	found   bool
	sha     string
	content []byte
}

func (r *initRun) getFile(repo, path string) (remoteFile, error) {
	out, err := r.gh("api", "repos/"+repo+"/contents/"+path)
	if err != nil {
		if isNotFound(err) {
			return remoteFile{}, nil
		}
		return remoteFile{}, err
	}
	var f struct {
		SHA     string `json:"sha"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(out, &f); err != nil {
		return remoteFile{}, fmt.Errorf("decode %s: %w", path, err)
	}
	data, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(f.Content, "\n", ""))
	if err != nil {
		return remoteFile{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return remoteFile{found: true, sha: f.SHA, content: data}, nil
}

func (r *initRun) putFile(repo, path, message string, content []byte, sha string) error {
	body := map[string]string{"message": message, "content": base64.StdEncoding.EncodeToString(content)}
	if sha != "" {
		body["sha"] = sha
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	_, err = ghExec(r.ctx, b, "api", "-X", "PUT", "repos/"+repo+"/contents/"+path, "--input", "-")
	return err
}

func (r *initRun) ensureFile(repo, path, message string, content []byte) ([]byte, error) {
	f, err := r.getFile(repo, path)
	if err != nil {
		return nil, err
	}
	switch {
	case f.found && !r.force:
		r.status("ok", path)
		return f.content, nil
	case f.found:
		if err := r.putFile(repo, path, message, content, f.sha); err != nil {
			return nil, err
		}
		r.status("updated", path)
	default:
		if err := r.putFile(repo, path, message, content, ""); err != nil {
			return nil, err
		}
		r.status("created", path)
	}
	return content, nil
}

func readmeBlock(snippet string) string {
	return readmeStart + "\n" + strings.TrimRight(snippet, "\n") + "\n" + readmeEnd
}

func mergeReadme(existing, snippet string) string {
	block := readmeBlock(snippet)
	if si := strings.Index(existing, readmeStart); si >= 0 {
		if ei := strings.Index(existing[si:], readmeEnd); ei >= 0 {
			return existing[:si] + block + existing[si+ei+len(readmeEnd):]
		}
	}
	switch {
	case existing == "":
		return block + "\n"
	case strings.HasSuffix(existing, "\n\n"):
		return existing + block + "\n"
	case strings.HasSuffix(existing, "\n"):
		return existing + "\n" + block + "\n"
	}
	return existing + "\n\n" + block + "\n"
}

func (r *initRun) ensureReadme(repo string, cfg config.Config) error {
	f, err := r.getFile(repo, readmePath)
	if err != nil {
		return err
	}
	snippet := render.Snippet(cfg)
	next := mergeReadme(string(f.content), snippet)
	if f.found && next == string(f.content) {
		r.status("ok", readmePath)
		return nil
	}
	if !r.yes {
		fmt.Fprintf(r.out, "\n%s\n\n", readmeBlock(snippet))
	}
	yes, err := r.confirm("Write it to README.md?")
	if err != nil {
		return err
	}
	if !yes {
		r.status("skipped", readmePath+": paste the block above by hand")
		return nil
	}
	if err := r.putFile(repo, readmePath, "docs: add the ghstats cards", []byte(next), f.sha); err != nil {
		return err
	}
	if f.found {
		r.status("updated", readmePath)
	} else {
		r.status("created", readmePath)
	}
	return nil
}

type secretEntry struct {
	Name string `json:"name"`
}

func (r *initRun) ensureSecret(repo string) error {
	out, err := r.gh("secret", "list", "-R", repo, "--json", "name")
	if err != nil {
		return err
	}
	var secrets []secretEntry
	if err := json.Unmarshal(out, &secrets); err != nil {
		return fmt.Errorf("decode gh secret list: %w", err)
	}
	present := slices.ContainsFunc(secrets, func(s secretEntry) bool { return s.Name == secretName })

	token := strings.TrimSpace(os.Getenv(secretName))
	if token == "" && present {
		r.status("ok", "secret "+secretName)
		return nil
	}
	if token == "" {
		if token, err = r.readToken(); err != nil {
			return err
		}
		if token == "" {
			return fmt.Errorf("no token: set %s or enter one when asked (see %s)", secretName, tokenDocsURL)
		}
	}
	if err := r.verifyToken(token); err != nil {
		return err
	}
	if _, err := ghExec(r.ctx, []byte(token), "secret", "set", secretName, "-R", repo); err != nil {
		return fmt.Errorf("could not set the secret: %s", redact(err.Error(), token))
	}
	if present {
		r.status("updated", "secret "+secretName)
	} else {
		r.status("created", "secret "+secretName)
	}
	return nil
}

func (r *initRun) readToken() (string, error) {
	fmt.Fprintf(r.out, "%s (fine-grained token): ", secretName)
	if f, ok := r.raw.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(r.out)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	line, err := r.in.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func redact(msg, token string) string {
	if token == "" {
		return msg
	}
	return strings.ReplaceAll(msg, token, "***")
}

func (r *initRun) verifyToken(token string) error {
	endpoint := os.Getenv("GHSTATS_GRAPHQL_URL")
	if endpoint == "" {
		endpoint = defaultGraphQLURL
	}
	hc := &http.Client{Timeout: initHTTPTimeout}
	gql := &ghapi.Client{HTTP: hc, Endpoint: endpoint, Token: token}
	var res struct {
		Viewer struct {
			Login string `json:"login"`
		} `json:"viewer"`
	}
	if err := gql.Do(r.ctx, "query { viewer { login } }", nil, &res); err != nil {
		return fmt.Errorf("GitHub refused the token: %s", redact(err.Error(), token))
	}
	if !strings.EqualFold(res.Viewer.Login, r.user) {
		return fmt.Errorf("the token belongs to %q, not %q: create it on the %s account", res.Viewer.Login, r.user, r.user)
	}

	base := os.Getenv("GHSTATS_REST_URL")
	if base == "" {
		base = defaultRESTURL
	}
	req, err := http.NewRequestWithContext(r.ctx, http.MethodGet, strings.TrimRight(base, "/")+"/repos/"+r.user+"/ghstats-data", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("could not check access to %s/ghstats-data: %s", r.user, redact(err.Error(), token))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("the token cannot read %s/ghstats-data (HTTP %d): check its repository permissions, see %s", r.user, resp.StatusCode, tokenDocsURL)
	}
	return nil
}
