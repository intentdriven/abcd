package site

// A project site shows nothing read from git history: no author, no model a
// commit declared, no commit count and no commit date. Git history is not
// something abcd writes to a repository, and a project site renders only what
// abcd writes (project.go). The composed route, abcd's own site, keeps all of
// it.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The project history fixture's distinctive git facts: an author, the same
// person under a second name (a same-author candidate pair on the health
// page), a declared model, and two commit dates nothing else in the fixture
// carries. None of them may reach a project site.
const (
	historyAuthor      = "Zebulon Quixwright"
	historyAuthorAlias = "zquix"
	historyNoreply     = "4242+zquix@users.noreply.github.com"
	historyModel       = "Quixvendor:quix-model-9"
	historyDateOlder   = "2031-07-19"
	historyDateNewer   = "2031-07-20"
)

// Two met shipped intents with timestamped ids. The OLDER id is shipped in the
// NEWER commit, so an order by the date each entered shipped/ and an order by
// id disagree.
const (
	newerIDIntent = "itd-2602010900002222"
	olderIDIntent = "itd-2601010900001111"
	newerIDQuote  = "The newer id is quoted first."
	olderIDQuote  = "The older id is quoted second."
)

// metIntentBody is a shipped intent whose audit was met and whose press
// release is written.
func metIntentBody(id, slug, title, quote string) string {
	return "---\nid: " + id + "\nslug: " + slug + "\nspec_id: null\nkind: standalone\nbuilds_on: []\n" +
		"severity: minor\nimpact: additive\n---\n\n# " + title + "\n\n## Press Release\n\n> **" + quote + "**\n\n" +
		"## Acceptance Criteria\n\n- Given the intent, then it is met.\n\n## Audit Notes\n\n" +
		"Acceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0\n"
}

// commitAs stages everything and commits it under a named author at a fixed
// date, with an attribution trailer.
func (f *fixture) commitAs(date, name, email, subject, assistedBy string) {
	f.t.Helper()
	f.git(date, "add", "-A")
	cmd := exec.Command("git", "-C", f.root, "-c", "commit.gpgsign=false",
		"commit", "--allow-empty", "-m", subject+"\n\nAssisted-by: "+assistedBy)
	cmd.Env = append(gittest.Env(f.t),
		"GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date,
		"GIT_AUTHOR_NAME="+name, "GIT_AUTHOR_EMAIL="+email,
		"GIT_COMMITTER_NAME="+name, "GIT_COMMITTER_EMAIL="+email)
	if out, err := cmd.CombinedOutput(); err != nil {
		f.t.Fatalf("git commit: %v\n%s", err, out)
	}
}

// newProjectHistoryFixture is the project fixture with the distinctive history
// above: two met intents with timestamped ids, each written and shipped by the
// fixture's distinctive author on a date nothing else carries.
func newProjectHistoryFixture(t *testing.T) *fixture {
	t.Helper()
	f := newProjectFixture(t)
	shipped := ".abcd/development/intents/shipped/"
	f.write(shipped+newerIDIntent+"-newer-id.md", metIntentBody(newerIDIntent, "newer-id", "The Newer Id", newerIDQuote))
	f.commitAs(historyDateOlder+"T09:00:00+00:00", historyAuthor, historyNoreply, "feat: ship the newer id", historyModel)
	f.write(shipped+olderIDIntent+"-older-id.md", metIntentBody(olderIDIntent, "older-id", "The Older Id", olderIDQuote))
	f.commitAs(historyDateNewer+"T09:00:00+00:00", historyAuthorAlias, "zquix@example.invalid", "feat: ship the older id", historyModel)
	return f
}

// Every page and record.json on the project route are free of git history:
// no author name, no declared model, no commit date, and no record.json key
// that carries one.
func TestAProjectSiteShowsNothingFromGitHistory(t *testing.T) {
	f := newProjectHistoryFixture(t)
	out, files := buildProject(t, f)
	if len(files) == 0 {
		t.Fatal("the project build wrote nothing")
	}
	for rel, data := range files {
		for _, word := range []string{historyAuthor, historyAuthorAlias, "quix-model-9", "Quixvendor", historyDateOlder, historyDateNewer} {
			if strings.Contains(string(data), word) {
				t.Errorf("%s carries %q, which is read from git history", rel, word)
			}
		}
	}

	var export map[string]json.RawMessage
	if err := json.Unmarshal(files["record.json"], &export); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"authorship", "history"} {
		if _, ok := export[key]; ok {
			t.Errorf("record.json carries %q, which is read from git history", key)
		}
	}
	var nodes struct {
		Nodes []map[string]json.RawMessage `json:"nodes"`
	}
	if err := json.Unmarshal(files["record.json"], &nodes); err != nil {
		t.Fatal(err)
	}
	for _, n := range nodes.Nodes {
		if _, ok := n["dates"]; ok {
			t.Errorf("record.json node %s carries git dates", n["id"])
			break
		}
	}

	// The health page keeps the checks the record makes of itself and drops
	// the ones git history answers; the timeline, drawn mostly from git dates,
	// is left out with its link.
	ui, err := projectUI()
	if err != nil {
		t.Fatal(err)
	}
	health := string(files[routeHealth+"index.html"])
	if health == "" {
		t.Fatal("the project site has no health page")
	}
	for _, label := range []string{ui.Health.SameAuthor, ui.Health.Undeclared, ui.Health.MultiTrailer, ui.Contributors.Assisted} {
		if strings.Contains(health, escapeText(label)) {
			t.Errorf("the health page carries %q, a check git history answers", label)
		}
	}
	for _, label := range []string{ui.Health.Unresolved, ui.Health.Isolated} {
		if !strings.Contains(health, escapeText(label)) {
			t.Errorf("the health page lost %q, a check the record makes of itself", label)
		}
	}
	for rel, data := range files {
		if strings.Contains(string(data), `class="tlsvg"`) || strings.Contains(string(data), ">"+escapeText(ui.RecordNav.Timeline)+"<") {
			t.Errorf("%s carries the timeline, which is drawn from git dates", rel)
		}
	}

	res, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: out, Project: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK() {
		for _, fd := range res.Findings {
			t.Errorf("finding: %+v", fd)
		}
	}
}

// The about page quotes the newest shipped intents in id order: a timestamped
// id `itd-<yymmddHHMMSS><rrrr>` carries when it was minted, and an ordinal id
// sorts before every timestamped one. The date each entered shipped/ is git's,
// and plays no part.
func TestTheAboutPageOrdersShippedIntentsByID(t *testing.T) {
	_, files := buildProject(t, newProjectHistoryFixture(t))
	about := string(files["index.html"])
	order := []string{newerIDQuote, olderIDQuote, "A fixture user opens the page and sees the record."}
	last := -1
	for _, q := range order {
		i := strings.Index(about, q)
		if i < 0 {
			t.Fatalf("the about page does not quote %q", q)
		}
		if i < last {
			t.Errorf("the about page quotes %q out of id order; want %q", q, order)
		}
		last = i
	}
}

// gitLogWrapper puts a git on PATH that records each invocation's arguments
// and then runs the real one, and returns the log's path.
func gitLogWrapper(t *testing.T) string {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git on PATH")
	}
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '" + logPath + "'\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

// historyCalls are the logged git invocations that read history: a log, a
// shortlog or a blame.
func historyCalls(t *testing.T, logPath string) []string {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		for _, arg := range strings.Fields(line) {
			if arg == "log" || arg == "shortlog" || arg == "blame" || arg == "whatchanged" {
				out = append(out, line)
				break
			}
		}
	}
	return out
}

// A project build and its gate never ask git for history. The composed build,
// run under the same wrapper, does, which is what shows the wrapper sees the
// calls at all.
func TestAProjectBuildNeverReadsGitHistory(t *testing.T) {
	f := newProjectHistoryFixture(t)
	composed := newFixture(t)
	logPath := gitLogWrapper(t)

	out := t.TempDir()
	if _, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp, Project: true}); err != nil {
		t.Fatalf("project build: %v", err)
	}
	if _, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: out, Project: true}); err != nil {
		t.Fatalf("project gate: %v", err)
	}
	if calls := historyCalls(t, logPath); len(calls) > 0 {
		t.Errorf("a project build read git history:\n%s", strings.Join(calls, "\n"))
	}

	if _, err := Build(Request{RepoRoot: composed.Root(), OutDir: t.TempDir(), Stamp: fixtureStamp}); err != nil {
		t.Fatalf("composed build: %v", err)
	}
	if len(historyCalls(t, logPath)) == 0 {
		t.Error("the composed build read no git history through the wrapper; the wrapper sees nothing")
	}
}

// The composed route keeps every git-history detail: the authorship and
// history blocks and each record's dates in record.json, the timeline, and the
// health page's authorship checks.
func TestTheComposedRouteKeepsItsGitHistoryDetails(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)
	files := siteFiles(t, out)
	var export map[string]json.RawMessage
	if err := json.Unmarshal(files["record.json"], &export); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"authorship", "history"} {
		if _, ok := export[key]; !ok {
			t.Errorf("the composed record.json lost %q", key)
		}
	}
	if !strings.Contains(string(files["record.json"]), `"dates": {`) {
		t.Error("the composed record.json lost the records' dates")
	}
	if !strings.Contains(string(files[routeDashboard+"index.html"]), `class="tlsvg"`) {
		t.Error("the composed dashboard lost its timeline")
	}
	ui, err := LoadUI(f.Root(), "site-src/ui.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(files[routeHealth+"index.html"]), escapeText(ui.Health.SameAuthor)) {
		t.Error("the composed health page lost its same-author panel")
	}
}
