package ahoy

// One install, kept and replayed. A fixture that needs a repository abcd has
// already set up runs Install for it, about forty git processes before the test
// has begun to test anything, and over a hundred tests start that way. The first
// install in a given starting state runs for real and what it changed is kept;
// every later one starting from the same state is written back from the record,
// so the test meets the same files, modes and links in its own directories.
//
// The kept writes are safe to replay because the starting state is the key: the
// fixture's name, the environment, and every file under the directories an
// install reads and writes, each with the test's own paths taken out. A test
// that changes any of them first gets an install of its own. So does a test that
// has swapped a package seam, whose effect no fingerprint can see.
// TestAKeptInstallIsTheInstallItReplays holds the replay to a real install.

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/question"
)

// installTemplates holds the kept installs by starting state; replays counts
// the times one was written back, for the test that needs to know it was.
var installTemplates struct {
	mu      sync.Mutex
	byKey   map[string]*keptInstall
	replays int
}

// keptInstall is what one install changed, in tokenised form.
type keptInstall struct {
	writes  []keptEntry
	removes []keptEntry
}

// keptEntry is one file, link or folder under a root: root is the root's token
// and rel its slash-separated place there. body and link carry tokens for the
// test's own paths.
type keptEntry struct {
	root, rel string
	mode      fs.FileMode
	link      string
	body      []byte
}

// installRoot is a directory an install reads or writes, with the token that
// stands for its path in anything kept.
type installRoot struct {
	token, dir string
}

// templatedInstall makes repo, an empty directory the test owns, what build
// would make it: by running build the first time this fixture is asked for from
// this starting state, and by replaying what that run changed after that.
func templatedInstall(t *testing.T, repo, fixture string, build func()) {
	t.Helper()
	roots, ok := installRoots(repo)
	if !ok || !seamsAtDefault() {
		build()
		return
	}
	before := captureRoots(t, roots)
	key := installKey(t, fixture, roots, before)

	installTemplates.mu.Lock()
	defer installTemplates.mu.Unlock()
	if kept, ok := installTemplates.byKey[key]; ok {
		kept.replay(t, roots)
		installTemplates.replays++
		return
	}
	build()
	kept := diffInstall(before, captureRoots(t, roots))
	if installTemplates.byKey == nil {
		installTemplates.byKey = map[string]*keptInstall{}
	}
	installTemplates.byKey[key] = kept
}

// installRoots names the directories in force, longest first so that a path
// inside two is tokenised by the nearer. It reports false when one of them is
// not a temporary directory of the test's, which is no state to key on.
func installRoots(repo string) ([]installRoot, bool) {
	candidates := []installRoot{
		{"@REPO@", repo},
		{"@HOME@", os.Getenv("HOME")},
		{"@PLUGIN@", os.Getenv("ABCD_PLUGIN_ROOT")},
		{"@DATA@", os.Getenv("CLAUDE_PLUGIN_DATA")},
		{"@CONFIG@", os.Getenv("CLAUDE_CONFIG_DIR")},
	}
	if target := os.Getenv("ABCD_BIN_TARGET"); target != "" {
		candidates = append(candidates, installRoot{"@BIN@", filepath.Dir(filepath.Dir(target))})
	}
	var roots []installRoot
	for _, r := range candidates {
		if r.dir == "" {
			continue
		}
		if !testOwnedDir(r.dir) {
			return nil, false
		}
		// One root inside another is captured once, as part of the outer.
		nested := false
		for _, o := range candidates {
			if o.dir != "" && o.dir != r.dir && strings.HasPrefix(r.dir, o.dir+string(filepath.Separator)) {
				nested = true
			}
		}
		if !nested {
			roots = append(roots, r)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return len(roots[i].dir) > len(roots[j].dir) })
	return roots, true
}

// testOwnedDir reports whether dir is under the temporary area tests make their
// directories in.
func testOwnedDir(dir string) bool {
	tmp := strings.TrimRight(os.TempDir(), string(filepath.Separator))
	return strings.HasPrefix(dir, tmp+string(filepath.Separator))
}

// physicalDir is dir with its links resolved, which is how a path the code
// under test resolved comes out where the temporary area sits behind a link.
func physicalDir(dir string) string {
	if p, err := filepath.EvalSymlinks(dir); err == nil {
		return p
	}
	return dir
}

// setupDateInConfig is the day an install stamps in the config it writes.
var setupDateInConfig = regexp.MustCompile(`"setup_date": "[0-9]{4}-[0-9]{2}-[0-9]{2}"`)

// repoNameInConfig is the repository's name as an install stamps it in the
// config, which is the base name of the test's temporary directory.
func repoNameInConfig(name string) string { return `"project_name": ` + strconv.Quote(name) }

// tokenise takes out of b what is the test's own or the day's: the paths, the
// name the repository's directory gives it, and the date.
func tokenise(b []byte, roots []installRoot) []byte {
	s := string(b)
	for _, r := range roots {
		s = strings.ReplaceAll(s, physicalDir(r.dir), r.token+"#physical")
		s = strings.ReplaceAll(s, r.dir, r.token)
		if r.token == "@REPO@" {
			s = strings.ReplaceAll(s, repoNameInConfig(filepath.Base(r.dir)), repoNameInConfig("@REPO_NAME@"))
		}
	}
	return []byte(setupDateInConfig.ReplaceAllString(s, `"setup_date": "@TODAY@"`))
}

// untokenise puts this test's own into what tokenise took it out of.
func untokenise(b []byte, roots []installRoot) []byte {
	s := string(b)
	for _, r := range roots {
		s = strings.ReplaceAll(s, r.token+"#physical", physicalDir(r.dir))
		s = strings.ReplaceAll(s, r.token, r.dir)
		if r.token == "@REPO@" {
			s = strings.ReplaceAll(s, repoNameInConfig("@REPO_NAME@"), repoNameInConfig(filepath.Base(r.dir)))
		}
	}
	return []byte(strings.ReplaceAll(s, "@TODAY@", time.Now().UTC().Format("2006-01-02")))
}

// captureRoots reads every entry under the roots, tokenised.
func captureRoots(t *testing.T, roots []installRoot) map[string]keptEntry {
	t.Helper()
	entries := map[string]keptEntry{}
	for _, r := range roots {
		err := filepath.WalkDir(r.dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if p == r.dir {
				return nil
			}
			rel, err := filepath.Rel(r.dir, p)
			if err != nil {
				return err
			}
			fi, err := d.Info()
			if err != nil {
				return err
			}
			e := keptEntry{root: r.token, rel: filepath.ToSlash(rel), mode: fi.Mode()}
			switch {
			case fi.Mode()&os.ModeSymlink != 0:
				link, err := os.Readlink(p)
				if err != nil {
					return err
				}
				e.link = string(tokenise([]byte(link), roots))
			case fi.Mode().IsRegular():
				body, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				e.body = tokenise(body, roots)
			case !fi.IsDir():
				return fmt.Errorf("%s: no replay for mode %v", p, fi.Mode())
			}
			entries[e.root+"/"+e.rel] = e
			return nil
		})
		if err != nil {
			t.Fatalf("reading %s: %v", r.dir, err)
		}
	}
	return entries
}

// environmentInKey is the environment with the test's own paths taken out: every
// variable is state an install may read. A temporary directory named in a value
// that is no root (a fake tool's folder on PATH) stands for what it holds at its
// top level. A fake gh's own script is left out, since it names its test's log
// and no install runs gh, as the replay's test checks.
func environmentInKey(roots []installRoot) []string {
	tmp := strings.TrimRight(os.TempDir(), string(filepath.Separator))
	tempDir := regexp.MustCompile(regexp.QuoteMeta(tmp) + `/[^/:]+/[0-9]+`)
	var env []string
	for _, kv := range os.Environ() {
		kv = string(tokenise([]byte(kv), roots))
		kv = tempDir.ReplaceAllStringFunc(kv, heldBy)
		env = append(env, kv)
	}
	sort.Strings(env)
	return env
}

// heldBy names what dir holds at its top level, with each file's digest.
func heldBy(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "@TEMP?@"
	}
	var held []string
	for _, e := range entries {
		item := e.Name()
		if e.Type().IsRegular() && e.Name() != "gh" {
			if body, err := os.ReadFile(filepath.Join(dir, e.Name())); err == nil {
				item += fmt.Sprintf(":%x", sha256.Sum256(body))
			}
		}
		held = append(held, item)
	}
	return "@TEMP[" + strings.Join(held, ",") + "]@"
}

// installKey is the fingerprint of the starting state.
func installKey(t *testing.T, fixture string, roots []installRoot, before map[string]keptEntry) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00", fixture, cwd)
	for _, kv := range environmentInKey(roots) {
		fmt.Fprintf(h, "%s\x00", kv)
	}
	names := make([]string, 0, len(before))
	for name := range before {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		e := before[name]
		fmt.Fprintf(h, "%s\x00%v\x00%s\x00%x\x00", name, e.mode, e.link, sha256.Sum256(e.body))
	}
	return string(h.Sum(nil))
}

// diffInstall is what an install changed.
func diffInstall(before, after map[string]keptEntry) *keptInstall {
	kept := &keptInstall{}
	for name, e := range after {
		if b, ok := before[name]; !ok || b.mode != e.mode || b.link != e.link || string(b.body) != string(e.body) {
			kept.writes = append(kept.writes, e)
		}
	}
	for name, e := range before {
		if _, ok := after[name]; !ok {
			kept.removes = append(kept.removes, e)
		}
	}
	byPlace := func(a, b keptEntry) int {
		return strings.Compare(a.root+"/"+a.rel, b.root+"/"+b.rel)
	}
	slices.SortFunc(kept.writes, byPlace)
	slices.SortFunc(kept.removes, byPlace)
	return kept
}

// replay makes the kept changes under this test's roots: parents before what
// they hold, and each folder's own mode last, since a read-only one could
// otherwise not be filled.
func (k *keptInstall) replay(t *testing.T, roots []installRoot) {
	t.Helper()
	dirOf := map[string]string{}
	for _, r := range roots {
		dirOf[r.token] = r.dir
	}
	place := func(e keptEntry) string { return filepath.Join(dirOf[e.root], filepath.FromSlash(e.rel)) }
	for i := len(k.removes) - 1; i >= 0; i-- {
		if err := os.Remove(place(k.removes[i])); err != nil {
			t.Fatal(err)
		}
	}
	for _, e := range k.writes {
		p := place(e)
		var err error
		switch {
		case e.mode.IsDir():
			err = os.MkdirAll(p, 0o700)
		case e.mode&os.ModeSymlink != 0:
			_ = os.Remove(p)
			err = os.Symlink(string(untokenise([]byte(e.link), roots)), p)
		default:
			err = os.WriteFile(p, untokenise(e.body, roots), 0o600)
			if err == nil {
				err = os.Chmod(p, e.mode.Perm())
			}
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := len(k.writes) - 1; i >= 0; i-- {
		if e := k.writes[i]; e.mode.IsDir() {
			if err := os.Chmod(place(e), e.mode.Perm()); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// defaultInstallSeams is every package seam as TestMain left it.
var defaultInstallSeams []string

// seamsAtDefault reports whether every seam an install reaches is as TestMain
// left it. A test that swapped one wants an install through its own.
func seamsAtDefault() bool {
	return slices.Equal(installSeams(), defaultInstallSeams)
}

// installSeams identifies each seam by the code it points at, and each value a
// test may set by the value.
func installSeams() []string {
	funcs := []any{
		downloadTag, newToolInstaller, afterCacheBound, readRootAgents, beforeToolFileRemove,
		copyRelease, checkReleaseClaim, goRunModule, checkoutBuild, readClaudeVersion,
		caseFoldingFS, shellSettingsAsks, measureFits, entryTrust, writeHarnessSettings,
		shellEnv, shellApply, removeBackupCopy, gitInstalled, harnessInstalled, osExecutable,
		afterHistoryReloadHook, beforeHistoryIndexCreateHook, currentVintage,
		waitingRange, waitingDefaultChanges,
	}
	seams := make([]string, 0, len(funcs)+4)
	for _, f := range funcs {
		seams = append(seams, fmt.Sprint(reflect.ValueOf(f).Pointer()))
	}
	return append(seams, fmt.Sprint(claudeCodeAgentsFloor), hostVersionTimeout.String(),
		historyLockTimeout.String(), rewriteLockTimeout.String())
}

// TestAKeptInstallIsTheInstallItReplays: a fixture that replayed an install
// leaves exactly what a real install leaves, in the repository and in every
// directory the install reaches, byte for byte, mode for mode and link for link,
// with each test's own paths in place. It is what the fixtures' sharing rests on:
// were an install to come to depend on anything the key does not see, or to write
// outside the roots, this is the test that says so.
func TestAKeptInstallIsTheInstallItReplays(t *testing.T) {
	const origin = "https://github.com/example-org/example-repo"
	for _, tc := range []struct {
		name string
		make func(t *testing.T) string
		real func(t *testing.T, repo string)
	}{
		{"installed", installedRepo, func(t *testing.T, repo string) {
			idMustMkdir(t, filepath.Join(repo, ".git"))
			if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
				t.Fatal(err)
			}
		}},
		{"managed", func(t *testing.T) string { return managedRepoWithOrigin(t, origin) }, func(t *testing.T, repo string) {
			idMustGit(t, repo, "init")
			idMustGit(t, repo, "config", "user.name", "Alice Example")
			idMustGit(t, repo, "config", "user.email", "alice@example.com")
			idMustGit(t, repo, "remote", "add", "origin", origin)
			if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The first may be the one that keeps it; the second replays it.
			t.Run("keep", func(t *testing.T) {
				setupHermetic(t)
				tc.make(t)
			})
			var replayed map[string]keptEntry
			t.Run("replay", func(t *testing.T) {
				setupHermetic(t)
				// Another numbered directory from the first, so anything an
				// install takes from the name of the directory shows below.
				t.TempDir()
				installTemplates.mu.Lock()
				was := installTemplates.replays
				installTemplates.mu.Unlock()
				repo := tc.make(t)
				installTemplates.mu.Lock()
				now := installTemplates.replays
				installTemplates.mu.Unlock()
				if now != was+1 {
					t.Fatalf("the second fixture replayed nothing (%d replays, then %d)", was, now)
				}
				if name := readConfigT(t, repo)["meta"].(map[string]any)["project_name"]; name != filepath.Base(repo) {
					t.Errorf("project_name = %v, want this repository's name %s", name, filepath.Base(repo))
				}
				if date := readConfigT(t, repo)["meta"].(map[string]any)["setup_date"]; date != time.Now().UTC().Format("2006-01-02") {
					t.Errorf("setup_date = %v, want today", date)
				}
				roots, _ := installRoots(repo)
				replayed = captureRoots(t, roots)
			})
			t.Run("real", func(t *testing.T) {
				setupHermetic(t)
				repo := t.TempDir()
				tc.real(t, repo)
				roots, _ := installRoots(repo)
				real := captureRoots(t, roots)
				for name, want := range real {
					got, ok := replayed[name]
					switch {
					case !ok:
						t.Errorf("%s: a real install writes it and the replay does not", name)
					case got.mode != want.mode:
						t.Errorf("%s: mode %v replayed, %v real", name, got.mode, want.mode)
					case got.link != want.link:
						t.Errorf("%s: link %q replayed, %q real", name, got.link, want.link)
					case string(got.body) != string(want.body):
						t.Errorf("%s differs:\nreplayed:\n%s\nreal:\n%s", name, got.body, want.body)
					}
				}
				for name := range replayed {
					if _, ok := real[name]; !ok {
						t.Errorf("%s: the replay writes it and a real install does not", name)
					}
				}
			})
		})
	}
}

// TestAStateTheKeyDoesNotKnowGetsItsOwnInstall: an environment or file changed
// before the fixture, or a swapped seam, is a real install and never a replay of
// one kept from another starting state.
func TestAStateTheKeyDoesNotKnowGetsItsOwnInstall(t *testing.T) {
	replays := func() int {
		installTemplates.mu.Lock()
		defer installTemplates.mu.Unlock()
		return installTemplates.replays
	}
	fresh := func(t *testing.T) {
		t.Helper()
		setupHermetic(t)
		installedRepo(t)
	}
	t.Run("keep", fresh)
	unique := strconv.FormatInt(time.Now().UnixNano(), 10)
	for name, change := range map[string]func(t *testing.T){
		// Each with a value no earlier run in this process kept (-count).
		"a variable": func(t *testing.T) { t.Setenv("ABCD_TEMPLATE_TEST_VARIABLE", unique) },
		"a file in the home": func(t *testing.T) {
			writeFixture(t, os.Getenv("HOME"), "a-file-of-the-tests", unique, 0o644)
		},
		"a seam": func(t *testing.T) {
			swap := measureFits
			t.Cleanup(func() { measureFits = swap })
			measureFits = func(h PromptHelp, l question.Limits, def string) bool { return swap(h, l, def) }
		},
	} {
		t.Run(name, func(t *testing.T) {
			setupHermetic(t)
			change(t)
			was := replays()
			installedRepo(t)
			if replays() != was {
				t.Errorf("the fixture replayed an install kept from another starting state")
			}
		})
	}
	t.Run("unchanged replays", func(t *testing.T) {
		was := replays()
		fresh(t)
		if replays() != was+1 {
			t.Errorf("an unchanged state did not replay")
		}
	})
}

// TestNoInstallRunsGh is what lets the key leave a fake gh's own script out: it
// names its test's log, and a replay must lose no call from that log.
func TestNoInstallRunsGh(t *testing.T) {
	setupHermetic(t)
	logPath := ghFake(t, bothDisabled)
	repo := t.TempDir()
	idMustGit(t, repo, "init")
	idMustGit(t, repo, "remote", "add", "origin", "https://github.com/example-org/example-repo")
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if raw, err := os.ReadFile(logPath); err == nil && len(raw) != 0 {
		t.Errorf("an install ran gh:\n%s", raw)
	}
}

func idMustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}
