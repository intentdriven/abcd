package banlist

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The pre-commit guard refreshes the sources corpus's generated block before it
// checks anything (itd-76 AC3, AC6). Both copies of the guard carry the step — the
// one this repository runs and the template `abcd ahoy` scaffolds into a managed
// repository — and they find the binary differently:
//
//   - this repository's copy builds ./cmd/abcd from the checkout, the rule for every
//     abcd invocation in a source checkout, and never runs an installed abcd
//     (iss-2609252007414882);
//   - the template runs only the binary `git config --local abcd.sourcesBinary`
//     names, an absolute path, and nothing when it is unset: how a scaffolded hook
//     finds a binary is a ruling owed to the product thinker, and opt-in decides
//     none of it (iss-2609252007419997).
//
// Either way the binary's own line is relayed, never discarded
// (iss-2609252007422630); the refresh updates an existing keyed store only
// (iss-2609252007426016); and a legacy store is named once, with its migration
// (iss-2609252007433563).
func guardCopies(t *testing.T) map[string]string {
	t.Helper()
	hook := locateHook(t)
	top := filepath.Dir(filepath.Dir(hook))
	return map[string]string{
		"repo":     hook,
		"template": filepath.Join(top, "internal", "core", "ahoy", "defaults", "pre-commit"),
	}
}

// newGuardRepo is newHookRepo with a chosen copy of the guard installed.
func newGuardRepo(t *testing.T, hookPath, body string) *hookRepo {
	t.Helper()
	r := newHookRepo(t, body)
	src, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.dir, ".git", "hooks", "pre-commit"), src, 0o755); err != nil {
		t.Fatal(err)
	}
	return r
}

// fakeAbcdOnPath puts an `abcd` on the guard's PATH (through the repo-local
// guardPath extension) that records its arguments, and returns the file the
// arguments land in. Neither copy of the guard may ever run it.
func fakeAbcdOnPath(t *testing.T, r *hookRepo) string {
	t.Helper()
	bin := t.TempDir()
	args := filepath.Join(t.TempDir(), "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" > '" + args + "'\n" + writeEntrySh + "\n"
	if err := os.WriteFile(filepath.Join(bin, "abcd"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	r.git("config", "--local", "abcd.guardPath", bin)
	return args
}

func makeCorpusDir(t *testing.T) {
	t.Helper()
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("no HOME")
	}
	if err := os.MkdirAll(abcdhome.Path(home, "sources"), 0o700); err != nil {
		t.Fatal(err)
	}
}

// A refresher is what one fake binary does, written once for each copy: as shell
// for the template's opted-in binary, and as Go for the source tree this
// repository's copy builds.
type refresher struct{ sh, gocode string }

const countLine = "abcd source sync-banlist — 1 confidential source, 1 pattern in .abcd/.work.local/private-names.txt (rewrote the block)"

const writeEntrySh = "printf '# abcd-banlist: keyed\\nsources/conf2026a/title widgetworks\\n' > .abcd/.work.local/private-names.txt; echo '" + countLine + "'"

var (
	writesEntry = refresher{
		sh: writeEntrySh,
		gocode: `os.MkdirAll(".abcd/.work.local", 0o700)
	os.WriteFile(".abcd/.work.local/private-names.txt", []byte("# abcd-banlist: keyed\nsources/conf2026a/title widgetworks\n"), 0o600)
	os.Stdout.WriteString("` + countLine + `\n")`,
	}
	// failsNamingAKey is a binary that HAS the verb (its help names --refresh, as
	// the real one's does) and refuses the refresh.
	failsNamingAKey = refresher{
		sh: `case "$*" in *--help*) echo '      --refresh   the refresh mode'; exit 0 ;; esac
echo 'abcd source sync-banlist: the corpus disagrees on conf2026a (nothing written)' >&2; exit 2`,
		gocode: `if strings.Contains(strings.Join(os.Args, " "), "--help") {
		os.Stdout.WriteString("      --refresh   the guard's mode\n")
		return
	}
	os.Stderr.WriteString("abcd source sync-banlist: the corpus disagrees on conf2026a (nothing written)\n")
	os.Exit(2)`,
	}
)

// install makes body the refresh binary the given copy of the guard runs, and
// returns the file its arguments land in.
func (f refresher) install(t *testing.T, r *hookRepo, copyName string) string {
	t.Helper()
	args := filepath.Join(t.TempDir(), "args")
	if copyName == "template" {
		bin := filepath.Join(t.TempDir(), "abcd")
		script := "#!/bin/sh\nprintf '%s\\n' \"$*\" > '" + args + "'\n" + f.sh + "\n"
		if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		r.git("config", "--local", "abcd.sourcesBinary", bin)
		return args
	}
	fakeSourceTree(t, r, `package main

import (
	"os"
	"strings"
)

func main() {
	os.WriteFile(`+"`"+args+"`"+`, []byte(strings.Join(os.Args[1:], " ")+"\n"), 0o600)
	`+f.gocode+`
}
`)
	return args
}

// fakeSourceTree makes the throwaway repository an abcd source checkout — a go.mod
// and a cmd/abcd holding main — so this repository's copy of the guard, installed
// under .git/hooks, finds it at the working tree root and builds it. The tree is
// never staged. The Go caches are the real ones (read before the test HOME moved),
// so the build is not a cold one.
func fakeSourceTree(t *testing.T, r *hookRepo, main string) {
	t.Helper()
	r.write("go.mod", "module example.com/fakeabcd\n\ngo 1.21\n")
	r.write("cmd/abcd/main.go", main)
	r.env = append(r.env, goCacheEnv(t)...)
}

var goCacheVals []string

// goCacheEnv is the Go environment the hook's build runs under.
func goCacheEnv(t *testing.T) []string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go unavailable: this repository's guard builds abcd")
	}
	if goCacheVals == nil {
		t.Skip("the Go caches were not read before HOME moved")
	}
	return append([]string{"GOTOOLCHAIN=local", "GOFLAGS=-mod=mod"}, goCacheVals...)
}

func init() {
	// Read once, before any test moves HOME: a HOME-relative default cache would
	// rebuild from cold per test.
	out, err := exec.Command("go", "env", "GOCACHE", "GOMODCACHE", "GOPATH").Output()
	if err != nil {
		return
	}
	vals := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(vals) != 3 {
		return
	}
	goCacheVals = []string{"GOCACHE=" + vals[0], "GOMODCACHE=" + vals[1], "GOPATH=" + vals[2]}
}

// commitNote stages one file carrying text and attempts a commit.
func commitNote(r *hookRepo, name, text string) (bool, string) {
	r.write(name, text)
	r.git("add", name)
	return r.commit()
}

// TestPreCommitHook_NoCorpusSaysSoAndProceeds is AC6 at the guard: no corpus is one
// line naming the skip, and the commit proceeds.
func TestPreCommitHook_NoCorpusSaysSoAndProceeds(t *testing.T) {
	for name, hook := range guardCopies(t) {
		t.Run(name, func(t *testing.T) {
			r := newGuardRepo(t, hook, "")
			blocked, out := commitNote(r, "note.md", "hello\n")
			if blocked {
				t.Fatalf("commit blocked with no corpus\n%s", out)
			}
			if strings.Count(out, "no sources corpus") != 1 {
				t.Fatalf("the guard does not say, once, that the corpus is absent\n%s", out)
			}
		})
	}
}

// TestPreCommitHook_NeverRunsAnInstalledAbcd: an abcd on the guard's PATH is never
// the refresh binary, in either copy. This repository's copy with no source tree to
// build, and the template with no opt-in, each say so on one line and proceed.
func TestPreCommitHook_NeverRunsAnInstalledAbcd(t *testing.T) {
	for name, hook := range guardCopies(t) {
		t.Run(name, func(t *testing.T) {
			makeCorpusDir(t)
			r := newGuardRepo(t, hook, privateFormatDecl+"\nhand-key zzunrelated\n")
			args := fakeAbcdOnPath(t, r)
			blocked, out := commitNote(r, "note.md", "the widgetworks draft\n")
			if _, err := os.Stat(args); err == nil {
				t.Fatalf("the guard ran the abcd on its PATH\n%s", out)
			}
			if blocked {
				t.Fatalf("the commit was blocked\n%s", out)
			}
			want := map[string]string{"repo": "cmd/abcd", "template": "abcd.sourcesBinary"}[name]
			if strings.Count(out, want) != 1 {
				t.Fatalf("the skip is not one line naming %q\n%s", want, out)
			}
		})
	}
}

// TestPreCommitHook_RefreshesTheSourcesBlockBeforeChecking is AC3 at the guard: the
// copy's own binary runs `source sync-banlist --refresh` BEFORE the store is read,
// so an entry the refresh writes is enforced on this very commit, and the binary's
// one-line count is relayed, so a refresh that wrote fewer patterns is visible.
func TestPreCommitHook_RefreshesTheSourcesBlockBeforeChecking(t *testing.T) {
	for name, hook := range guardCopies(t) {
		t.Run(name, func(t *testing.T) {
			makeCorpusDir(t)
			r := newGuardRepo(t, hook, privateFormatDecl+"\nhand-key zzunrelated\n")
			args := writesEntry.install(t, r, name)
			blocked, out := commitNote(r, "note.md", "the widgetworks draft\n")
			got, err := os.ReadFile(args)
			if err != nil {
				t.Fatalf("the guard did not run its refresh binary\n%s", out)
			}
			if strings.TrimSpace(string(got)) != "source sync-banlist --refresh" {
				t.Fatalf("the guard ran the binary with %q", got)
			}
			if !blocked || !strings.Contains(out, "sources/conf2026a/title") {
				t.Fatalf("the refreshed entry was not enforced on this commit\n%s", out)
			}
			if !strings.Contains(out, "pre-commit: "+countLine) {
				t.Fatalf("the binary's count line was not relayed\n%s", out)
			}
		})
	}
}

// TestPreCommitHook_AFailedRefreshIsRelayedAndTheStoreStillGuards: a refresh that
// fails is relayed in the binary's own words (which name keys only), never fatal,
// and the existing store still guards.
func TestPreCommitHook_AFailedRefreshIsRelayedAndTheStoreStillGuards(t *testing.T) {
	for name, hook := range guardCopies(t) {
		t.Run(name, func(t *testing.T) {
			makeCorpusDir(t)
			r := newGuardRepo(t, hook, privateFormatDecl+"\nhand-key widgetworks\n")
			failsNamingAKey.install(t, r, name)
			blocked, out := commitNote(r, "note.md", "the widgetworks draft\n")
			if !strings.Contains(out, "pre-commit: abcd source sync-banlist: the corpus disagrees on conf2026a") {
				t.Fatalf("the refresh's refusal was not relayed\n%s", out)
			}
			if !strings.Contains(out, "checked as it stands") {
				t.Fatalf("a failed refresh does not say the store is checked as it stands\n%s", out)
			}
			if !blocked || !strings.Contains(out, "hand-key") {
				t.Fatalf("the existing store stopped guarding after a failed refresh\n%s", out)
			}
		})
	}
}

// TestPreCommitHook_RepoCopyWarnsOnceWhenTheBuildFails: this repository's copy
// builds ./cmd/abcd, and a tree that does not build is ONE warning line; the commit
// proceeds and the store is checked as it stands.
func TestPreCommitHook_RepoCopyWarnsOnceWhenTheBuildFails(t *testing.T) {
	hook := guardCopies(t)["repo"]
	makeCorpusDir(t)
	r := newGuardRepo(t, hook, privateFormatDecl+"\nhand-key widgetworks\n")
	fakeSourceTree(t, r, "package main\n\nthis does not compile\n")
	blocked, out := commitNote(r, "note.md", "the widgetworks draft\n")
	if strings.Count(out, "could not build ./cmd/abcd") != 1 {
		t.Fatalf("a failed build is not one warning line\n%s", out)
	}
	if strings.Contains(out, "does not compile") || strings.Contains(out, "syntax error") {
		t.Fatalf("the build's own output was printed; the warning is one line\n%s", out)
	}
	if !blocked || !strings.Contains(out, "hand-key") {
		t.Fatalf("the store stopped guarding after a failed build\n%s", out)
	}
}

// TestPreCommitHook_TemplateRefusesAnUnusableOptIn: abcd.sourcesBinary must be an
// absolute path to an executable regular file. Anything else is one line, nothing
// runs, and the commit proceeds — no PATH search stands in for it.
func TestPreCommitHook_TemplateRefusesAnUnusableOptIn(t *testing.T) {
	hook := guardCopies(t)["template"]
	for _, value := range []string{"abcd", "bin/abcd", "/nonexistent/abcd"} {
		t.Run(value, func(t *testing.T) {
			makeCorpusDir(t)
			r := newGuardRepo(t, hook, privateFormatDecl+"\nhand-key zzunrelated\n")
			args := fakeAbcdOnPath(t, r)
			r.git("config", "--local", "abcd.sourcesBinary", value)
			blocked, out := commitNote(r, "note.md", "the widgetworks draft\n")
			if _, err := os.Stat(args); err == nil {
				t.Fatalf("an unusable opt-in fell back to the abcd on PATH\n%s", out)
			}
			if blocked || strings.Count(out, "abcd.sourcesBinary") != 1 || !strings.Contains(out, "absolute path") {
				t.Fatalf("an unusable opt-in is not one line naming the setting\n%s", out)
			}
		})
	}
}

// TestPreCommitHook_TemplateNamesABinaryWithoutTheVerbOnce: an opted-in binary that
// predates the source verb is ONE line naming the remedy — never a "refresh failed"
// warning, which on every commit would teach the committer to ignore it.
func TestPreCommitHook_TemplateNamesABinaryWithoutTheVerbOnce(t *testing.T) {
	hook := guardCopies(t)["template"]
	makeCorpusDir(t)
	r := newGuardRepo(t, hook, privateFormatDecl+"\nhand-key zzunrelated\n")
	old := refresher{sh: `case "$*" in *--help*) echo 'Usage: abcd [command]'; exit 0 ;; esac
echo 'abcd: unknown flag: --refresh' >&2; exit 1`}
	old.install(t, r, "template")
	blocked, out := commitNote(r, "note.md", "hello\n")
	if blocked {
		t.Fatalf("commit blocked\n%s", out)
	}
	if strings.Contains(out, "failed") || strings.Contains(out, "unknown flag") {
		t.Fatalf("a binary without the verb reads as a failure\n%s", out)
	}
	if strings.Count(out, "has no 'source sync-banlist --refresh'") != 1 {
		t.Fatalf("a binary without the verb is not one remedy line\n%s", out)
	}
}

// TestPreCommitHook_TheRefreshNeverCreatesTheStore: with a corpus and no private
// store in this working tree, neither copy runs its binary — creating the store is
// the by-hand sync's act — and each says so on one line.
func TestPreCommitHook_TheRefreshNeverCreatesTheStore(t *testing.T) {
	for name, hook := range guardCopies(t) {
		t.Run(name, func(t *testing.T) {
			makeCorpusDir(t)
			r := newGuardRepo(t, hook, "")
			args := writesEntry.install(t, r, name)
			blocked, out := commitNote(r, "note.md", "the widgetworks draft\n")
			if _, err := os.Stat(args); err == nil {
				t.Fatalf("the guard ran its refresh binary with no store\n%s", out)
			}
			if _, err := os.Stat(filepath.Join(r.dir, filepath.FromSlash(PrivateRelPath))); !os.IsNotExist(err) {
				t.Fatalf("the refresh created the private store (%v)\n%s", err, out)
			}
			if blocked || strings.Count(out, "never creates one") != 1 {
				t.Fatalf("the skip is not one line\n%s", out)
			}
		})
	}
}

// TestPreCommitHook_ALegacyStoreIsNamedOnce: a legacy store is not refreshed, the
// migration is named on the first commit that meets it, and not again — a notice on
// every commit is one the committer learns to skip. The binary never runs.
func TestPreCommitHook_ALegacyStoreIsNamedOnce(t *testing.T) {
	for name, hook := range guardCopies(t) {
		t.Run(name, func(t *testing.T) {
			makeCorpusDir(t)
			r := newGuardRepo(t, hook, "# legacy\nzzunrelated\n")
			args := writesEntry.install(t, r, name)
			blocked, out := commitNote(r, "one.md", "hello\n")
			if blocked || strings.Count(out, "abcd banlist migrate") != 1 {
				t.Fatalf("the first commit does not name the migration once\n%s", out)
			}
			blocked, out = commitNote(r, "two.md", "hello again\n")
			if blocked || strings.Contains(out, "abcd banlist migrate") {
				t.Fatalf("the second commit named the migration again\n%s", out)
			}
			if _, err := os.Stat(args); err == nil {
				t.Fatalf("the guard ran its refresh binary over a legacy store\n%s", out)
			}
		})
	}
}
