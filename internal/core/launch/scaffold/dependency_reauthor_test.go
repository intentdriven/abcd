package scaffold

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// itd-2609221842494980 / spc-2609221843355061: a bot-opened dependency bump is
// re-authored as the repository owner only inside the bound, the attribution
// gate is unchanged and judges the result, and the scaffold lays it only for a
// repository that opts in.

const (
	reauthorBot      = "dependabot[bot]"
	reauthorBotMail  = "49699333+dependabot[bot]@users.noreply.github.com"
	reauthorOwner    = "Example Owner"
	reauthorOwnerTo  = "owner@example.com"
	reauthorRepo     = "example/fixture"
	reauthorBranch   = "dependabot/go_modules/example.com/lib-1.2.3"
	reauthorAppID    = "123456"
	reauthorAppToken = "fixture-installation-credential"
)

// reauthorTools skips a run that needs the script's own tools where they are
// absent on a developer machine, and fails in CI, where every runner has them.
func reauthorTools(t *testing.T, tools ...string) {
	t.Helper()
	for _, tool := range append([]string{"bash", "git"}, tools...) {
		if _, err := exec.LookPath(tool); err != nil {
			if os.Getenv("CI") != "" {
				t.Fatalf("%s is not on PATH in CI", tool)
			}
			t.Skipf("%s is not on PATH", tool)
		}
	}
}

// reauthorFixture is a repository holding the declaration, a base commit by a
// person and, on the bot's branch, one bump commit by the bot touching go.mod
// and go.sum. It returns the repository, the base and the bump's head.
type reauthorFixture struct {
	dir, base, head, script string
}

func (f reauthorFixture) git(t *testing.T, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = f.dir
	cmd.Env = append(gittest.Env(t), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func identity(name, mail string) []string {
	return []string{"GIT_AUTHOR_NAME=" + name, "GIT_AUTHOR_EMAIL=" + mail,
		"GIT_COMMITTER_NAME=" + name, "GIT_COMMITTER_EMAIL=" + mail}
}

func botIdentity() []string {
	return []string{"GIT_AUTHOR_NAME=" + reauthorBot, "GIT_AUTHOR_EMAIL=" + reauthorBotMail,
		"GIT_COMMITTER_NAME=GitHub", "GIT_COMMITTER_EMAIL=noreply@github.com"}
}

func newReauthorFixture(t *testing.T, owner string, bump map[string]string) reauthorFixture {
	t.Helper()
	rendered, err := Render(AbcdSubstitutions())
	if err != nil {
		t.Fatal(err)
	}
	conf := strings.Replace(string(rendered.ReauthorConf), "owner_name=\nowner_email=\n", owner, 1)
	return newReauthorFixtureConf(t, conf, nil, bump)
}

// newReauthorFixtureConf is newReauthorFixture with the declaration given
// whole and extra files in the base commit.
func newReauthorFixtureConf(t *testing.T, conf string, base, bump map[string]string) reauthorFixture {
	t.Helper()
	reauthorTools(t)
	rendered, err := Render(AbcdSubstitutions())
	if err != nil {
		t.Fatal(err)
	}
	f := reauthorFixture{dir: t.TempDir()}
	f.script = filepath.Join(t.TempDir(), "dependency-reauthor.sh")
	if err := os.WriteFile(f.script, rendered.ReauthorScript, 0o644); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(f.dir, filepath.FromSlash(ReauthorConfPath)), conf)
	mustWrite(t, filepath.Join(f.dir, "go.mod"), "module example.com/fixture\n\nrequire example.com/lib v1.2.2\n")
	mustWrite(t, filepath.Join(f.dir, "go.sum"), "example.com/lib v1.2.2 h1:old=\n")
	mustWrite(t, filepath.Join(f.dir, "README.md"), "fixture\n")
	for name, body := range base {
		mustWrite(t, filepath.Join(f.dir, filepath.FromSlash(name)), body)
	}
	f.git(t, nil, "init", "-q", "-b", "main")
	f.git(t, nil, "add", "-A")
	f.git(t, identity("Base Person", "base@example.com"), "commit", "-q", "-m", "base\n\nAssisted-by: None")
	f.base = f.git(t, nil, "rev-parse", "HEAD")
	f.git(t, nil, "checkout", "-q", "-b", reauthorBranch)
	if bump == nil {
		bump = map[string]string{
			"go.mod": "module example.com/fixture\n\nrequire example.com/lib v1.2.3\n",
			"go.sum": "example.com/lib v1.2.3 h1:new=\n",
		}
	}
	for name, body := range bump {
		mustWrite(t, filepath.Join(f.dir, filepath.FromSlash(name)), body)
	}
	f.git(t, nil, "add", "-A")
	f.git(t, botIdentity(), "commit", "-q", "-m",
		"build(deps): bump example.com/lib from 1.2.2 to 1.2.3\n\nSigned-off-by: dependabot[bot] <support@github.com>")
	f.head = f.git(t, nil, "rev-parse", "HEAD")
	f.git(t, nil, "checkout", "-q", "main")
	return f
}

const ownerDeclared = "owner_name=" + reauthorOwner + "\nowner_email=" + reauthorOwnerTo + "\n"

// event is the environment the workflow fills from a pull request by the bot.
func (f reauthorFixture) event() map[string]string {
	return map[string]string{
		"PR_AUTHOR": reauthorBot, "ACTOR": reauthorBot, "HEAD_REF": reauthorBranch, "HEAD_SHA": f.head,
		"BASE_SHA": f.base, "HEAD_REPO": reauthorRepo, "GITHUB_REPOSITORY": reauthorRepo,
	}
}

// runScript runs the rendered script in the fixture with the event plus
// overrides (a "" value removes the variable), returning its output and exit.
func (f reauthorFixture) runScript(t *testing.T, mode string, over map[string]string) (string, int) {
	t.Helper()
	vars := f.event()
	for k, v := range over {
		vars[k] = v
	}
	env := gittest.Env(t)
	for k, v := range vars {
		if v != "" {
			env = append(env, k+"="+v)
		}
	}
	cmd := exec.Command("bash", f.script, mode)
	cmd.Dir = f.dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	var ee *exec.ExitError
	switch {
	case err == nil:
		return string(out), 0
	case errors.As(err, &ee):
		return string(out), ee.ExitCode()
	default:
		t.Fatalf("run script: %v\n%s", err, out)
		return "", -1
	}
}

// Criterion 1's test half: an in-bound bump is recognised as one.
func TestReauthorBoundAdmitsAManifestAndLockBump(t *testing.T) {
	f := newReauthorFixture(t, ownerDeclared, nil)
	out, code := f.runScript(t, "check", nil)
	if code != 0 || !strings.Contains(out, "in bound: dependabot[bot], dependabot/go_modules/, 2 file(s)") {
		t.Fatalf("an in-bound bump was not admitted (exit %d):\n%s", code, out)
	}
}

// Criteria 2 and 5: anything outside the bound is left alone, green, with the
// clause it failed named.
func TestReauthorBoundLeavesEverythingElseAloneAndNamesTheClause(t *testing.T) {
	cases := []struct {
		name   string
		bump   map[string]string
		over   map[string]string
		clause string
	}{
		{"a file outside the ecosystem's pair", map[string]string{"go.mod": "module x\n", "README.md": "changed\n"}, nil,
			"left alone (diff): README.md is not one of the declared files"},
		{"an unlisted bot", nil, map[string]string{"PR_AUTHOR": "renovate[bot]"},
			"left alone (author): the pull request's author renovate[bot] is not a bot this repository declares"},
		{"a person", nil, map[string]string{"PR_AUTHOR": "some-person"},
			"left alone (author): the pull request's author some-person"},
		{"an undeclared ecosystem", nil, map[string]string{"HEAD_REF": "dependabot/github_actions/actions/checkout-7"},
			"left alone (ecosystem): the branch dependabot/github_actions/actions/checkout-7 matches no ecosystem"},
		{"a fork's branch", nil, map[string]string{"HEAD_REPO": "someone/fixture"},
			"left alone (head-repo): the branch lives in someone/fixture"},
		// A new manifest elsewhere carves a module out of the tree the checks
		// run over, so an added file is never a bump, wherever it lands.
		{"an added manifest in another directory", map[string]string{"internal/go.mod": "module example.com/fixture/internal\n"}, nil,
			"left alone (diff): internal/go.mod is A in the diff"},
		{"an added manifest where the row declares it", map[string]string{"requirements.txt": "lib==1.2.3\n"},
			map[string]string{"HEAD_REF": "dependabot/pip/lib-1.2.3"},
			"left alone (diff): requirements.txt is A in the diff"},
		// The bot opened it but a person pushed it: the pusher is judged too.
		{"a pusher who is not the bot", nil, map[string]string{"ACTOR": "some-person"},
			"left alone (actor): the event's actor some-person is not dependabot[bot]"},
		{"a branch name past the cap", nil, map[string]string{"HEAD_REF": "dependabot/go_modules/" + strings.Repeat("a", 256-len("dependabot/go_modules/"))},
			"left alone (branch): the branch name is 256 characters; the cap is 255"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newReauthorFixture(t, ownerDeclared, c.bump)
			out, code := f.runScript(t, "run", c.over)
			if code != 0 || !strings.Contains(out, c.clause) {
				t.Fatalf("want exit 0 naming %q, got exit %d:\n%s", c.clause, code, out)
			}
			if strings.Contains(out, "re-authored") {
				t.Fatalf("a commit outside the bound was re-authored:\n%s", out)
			}
		})
	}
}

// A row matches the full path: its directory, as .github/dependabot.yml
// declares it, joined to each file it names. The same file name in any other
// directory is outside the bound, and a row without a directory is malformed.
func TestReauthorBoundMatchesTheDeclaredDirectory(t *testing.T) {
	rows := "ecosystem=dependabot[bot] dependabot/go_modules/ / go.mod go.sum\n" +
		"ecosystem=dependabot[bot] dependabot/pip/ /docs requirements.txt\n"
	base := map[string]string{"docs/requirements.txt": "lib==1.2.2\n", "requirements.txt": "lib==1.2.2\n", "tools/go.mod": "module example.com/tools\n"}
	pip := map[string]string{"HEAD_REF": "dependabot/pip/docs/lib-1.2.3"}
	cases := []struct {
		name string
		bump map[string]string
		over map[string]string
		want string
	}{
		{"the declared directory", map[string]string{"docs/requirements.txt": "lib==1.2.3\n"}, pip,
			"in bound: dependabot[bot], dependabot/pip/, 1 file(s)"},
		{"the file name at the root", map[string]string{"requirements.txt": "lib==1.2.3\n"}, pip,
			"left alone (diff): requirements.txt is not one of the declared files for dependabot/pip/: docs/requirements.txt"},
		{"the file name in another directory", map[string]string{"tools/go.mod": "module example.com/tools\n\ngo 1.26\n"}, nil,
			"left alone (diff): tools/go.mod is not one of the declared files for dependabot/go_modules/: go.mod go.sum"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newReauthorFixtureConf(t, ownerDeclared+rows, base, c.bump)
			out, code := f.runScript(t, "check", c.over)
			if code != 0 || !strings.Contains(out, c.want) {
				t.Fatalf("want exit 0 with %q, got exit %d:\n%s", c.want, code, out)
			}
		})
	}
	t.Run("a row without a directory", func(t *testing.T) {
		f := newReauthorFixtureConf(t, ownerDeclared+"ecosystem=dependabot[bot] dependabot/go_modules/ go.mod go.sum\n", nil, nil)
		out, code := f.runScript(t, "check", nil)
		if want := "an ecosystem row is <bot> <branch prefix> <directory> <file>..."; code != 1 || !strings.Contains(out, want) {
			t.Fatalf("want exit 1 with %q, got exit %d:\n%s", want, code, out)
		}
	})
}

// abcd's own rows carry, for each ecosystem, the directory its
// .github/dependabot.yml opens bumps in.
func TestAbcdsReauthorRowsCarryTheDirectoriesDependabotDeclares(t *testing.T) {
	branchSegment := map[string]string{"gomod": "go_modules", "pip": "pip", "github-actions": "github_actions",
		"npm": "npm_and_yarn", "bundler": "bundler", "cargo": "cargo"}
	declared := map[string]bool{}
	ecosystem := ""
	for _, line := range strings.Split(readFile(t, filepath.Join(repoRoot(t), ".github", "dependabot.yml")), "\n") {
		line = strings.TrimSpace(line)
		if v, ok := strings.CutPrefix(line, "- package-ecosystem:"); ok {
			ecosystem = branchSegment[strings.TrimSpace(v)]
		} else if v, ok := strings.CutPrefix(line, "directory:"); ok && ecosystem != "" {
			declared["dependabot/"+ecosystem+"/ "+strings.TrimSpace(v)] = true
		}
	}
	rows := 0
	for _, line := range strings.Split(readFile(t, filepath.Join(repoRoot(t), filepath.FromSlash(ReauthorConfPath))), "\n") {
		v, ok := strings.CutPrefix(line, "ecosystem=")
		if !ok {
			continue
		}
		rows++
		fields := strings.Fields(v)
		if len(fields) < 4 || !declared[fields[1]+" "+fields[2]] {
			t.Errorf("the row %q names no directory .github/dependabot.yml declares for its ecosystem (%v)", line, declared)
		}
	}
	if rows == 0 {
		t.Fatalf("abcd's declaration carries no ecosystem row")
	}
}

// A branch a person added a commit to, and a bump already re-authored (the run
// its own push starts), are both left alone.
func TestReauthorLeavesAPersonsCommitAndItsOwnResultAlone(t *testing.T) {
	f := newReauthorFixture(t, ownerDeclared, nil)
	f.git(t, nil, "checkout", "-q", reauthorBranch)
	mustWrite(t, filepath.Join(f.dir, "go.sum"), "example.com/lib v1.2.3 h1:fixed=\n")
	f.git(t, identity("Some Person", "person@example.com"), "commit", "-q", "-am", "fix sum\n\nAssisted-by: None")
	two := f.git(t, nil, "rev-parse", "HEAD")
	out, code := f.runScript(t, "run", map[string]string{"HEAD_SHA": two})
	if code != 0 || !strings.Contains(out, "left alone (commits): the branch carries 2 commits") {
		t.Fatalf("two commits: exit %d:\n%s", code, out)
	}
	f.git(t, nil, "reset", "-q", "--hard", f.head)
	f.git(t, identity(reauthorOwner, reauthorOwnerTo), "commit", "-q", "--amend", "--reset-author", "-m", "bump\n\nAssisted-by: None")
	mine := f.git(t, nil, "rev-parse", "HEAD")
	out, code = f.runScript(t, "run", map[string]string{"HEAD_SHA": mine})
	if code != 0 || !strings.Contains(out, "left alone (commit-author): the head commit is authored by Example Owner") {
		t.Fatalf("an already re-authored head: exit %d:\n%s", code, out)
	}
}

// The owner is the person's to set and each secret is the person's to create:
// an in-bound bump with any of them missing is refused by name, and nothing is
// pushed — there is no fallback to GITHUB_TOKEN or to the bot as author.
func TestReauthorRefusesWithoutTheOwnerOrEitherSecret(t *testing.T) {
	cases := []struct {
		name, owner string
		over        map[string]string
		want        string
	}{
		{"no owner", "owner_name=\nowner_email=\n", nil, "refused: owner_name is empty"},
		{"no owner address", "owner_name=" + reauthorOwner + "\nowner_email=\n", nil, "refused: owner_email is empty"},
		{"no app id", ownerDeclared, map[string]string{"DEPENDENCY_REAUTHOR_APP_KEY": "k"},
			"refused: the secret DEPENDENCY_REAUTHOR_APP_ID is not set"},
		{"no app key", ownerDeclared, map[string]string{"DEPENDENCY_REAUTHOR_APP_ID": reauthorAppID},
			"refused: the secret DEPENDENCY_REAUTHOR_APP_KEY is not set"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newReauthorFixture(t, c.owner, nil)
			remote := f.bareRemote(t)
			over := map[string]string{"REAUTHOR_REMOTE": remote, "GITHUB_TOKEN": "fixture-workflow-token"}
			for k, v := range c.over {
				over[k] = v
			}
			out, code := f.runScript(t, "run", over)
			if code != 1 || !strings.Contains(out, c.want) {
				t.Fatalf("want exit 1 with %q, got exit %d:\n%s", c.want, code, out)
			}
			if got := remoteHead(t, remote); got != f.head {
				t.Fatalf("a refused run moved the branch to %s", got)
			}
		})
	}
}

func (f reauthorFixture) bareRemote(t *testing.T) string {
	t.Helper()
	remote := filepath.Join(t.TempDir(), "remote.git")
	f.git(t, nil, "clone", "-q", "--bare", f.dir, remote)
	return remote
}

func remoteHead(t *testing.T, remote string) string {
	t.Helper()
	cmd := exec.Command("git", "--git-dir", remote, "rev-parse", "refs/heads/"+reauthorBranch)
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("read the remote branch: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// appForge stands in for the forge's App API: it verifies the App's signed
// token against the public half of the key, hands out one installation token
// scoped to the repository's contents, and records its revocation.
type appForge struct {
	t       *testing.T
	pub     *rsa.PublicKey
	mu      sync.Mutex
	scope   string
	revoked bool
}

func (g *appForge) verifyJWT(r *http.Request) bool {
	auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	parts := strings.Split(auth, ".")
	if len(parts) != 3 {
		return false
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(g.pub, crypto.SHA256, sum[:], sig) != nil {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var claims struct {
		Iss      string `json:"iss"`
		Iat, Exp int64
	}
	return json.Unmarshal(raw, &claims) == nil && claims.Iss == reauthorAppID && claims.Exp > claims.Iat
}

func (g *appForge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/repos/"+reauthorRepo+"/installation":
		if !g.verifyJWT(r) {
			http.Error(w, "bad jwt", http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, `{"id": 42, "account": {"id": 7}}`)
	case r.Method == http.MethodPost && r.URL.Path == "/app/installations/42/access_tokens":
		if !g.verifyJWT(r) {
			http.Error(w, "bad jwt", http.StatusUnauthorized)
			return
		}
		body, _ := io.ReadAll(r.Body)
		g.scope = string(body)
		_, _ = io.WriteString(w, `{"token": "`+reauthorAppToken+`", "expires_at": "2030-01-01T00:00:00Z"}`)
	case r.Method == http.MethodDelete && r.URL.Path == "/installation/token":
		if r.Header.Get("Authorization") != "Bearer "+reauthorAppToken {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		g.revoked = true
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusNotFound)
	}
}

// Criteria 1, 3, 4 and 7 end to end, offline: the App token is minted from the
// two secrets against a stand-in forge, the bump is replayed with the owner as
// author and committer, the branch is pushed under a lease, the token is
// revoked, the run records what it did, and the UNCHANGED attribution gate
// passes the re-authored commit while it still refuses the bot's original.
func TestReauthorReplaysAnInBoundBumpAsTheOwnerAndTheGatePassesIt(t *testing.T) {
	reauthorTools(t, "openssl", "curl", "jq")
	// The gate's outbound half runs from a binary built from this checkout,
	// built before the fixture isolates HOME so the module cache is the caller's.
	bin := filepath.Join(t.TempDir(), "abcd")
	build := exec.Command("go", "build", "-o", bin, "./cmd/abcd")
	build.Dir = repoRoot(t)
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build the outbound checker: %v\n%s", err, b)
	}
	f := newReauthorFixture(t, ownerDeclared, nil)
	remote := f.bareRemote(t)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	forge := &appForge{t: t, pub: &key.PublicKey}
	srv := httptest.NewServer(forge)
	defer srv.Close()
	summary := filepath.Join(t.TempDir(), "summary.md")

	out, code := f.runScript(t, "run", map[string]string{
		"DEPENDENCY_REAUTHOR_APP_ID": reauthorAppID, "DEPENDENCY_REAUTHOR_APP_KEY": pemKey,
		"GITHUB_API_URL": srv.URL, "REAUTHOR_REMOTE": remote, "GITHUB_STEP_SUMMARY": summary,
		"GITHUB_ACTIONS": "true",
	})
	if code != 0 {
		t.Fatalf("an in-bound run failed (exit %d):\n%s", code, out)
	}
	// The installation token appears once, on the runner's mask command, which
	// the runner consumes rather than prints; the key never appears at all.
	mask := "::add-mask::" + reauthorAppToken + "\n"
	if !strings.Contains(out, mask) {
		t.Errorf("the run did not mask the installation token:\n%s", out)
	}
	if rest := strings.Replace(out, mask, "", 1); strings.Contains(rest, pemKey) || strings.Contains(rest, reauthorAppToken) {
		t.Fatalf("the run printed a credential:\n%s", out)
	}
	moved := remoteHead(t, remote)
	if moved == f.head {
		t.Fatalf("the branch was not updated:\n%s", out)
	}
	f.git(t, nil, "fetch", "-q", remote, "refs/heads/"+reauthorBranch)
	show := func(format string) string { return f.git(t, nil, "show", "-s", "--format="+format, moved) }
	if got := show("%an <%ae>|%cn <%ce>"); got != reauthorOwner+" <"+reauthorOwnerTo+">|"+reauthorOwner+" <"+reauthorOwnerTo+">" {
		t.Errorf("author|committer = %q, want the owner in both roles", got)
	}
	if show("%T") != f.git(t, nil, "show", "-s", "--format=%T", f.head) || show("%P") != f.base {
		t.Errorf("the re-authored commit changed the tree or the parent")
	}
	msg := show("%B")
	for _, want := range []string{
		"build(deps): bump example.com/lib from 1.2.2 to 1.2.3\n",
		"Proposed by dependabot[bot] as " + f.head,
		"re-authored by .github/workflows/dependency-reauthor.yml",
		"\nAssisted-by: None",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("message lacks %q:\n%s", want, msg)
		}
	}
	if !strings.Contains(forge.scope, `"repositories":["fixture"]`) || !strings.Contains(forge.scope, `"contents":"write"`) {
		t.Errorf("the token was not scoped to this repository's contents: %s", forge.scope)
	}
	if !forge.revoked {
		t.Errorf("the installation token was not revoked")
	}
	rec := readFile(t, summary)
	if !strings.Contains(rec, "re-authored dependabot[bot] bump \"build(deps): bump example.com/lib from 1.2.2 to 1.2.3\" on "+
		reauthorBranch+": "+f.head+" -> "+moved) {
		t.Errorf("no record names the bot, the bump and the commit:\n%s", rec)
	}

	// The gate, unchanged, on both commits.
	gate := filepath.Join(repoRoot(t), "scripts", "check-attribution.sh")
	judge := func(head string) (string, error) {
		cmd := exec.Command("bash", gate, "commits", f.base, head)
		cmd.Dir = f.dir
		cmd.Env = append(gittest.Env(t), "ABCD_OUTBOUND_BIN="+bin)
		b, err := cmd.CombinedOutput()
		return string(b), err
	}
	if b, err := judge(moved); err != nil {
		t.Errorf("the attribution gate refused the re-authored commit: %v\n%s", err, b)
	}
	if b, err := judge(f.head); err == nil || !strings.Contains(b, "a machine author identity") {
		t.Errorf("the attribution gate passed the bot's original: %v\n%s", err, b)
	}

	// A branch the bot moved meanwhile is not overwritten: the lease refuses.
	g := newReauthorFixture(t, ownerDeclared, nil)
	gremote := g.bareRemote(t)
	g.git(t, nil, "push", "-q", "--force", gremote, g.base+":refs/heads/"+reauthorBranch)
	out, code = g.runScript(t, "run", map[string]string{
		"DEPENDENCY_REAUTHOR_APP_ID": reauthorAppID, "DEPENDENCY_REAUTHOR_APP_KEY": pemKey,
		"GITHUB_API_URL": srv.URL, "REAUTHOR_REMOTE": gremote,
	})
	if code != 1 || !strings.Contains(out, "refused: the push of") || remoteHead(t, gremote) != g.base {
		t.Fatalf("a moved branch was overwritten (exit %d):\n%s", code, out)
	}
}

// Criterion 6: the scaffold lays the re-authoring only for a repository that
// opts in — by the flag, which seeds the declaration, or by the declaration
// already being there — and keeps the declaration as the repository's own.
func TestScaffoldLaysTheReauthoringOnlyWhenOptedIn(t *testing.T) {
	reauthorPaths := []string{ReauthorConfPath, ReauthorYMLPath, ReauthorScriptPath}
	for _, kind := range []string{"plugin", "binary"} {
		t.Run(kind, func(t *testing.T) {
			dir := kindRepo(t, kind)
			rep, err := Scaffold(Request{RepoRoot: dir})
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range reauthorPaths {
				if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(p))); err == nil {
					t.Errorf("a repository that has not opted in received %s", p)
				}
			}
			if rep.DependencyReauthor {
				t.Errorf("report says opted in without the flag or a declaration")
			}

			rep, err = Scaffold(Request{RepoRoot: dir, DependencyReauthor: true})
			if err != nil {
				t.Fatalf("%v: %+v", err, rep.Files)
			}
			for _, p := range reauthorPaths {
				if o, ok := statusOf(rep, p); !ok || o.Status != StatusWritten {
					t.Errorf("opt-in: %s = %+v, want written", p, o)
				}
			}
			rendered, _ := Render(BareSubstitutions("main"))
			if got := readFile(t, filepath.Join(dir, ".github", "workflows", "dependency-reauthor.yml")); got != string(rendered.ReauthorYML) ||
				!strings.Contains(got, "bash "+ReauthorScriptPath+" run") {
				t.Errorf("the workflow is not the managed rendering, or does not run the script beside the runbook")
			}

			// The person sets the owner; a plain re-run keeps it and the machinery.
			conf := filepath.Join(dir, filepath.FromSlash(ReauthorConfPath))
			mustWrite(t, conf, strings.Replace(readFile(t, conf), "owner_name=\nowner_email=\n", ownerDeclared, 1))
			rep, err = Scaffold(Request{RepoRoot: dir})
			if err != nil {
				t.Fatal(err)
			}
			if o, _ := statusOf(rep, ReauthorConfPath); o.Status != StatusKept {
				t.Errorf("a set declaration was not kept: %+v", o)
			}
			if o, _ := statusOf(rep, ReauthorYMLPath); o.Status != StatusCurrent || !rep.DependencyReauthor {
				t.Errorf("the declaration did not keep the repository opted in: %+v", o)
			}
			if !strings.Contains(readFile(t, conf), ownerDeclared) {
				t.Errorf("the scaffold rewrote the owner")
			}
		})
	}
}

// The workflow reads event text only through the environment, holds the least
// permission, never persists a credential in the checkout, never takes the
// privileged trigger, and never pushes with the workflow's own token.
func TestReauthorWorkflowHoldsTheLeastItNeeds(t *testing.T) {
	for name, subs := range AuditProfiles() {
		rendered, err := Render(subs)
		if err != nil {
			t.Fatal(err)
		}
		doc := string(rendered.ReauthorYML)
		for _, dup := range duplicateKeys(doc) {
			t.Errorf("%s: duplicate key %s", name, dup)
		}
		for _, inj := range expressionsInRunScripts(doc) {
			t.Errorf("%s: a ${{ }} expression inside a run script: %s", name, inj)
		}
		for _, want := range []string{"\npermissions: {}\n", "      contents: read\n", "persist-credentials: false",
			"ref: ${{ github.event.pull_request.base.sha }}", "          ACTOR: ${{ github.actor }}\n"} {
			if !strings.Contains(doc, want) {
				t.Errorf("%s: the workflow lacks %q", name, want)
			}
		}
		code := uncommented(doc)
		for _, banned := range []string{"pull_request_target", "GITHUB_TOKEN", "github.token", "contents: write"} {
			if strings.Contains(code, banned) {
				t.Errorf("%s: the workflow carries %q", name, banned)
			}
		}
		for _, ref := range secretRefs(code) {
			if ref != "secrets.DEPENDENCY_REAUTHOR_APP_ID" && ref != "secrets.DEPENDENCY_REAUTHOR_APP_KEY" {
				t.Errorf("%s: the workflow reads %s", name, ref)
			}
		}
	}
}

// uncommented drops a workflow's comment lines, which explain what the
// workflow refuses and so name it.
func uncommented(doc string) string {
	var keep []string
	for _, line := range strings.Split(doc, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			keep = append(keep, line)
		}
	}
	return strings.Join(keep, "\n")
}
