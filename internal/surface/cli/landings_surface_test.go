package cli

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/landings"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The `abcd landings` front door (spc-2610301950480609 step 4,
// itd-2610301850454860): its range and refusals, its text and --json, its
// filters, the positional trace, what it prints of text someone typed, and
// that it fetches and pushes nothing. The reader's classes, forge read and
// revert trace are proved in internal/core/landings; these tests prove the
// verb drives them and prints what they say. No test here asks a live forge:
// each hands the verb landingsFake.

// landingsPR is what landingsFake answers for one pull request: the commit
// it merged as, its head, and the login of an approving review of that head,
// "" for none.
type landingsPR struct {
	merge, head, approvedBy string
}

// landingsFake is a forge answering the reader's GraphQL documents from prs:
// each aliased node p<N> is answered from prs[N], or null for a number it
// does not hold; a body document (one asking lastEditedAt) gets an empty,
// unedited body. fail fails every call with that text. Every call's arguments
// are kept.
type landingsFake struct {
	prs   map[int]landingsPR
	fail  string
	calls [][]string
}

var landingsAliasRe = regexp.MustCompile(`p([0-9]+):pullRequest\(`)

func (f *landingsFake) call(args ...string) (string, error) {
	f.calls = append(f.calls, args)
	if f.fail != "" {
		return "", errors.New(f.fail)
	}
	query := ""
	for _, a := range args {
		if q, ok := strings.CutPrefix(a, "query="); ok {
			query = q
		}
	}
	body := strings.Contains(query, "lastEditedAt")
	at := "2026-11-01T10:00:00Z"
	nodes := map[string]any{}
	for _, m := range landingsAliasRe.FindAllStringSubmatch(query, -1) {
		n, _ := strconv.Atoi(m[1])
		pr, ok := f.prs[n]
		if !ok {
			nodes["p"+m[1]] = nil
			continue
		}
		if body {
			nodes["p"+m[1]] = map[string]any{"number": n, "mergeCommit": map[string]any{"oid": pr.merge}, "mergedAt": at,
				"lastEditedAt": nil, "body": "", "baseRepository": map[string]any{"nameWithOwner": "org/repo"}}
			continue
		}
		reviews := []any{}
		if pr.approvedBy != "" {
			reviews = append(reviews, map[string]any{"author": map[string]any{"__typename": "User", "login": pr.approvedBy},
				"authorAssociation": "MEMBER", "authorCanPushToRepository": true, "state": "APPROVED",
				"commit": map[string]any{"oid": pr.head}, "submittedAt": "2026-11-01T09:00:00Z"})
		}
		nodes["p"+m[1]] = map[string]any{"number": n, "author": map[string]any{"login": "pr-author"}, "headRefOid": pr.head,
			"mergedAt": at, "mergeCommit": map[string]any{"oid": pr.merge},
			"reviews": map[string]any{"totalCount": len(reviews), "nodes": reviews},
			"commits": map[string]any{"totalCount": 0, "nodes": []any{}}}
	}
	out, err := json.Marshal(map[string]any{"data": map[string]any{"repository": nodes}})
	return string(out), err
}

// landingsWorld is a default branch built by hand in a checkout the test
// stands in: merges of pull requests and direct pushes over a root commit
// that sets the facilitator mode, with the forge the verb is handed.
type landingsWorld struct {
	t     *testing.T
	r     *gittest.Repo
	forge *landingsFake
}

func newLandingsWorld(t *testing.T, mode string) *landingsWorld {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(".gitignore", ".abcd/.work.local/\n")
	r.Write(".abcd/.work.local/.keep", "")
	r.Write(".abcd/config.json", `{"facilitator":{"mode":"`+mode+`"}}`+"\n")
	r.Commit("chore: root")
	w := &landingsWorld{t: t, r: r, forge: &landingsFake{prs: map[int]landingsPR{}}}
	prev := landingsForge
	landingsForge = w.forge.call
	t.Cleanup(func() { landingsForge = prev })
	t.Chdir(r.Root())
	return w
}

func (w *landingsWorld) head() string { return w.r.Git("rev-parse", "HEAD") }

// push makes one commit on main with msg, as a person's push.
func (w *landingsWorld) push(msg string) string {
	w.t.Helper()
	w.r.Write("main/"+strconv.Itoa(len(w.r.Git("rev-list", "--all")))+".txt", msg)
	w.r.Commit(msg)
	return w.head()
}

// merge merges a branch of msgs into main as the forge's merge of pull
// request pr, which the forge answers for, approved by approvedBy where it is
// not ""; it returns the merge and the branch's commits.
func (w *landingsWorld) merge(pr int, approvedBy string, msgs ...string) (string, []string) {
	w.t.Helper()
	name := "topic-" + strconv.Itoa(pr)
	w.r.Git("checkout", "-q", "-b", name, "main")
	var commits []string
	for i, m := range msgs {
		w.r.Write(name+"/"+strconv.Itoa(i), m)
		w.r.Commit(m)
		commits = append(commits, w.head())
	}
	tip := w.head()
	w.r.Git("checkout", "-q", "main")
	w.r.Git("merge", "-q", "--no-ff", "-m", fmt.Sprintf("Merge pull request #%d from org/%s", pr, name), name)
	m := w.head()
	w.forge.prs[pr] = landingsPR{merge: m, head: tip, approvedBy: approvedBy}
	return m, commits
}

// runLandings drives the verb, mapping its error to an exit code as Run does.
func runLandings(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	root := NewRootCommand()
	root.SetArgs(append([]string{"landings"}, args...))
	var so, se bytes.Buffer
	root.SetOut(&so)
	root.SetErr(&se)
	err := root.Execute()
	if err == nil {
		return so.String(), se.String(), 0
	}
	code = 1
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		code = coded.ExitCode()
	}
	return so.String(), se.String() + err.Error(), code
}

// landingsJSON runs the verb with --json and args, and reads its listing.
func landingsJSON(t *testing.T, args ...string) landingsOutput {
	t.Helper()
	out, errOut, code := runLandings(t, append([]string{"--json"}, args...)...)
	if code != 0 {
		t.Fatalf("landings --json %v exited %d: %s", args, code, errOut)
	}
	var l landingsOutput
	if err := json.Unmarshal([]byte(out), &l); err != nil {
		t.Fatalf("landings --json %v: %v\n%s", args, err, out)
	}
	return l
}

func landingsCommits(rows []landings.Row) []string {
	var out []string
	for _, r := range rows {
		out = append(out, r.Commit)
	}
	return out
}

// TestLandingsDefaultRangeIsTheLastReleaseTag: with no --since the range
// opens after the last release tag the tip reaches, and with none the verb
// refuses at exit 2, naming the flag and listing nothing.
func TestLandingsDefaultRangeIsTheLastReleaseTag(t *testing.T) {
	w := newLandingsWorld(t, "auto")
	w.push("docs: before")
	out, errOut, code := runLandings(t)
	if code != 2 || !strings.Contains(errOut, "--since") || out != "" {
		t.Fatalf("no tag: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	tagged := w.push("chore: release")
	w.r.Git("tag", "v0.1.0", tagged)
	m, _ := w.merge(1, "", "feat: one")
	p := w.push("docs: after")
	l := landingsJSON(t)
	if got := landingsCommits(l.Rows); !slices.Equal(got, []string{p, m}) || l.Since != "v0.1.0" || l.From != tagged || l.To != p {
		t.Fatalf("default range: rows %v since %q from %q to %q", got, l.Since, l.From, l.To)
	}
}

// TestLandingsSinceTakesATagOrADate: --since names a tag or a date, and one
// naming neither is refused at exit 2.
func TestLandingsSinceTakesATagOrADate(t *testing.T) {
	w := newLandingsWorld(t, "auto")
	var c []string
	for _, d := range []string{"2030-10-01T12:00:00Z", "2030-10-02T12:00:00Z", "2030-10-03T12:00:00Z"} {
		w.r.Write("dated/"+d, d)
		w.r.Git("add", "-A")
		cmd := exec.Command("git", "-C", w.r.Root(), "-c", "user.email=d@example.invalid", "-c", "user.name=Dated", "-c", "commit.gpgsign=false",
			"commit", "-q", "-m", "chore: "+d)
		cmd.Env = append(w.r.Env(), "GIT_COMMITTER_DATE="+d, "GIT_AUTHOR_DATE="+d)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v %s", err, out)
		}
		c = append(c, w.head())
	}
	w.r.Git("tag", "base", c[0])
	if got := landingsCommits(landingsJSON(t, "--since", "base").Rows); !slices.Equal(got, []string{c[2], c[1]}) {
		t.Fatalf("--since base: %v", got)
	}
	if l := landingsJSON(t, "--since", "2030-10-03"); !slices.Equal(landingsCommits(l.Rows), []string{c[2]}) || l.Since != "2030-10-03" {
		t.Fatalf("--since 2030-10-03: %v since %q", landingsCommits(l.Rows), l.Since)
	}
	if _, errOut, code := runLandings(t, "--since", "no-such-tag"); code != 2 || !strings.Contains(errOut, "neither a tag") {
		t.Fatalf("--since no-such-tag: exit %d, %q", code, errOut)
	}
}

// landingsTextRow is the commit a text line begins with, "" for a line that
// is no row.
var landingsTextRow = regexp.MustCompile(`^ *([0-9a-f]{12})  `)

// TestLandingsTextAndJSONHoldTheSameRows: the text prints one line per row,
// in the listing's order, then the notes and the tip read; --json holds the
// same rows, with the reader's stable keys.
func TestLandingsTextAndJSONHoldTheSameRows(t *testing.T) {
	w := newLandingsWorld(t, "auto")
	base := w.head()
	m1, _ := w.merge(1, "alice-reviewer", "feat: one\n\nResolves: iss-1")
	m2, _ := w.merge(2, "", "feat: two")
	p := w.push("docs: pushed")
	w.r.Git("tag", "start", base)
	text, _, code := runLandings(t, "--since", "start")
	if code != 0 {
		t.Fatalf("landings: exit %d\n%s", code, text)
	}
	var fromText []string
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	for _, l := range lines {
		if m := landingsTextRow.FindStringSubmatch(l); m != nil {
			fromText = append(fromText, m[1])
		}
	}
	l := landingsJSON(t, "--since", "start")
	var fromJSON []string
	for _, c := range landingsCommits(l.Rows) {
		fromJSON = append(fromJSON, c[:12])
	}
	if !slices.Equal(fromText, fromJSON) || !slices.Equal(landingsCommits(l.Rows), []string{p, m2, m1}) {
		t.Fatalf("text rows %v, json rows %v\n%s", fromText, fromJSON, text)
	}
	approved := regexp.MustCompile(`(?m)^` + m1[:12] + `  .*$`).FindString(text)
	if approved == "" || strings.Contains(approved, "unattended") {
		t.Fatalf("the person-approved row's line is missing or carries the unattended mark:\n%s", text)
	}
	if !strings.Contains(text, "#1  person-approved  facilitator.mode: auto") || !strings.Contains(text, "approval forge: alice-reviewer, 2026-11-01T09:00:00Z, seen  iss-1") ||
		!strings.Contains(text, "#2  automatic  facilitator.mode: auto  lane unknown  approval none seen") {
		t.Fatalf("the rows' text:\n%s", text)
	}
	if last := lines[len(lines)-1]; last != "read main at "+p[:12]+", since start (after "+base[:12]+")" {
		t.Fatalf("the tip line: %q", last)
	}
	raw, _, _ := runLandings(t, "--json", "--since", "start")
	var keys map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"tip", "from", "to", "since", "rows"} {
		if _, ok := keys[k]; !ok {
			t.Fatalf("--json lacks %q: %s", k, raw)
		}
	}
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(keys["rows"], &rows); err != nil || len(rows) != 3 {
		t.Fatalf("rows: %v %s", err, keys["rows"])
	}
	for _, k := range []string{"commit", "when", "class", "rule", "approval", "mode", "unattended"} {
		if _, ok := rows[1][k]; !ok {
			t.Fatalf("a row lacks %q: %s", k, keys["rows"])
		}
	}
}

// TestLandingsUnattendedKeepsExactlyTheUnattendedRows: --unattended keeps
// every row the listing marks unattended and no other, in text and JSON.
func TestLandingsUnattendedKeepsExactlyTheUnattendedRows(t *testing.T) {
	w := newLandingsWorld(t, "auto")
	w.r.Git("tag", "start", w.head())
	m1, _ := w.merge(1, "alice-reviewer", "feat: one")
	m2, _ := w.merge(2, "", "feat: two")
	p := w.push("docs: pushed")
	all := landingsJSON(t, "--since", "start")
	var want []string
	for _, r := range all.Rows {
		if r.Unattended {
			want = append(want, r.Commit)
		}
	}
	if !slices.Equal(want, []string{p, m2}) {
		t.Fatalf("the fixture's unattended rows are %v, want the push and #2", want)
	}
	if got := landingsCommits(landingsJSON(t, "--since", "start", "--unattended").Rows); !slices.Equal(got, want) {
		t.Fatalf("--unattended --json: %v, want %v", got, want)
	}
	text, _, _ := runLandings(t, "--since", "start", "--unattended")
	var rows []string
	for _, l := range strings.Split(text, "\n") {
		if m := landingsTextRow.FindStringSubmatch(l); m != nil {
			rows = append(rows, m[1])
		}
	}
	if !slices.Equal(rows, []string{p[:12], m2[:12]}) || m1 == m2 {
		t.Fatalf("--unattended text rows %v:\n%s", rows, text)
	}
}

// TestLandingsRevertedRendersTheGroups: --reverted lists the reverted rows
// under the rule that admitted them, with a count, and marks each with the
// commit that reverts it.
func TestLandingsRevertedRendersTheGroups(t *testing.T) {
	w := newLandingsWorld(t, "auto")
	w.r.Git("tag", "start", w.head())
	m1, branch := w.merge(1, "", "feat: one")
	w.merge(2, "", "feat: two")
	w.r.Git("revert", "--no-edit", branch[0])
	rv := w.head()
	l := landingsJSON(t, "--since", "start", "--reverted")
	if len(l.Groups) != 1 || l.Groups[0].Rule != "facilitator.mode: auto" || l.Groups[0].Count != 1 ||
		l.Groups[0].Rows[0].Commit != m1 || l.Groups[0].Rows[0].Reverted == nil || l.Groups[0].Rows[0].Reverted.By != rv {
		t.Fatalf("--reverted groups: %+v", l.Groups)
	}
	text, _, code := runLandings(t, "--since", "start", "--reverted")
	if code != 0 || !strings.HasPrefix(text, "facilitator.mode: auto  1\n  "+m1[:12]+"  #1  automatic") ||
		!strings.Contains(text, "reverted by "+rv[:12]) {
		t.Fatalf("--reverted text (exit %d):\n%s", code, text)
	}
}

// TestLandingsTracesOneCommit: one commit named prints the row that brought
// it onto the default branch, however far back; a commit the tip does not
// reach is said to be off the default branch at exit 0; a name that names no
// commit, or a trace given a listing's flag, is refused at exit 2.
func TestLandingsTracesOneCommit(t *testing.T) {
	w := newLandingsWorld(t, "auto")
	m1, branch := w.merge(1, "", "feat: one", "feat: one more")
	w.r.Git("tag", "v0.1.0", w.head())
	w.push("docs: after the tag")
	text, errOut, code := runLandings(t, branch[0])
	if code != 0 || !strings.HasPrefix(text, branch[0][:12]+" landed in:\n  "+m1[:12]+"  #1  automatic") {
		t.Fatalf("trace of a branch commit (exit %d, %s):\n%s", code, errOut, text)
	}
	var tr landings.Trace
	raw, _, _ := runLandings(t, "--json", branch[1])
	if err := json.Unmarshal([]byte(raw), &tr); err != nil || tr.Row == nil || tr.Row.Commit != m1 || tr.Commit != branch[1] {
		t.Fatalf("trace --json: %+v, %v\n%s", tr, err, raw)
	}
	w.r.Git("checkout", "-q", "-b", "elsewhere")
	off := w.push("feat: never merged")
	w.r.Git("checkout", "-q", "main")
	text, _, code = runLandings(t, off)
	if code != 0 || text != off[:12]+": "+landings.SaysNotOnDefault+"\nread main at "+w.head()[:12]+"\n" {
		t.Fatalf("trace off the default branch (exit %d):\n%q", code, text)
	}
	if _, errOut, code := runLandings(t, "notacommit"); code != 2 || !strings.Contains(errOut, "no object name") {
		t.Fatalf("a name that is no commit: exit %d, %q", code, errOut)
	}
	if _, _, code := runLandings(t, "--unattended", m1); code != 2 {
		t.Fatalf("a trace with --unattended: exit %d", code)
	}
	if _, _, code := runLandings(t, m1, branch[0]); code != 2 {
		t.Fatalf("two commits: exit %d", code)
	}
}

// TestLandingsPrintsTypedTextMadeSafe: what a row and a note print of text
// someone typed (a person's push's committer, the forge's own failure text)
// reaches the terminal with its control characters masked, and no line it
// prints runs past the render's bound.
func TestLandingsPrintsTypedTextMadeSafe(t *testing.T) {
	w := newLandingsWorld(t, "auto")
	w.r.Git("tag", "start", w.head())
	w.r.Write("pushed.txt", "x")
	w.r.Git("add", "-A")
	cmd := exec.Command("git", "-C", w.r.Root(), "-c", "user.email=p@example.invalid", "-c", "user.name=Pat \x1b[31mPusher\u009b2J\u202e",
		"-c", "commit.gpgsign=false", "commit", "-q", "-m", "fix: pushed \x1b[2J")
	cmd.Env = w.r.Env()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	w.merge(1, "", "feat: one")
	w.forge.fail = "gh: \x1b[2Jboom\u009b " + strings.Repeat("x", 5000)
	text, _, code := runLandings(t, "--since", "start")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, text)
	}
	for _, bad := range []string{"\x1b", "\u009b", "\u202e"} {
		if strings.Contains(text, bad) {
			t.Fatalf("the text carries %q raw:\n%q", bad, text)
		}
	}
	if !regexp.MustCompile(`(?m)^[0-9a-f]{12}  -  person-push  committer Pat \?\[31mPusher\?2J\?  `).MatchString(text) ||
		strings.Contains(text, "pushed by") || !strings.Contains(text, "\nforge: ") {
		t.Fatalf("the masked committer or the forge note is missing:\n%q", text)
	}
	for _, l := range strings.Split(text, "\n") {
		if len(l) > 2*landingsNoteBytes {
			t.Fatalf("a line of %d bytes: %q", len(l), l)
		}
	}
}

// landingsGit puts a git on PATH for the rest of t that appends each call's
// arguments to log (when log is not "") and fails, as a read git could not
// make, every call one of whose arguments is one of fail; every other call
// runs the real git.
func landingsGit(t *testing.T, log string, fail ...string) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	script := "#!/bin/sh\n"
	if log != "" {
		script += "{ printf '%s' \"$*\" | tr '\\n' ' '; echo; } >> '" + log + "'\n"
	}
	script += "for a in \"$@\"; do\n  case \"$a\" in\n"
	for _, f := range fail {
		script += "    " + f + ") echo 'fatal: a read that failed' >&2; exit 128;;\n"
	}
	script += "  esac\ndone\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// landingsTreeHash is a hash of every file under dir (its path, mode and
// bytes) but those under a path in skip, relative to dir, and the paths it
// read.
func landingsTreeHash(t *testing.T, dir string, skip ...string) (string, []string) {
	t.Helper()
	h := sha256.New()
	var paths []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if slices.Contains(skip, filepath.ToSlash(rel)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s %v\n", rel, info.Mode())
		if d.Type().IsRegular() {
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			h.Write(b)
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil)), paths
}

// TestLandingsFetchesNothingAndWritesOnlyItsCache: with an origin that points
// nowhere and the default branch read from its remote-tracking ref, the
// listing and the trace make no git call that fetches or pushes, ask the
// forge only GraphQL documents, and change nothing in the git directory, in
// the local tier outside the forge cache, or in the working tree. Nothing is
// left out of the git directory's hash, the index included: every call the
// verb makes reads, and none refreshes the index or writes a ref, a log or
// an object.
func TestLandingsFetchesNothingAndWritesOnlyItsCache(t *testing.T) {
	w := newLandingsWorld(t, "auto")
	w.r.Git("tag", "start", w.head())
	_, branch := w.merge(1, "", "feat: one")
	w.push("docs: pushed")
	w.r.Git("remote", "add", "origin", filepath.Join(t.TempDir(), "no-such-remote.git"))
	w.r.Git("update-ref", "refs/remotes/origin/main", "main")
	w.r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	// The listing must be read from origin's ref, not the local branch.
	w.r.Git("checkout", "-q", "-b", "local-only")
	w.push("docs: never fetched by origin")
	gitDir := filepath.Join(w.r.Root(), ".git")
	tier := filepath.Join(w.r.Root(), ".abcd", ".work.local")
	// The test's own status runs first: it may refresh the index, and the
	// verb runs no command that does.
	before := w.r.Git("status", "--porcelain", "--ignored", "--untracked-files=all")
	gitBefore, _ := landingsTreeHash(t, gitDir)
	tierBefore, _ := landingsTreeHash(t, tier, "landings")
	log := filepath.Join(t.TempDir(), "calls")
	landingsGit(t, log)
	for _, args := range [][]string{{"--since", "start"}, {"--json", "--since", "start", "--reverted"}, {branch[0]}} {
		out, errOut, code := runLandings(t, args...)
		if code != 0 || !strings.Contains(out, "origin/main") {
			t.Fatalf("landings %v: exit %d, %s %s", args, code, out, errOut)
		}
	}
	raw, err := os.ReadFile(log)
	if err != nil || len(raw) == 0 {
		t.Fatalf("no git call was counted: %v", err)
	}
	words := regexp.MustCompile(`(^| )(fetch|push|pull|ls-remote|remote|clone|send-pack|fetch-pack)( |$)`)
	for _, l := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if words.MatchString(l) {
			t.Fatalf("the verb ran git %s", l)
		}
	}
	if len(w.forge.calls) == 0 {
		t.Fatal("the forge was never asked")
	}
	for _, c := range w.forge.calls {
		if len(c) < 2 || c[0] != "api" || c[1] != "graphql" {
			t.Fatalf("the forge was asked %v", c)
		}
	}
	if after, paths := landingsTreeHash(t, gitDir); after != gitBefore {
		t.Fatalf("the verb changed the git directory (%d paths now)", len(paths))
	}
	if after, _ := landingsTreeHash(t, tier, "landings"); after != tierBefore {
		t.Fatal("the verb changed the local tier outside its forge cache")
	}
	after := w.r.Git("status", "--porcelain", "--ignored", "--untracked-files=all")
	for _, l := range strings.Split(after, "\n") {
		if l != "" && !strings.Contains(before, l) && !strings.HasPrefix(l, "!! "+landings.CacheRelDir+"/") {
			t.Fatalf("the verb left %q outside its cache\nbefore:\n%s\nafter:\n%s", l, before, after)
		}
	}
	if !strings.Contains(after, landings.CacheRelDir+"/pr-1.json") {
		t.Fatalf("no answer was kept in the cache:\n%s", after)
	}
}

// landingsFieldPaths is every leaf field of typ, by dotted path: a struct's
// fields are followed through pointers and slices, and a time or a type in
// stop is a leaf.
func landingsFieldPaths(typ reflect.Type, prefix string, stop ...reflect.Type) []string {
	for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct || typ == reflect.TypeFor[time.Time]() || (prefix != "" && slices.Contains(stop, typ)) {
		return []string{prefix}
	}
	var out []string
	for i := range typ.NumField() {
		f := typ.Field(i)
		p := f.Name
		if prefix != "" {
			p = prefix + "." + f.Name
		}
		out = append(out, landingsFieldPaths(f.Type, p, stop...)...)
	}
	return out
}

// landingsAllRow is a row with every field set, each to a value of its own,
// and what the text must print of each, by field path.
func landingsAllRow() (landings.Row, map[string][]string) {
	sha := func(c string) string { return strings.Repeat(c, 40) }
	r := landings.Row{
		Commit: sha("a1"[:1]), When: time.Date(2026, 11, 2, 3, 4, 5, 0, time.UTC), PR: 4242, MergedHead: sha("b"),
		Committer: "Casey Committer", Class: landings.ClassDrain,
		Rule: landings.Rule{Kind: "drain", ID: "adr-77", Path: ".abcd/rule-path.md", At: sha("d")},
		Lane: "lane-kind-x", Approval: landings.Approval{Seen: true, By: "reviewer-login", At: time.Date(2026, 11, 1, 9, 8, 7, 0, time.UTC), Source: landings.SourceForge},
		Mode: "mode-y", Resolves: []string{"iss-11"}, Delivers: []string{"itd-22"}, AssistedBy: []string{"Vendor:model-9"},
		ForgeSays: "the forge says it merged as another commit", ForgeUnreadable: "the forge timed out for this one",
		Unattended: true,
		Reverted:   &landings.Revert{By: sha("e"), ByRow: sha("f"), Kind: "git-kind", WithdrawnBy: sha("9")},
		Reverts: []landings.RevertLine{
			{Commit: sha("1"), Kind: "forge-kind", Names: "#77", Target: sha("2"), Row: sha("3"), Unmarked: "a reason beside the mark"},
			{Commit: sha("4"), Kind: landings.RevertUntraceable, Names: "names-zzz", Target: sha("5"), Unmarked: "names a commit outside the listed range"},
		},
	}
	short := func(c string) string { return sha(c)[:12] }
	return r, map[string][]string{
		"Commit": {short("a")}, "When": {"landed 2026-11-02T03:04:05Z"}, "PR": {"#4242"}, "MergedHead": {"merged head " + short("b")},
		"Committer": {"committer Casey Committer"}, "Class": {"  drain  "},
		"Rule.Kind": {"drain adr-77"}, "Rule.ID": {"drain adr-77"}, "Rule.Path": {"rule read from .abcd/rule-path.md"}, "Rule.At": {"at " + short("d")},
		"Lane":          {"lane lane-kind-x"},
		"Approval.Seen": {"2026-11-01T09:08:07Z, seen"}, "Approval.By": {"approval forge: reviewer-login"}, "Approval.At": {"reviewer-login, 2026-11-01T09:08:07Z"}, "Approval.Source": {"approval forge"},
		"Mode": {"mode mode-y"}, "Resolves": {"iss-11"}, "Delivers": {"itd-22"}, "AssistedBy": {"assisted by Vendor:model-9"},
		"ForgeSays": {"forge says: the forge says it merged as another commit"}, "ForgeUnreadable": {"forge unread: the forge timed out for this one"},
		"Unattended":  {"  unattended"},
		"Reverted.By": {"reverted by " + short("e")}, "Reverted.ByRow": {"in row " + short("f")}, "Reverted.Kind": {"(git-kind, in row"}, "Reverted.WithdrawnBy": {"withdrawn by " + short("9")},
		"Reverts.Commit": {"line in " + short("1"), "line in " + short("4")}, "Reverts.Kind": {"(forge-kind line", "(untraceable line"},
		"Reverts.Names": {"names #77", "names names-zzz"}, "Reverts.Target": {"resolved to " + short("2"), "resolved to " + short("5")},
		"Reverts.Row": {"marks row " + short("3")}, "Reverts.Unmarked": {"; a reason beside the mark", "marks no row: names a commit outside the listed range"},
	}
}

// landingsLeaves is every value the dotted path names under v: a pointer is
// followed, and a slice's every element.
func landingsLeaves(v reflect.Value, path []string) []reflect.Value {
	switch {
	case v.Kind() == reflect.Pointer:
		if v.IsNil() {
			return nil
		}
		return landingsLeaves(v.Elem(), path)
	case len(path) == 0:
		return []reflect.Value{v}
	case v.Kind() == reflect.Slice:
		var out []reflect.Value
		for i := range v.Len() {
			out = append(out, landingsLeaves(v.Index(i), path)...)
		}
		return out
	}
	return landingsLeaves(v.FieldByName(path[0]), path[1:])
}

// landingsChange changes v to another value of its kind: a string gains a
// leading "Z" (so a commit's first twelve characters change too), an int one
// more, a bool its other state, a time an hour; a struct, pointer or slice has
// each of its values changed. A kind it does not know fails t.
func landingsChange(t *testing.T, v reflect.Value) {
	t.Helper()
	switch v.Kind() {
	case reflect.String:
		v.SetString("Z" + v.String())
	case reflect.Int:
		v.SetInt(v.Int() + 1)
	case reflect.Bool:
		v.SetBool(!v.Bool())
	case reflect.Pointer:
		if !v.IsNil() {
			landingsChange(t, v.Elem())
		}
	case reflect.Slice:
		for i := range v.Len() {
			landingsChange(t, v.Index(i))
		}
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[time.Time]() {
			v.Set(reflect.ValueOf(v.Interface().(time.Time).Add(time.Hour)))
			return
		}
		for i := range v.NumField() {
			landingsChange(t, v.Field(i))
		}
	default:
		t.Fatalf("landingsChange: a value of kind %s", v.Kind())
	}
}

// landingsCovers fails t unless want names every leaf path of T (so a field
// added later must be given its text), render(build()) holds every text want
// names, and changing any one leaf of a fresh build() changes what render
// prints, so a field's text is its own and not text the output holds for
// another reason. A bool leaf set false prints none of the texts want names
// for it, so a mark printed on every row fails too.
func landingsCovers[T any](t *testing.T, what string, build func() T, render func(T) string, want map[string][]string, stop ...reflect.Type) {
	t.Helper()
	paths := landingsFieldPaths(reflect.TypeFor[T](), "", stop...)
	for _, p := range paths {
		if _, ok := want[p]; !ok {
			t.Fatalf("%s: the field %s is not given the text it prints", what, p)
		}
	}
	if len(want) != len(paths) {
		t.Fatalf("%s: the test names %d fields, the type has %d (%v)", what, len(want), len(paths), paths)
	}
	text := render(build())
	for p, vals := range want {
		for _, v := range vals {
			if !strings.Contains(text, v) {
				t.Fatalf("%s: the text does not print %s (%q):\n%s", what, p, v, text)
			}
		}
	}
	for _, p := range paths {
		v := build()
		leaves := landingsLeaves(reflect.ValueOf(&v).Elem(), strings.Split(p, "."))
		if len(leaves) == 0 {
			t.Fatalf("%s: the built value holds no %s to change", what, p)
		}
		wasTrue := leaves[0].Kind() == reflect.Bool && leaves[0].Bool()
		for _, l := range leaves {
			landingsChange(t, l)
		}
		got := render(v)
		if got == text {
			t.Fatalf("%s: the text does not change when %s does:\n%s", what, p, text)
		}
		for _, s := range want[p] {
			if wasTrue && strings.Contains(got, s) {
				t.Fatalf("%s: with %s false the text still prints %q:\n%s", what, p, s, got)
			}
		}
	}
}

// TestLandingsTextSaysWhatTheJSONSays: everything --json carries of a row,
// the listing, a group and a trace, the text prints, a row's attendance and
// reverts included: the unattended mark, the mode, what the forge said and
// why its answer was not read, and every revert line with the row it marks
// or why it marks none. Each struct's fields are walked, so one added later
// fails here until it is printed.
func TestLandingsTextSaysWhatTheJSONSays(t *testing.T) {
	buildRow := func() landings.Row { r, _ := landingsAllRow(); return r }
	row, want := landingsAllRow()
	text := strings.Join(landingRow(row), "\n")
	landingsCovers(t, "a row", buildRow, func(r landings.Row) string { return strings.Join(landingRow(r), "\n") }, want)

	rowType := reflect.TypeFor[landings.Row]()
	sha := func(c string) string { return strings.Repeat(c, 40) }
	buildListing := func() landings.Listing {
		return landings.Listing{Tip: "tip-ref", From: sha("6"), To: sha("7"), Since: "since-tag", SinceNote: "a since note",
			Rows: []landings.Row{buildRow()}, ForgeUnread: "a forge note", CacheUnwritten: "a cache note", ModeUnread: "a mode note"}
	}
	render := func(f func(io.Writer)) string {
		var b bytes.Buffer
		f(&b)
		return b.String()
	}
	landingsCovers(t, "the listing", buildListing, func(l landings.Listing) string {
		return render(func(w io.Writer) { renderLandings(w, landingsView(l, false, false), false, false) })
	}, map[string][]string{
		"Tip": {"read tip-ref"}, "From": {"after " + sha("6")[:12]}, "To": {"at " + sha("7")[:12]}, "Since": {"since since-tag"},
		"SinceNote": {"range: a since note"}, "Rows": {text}, "ForgeUnread": {"forge: a forge note"},
		"CacheUnwritten": {"cache: a cache note"}, "ModeUnread": {"mode: a mode note"},
	}, rowType)

	indented := strings.ReplaceAll("  "+text, "\n", "\n  ")
	landingsCovers(t, "a group", func() landings.RevertGroup {
		return landings.RevertGroup{Rule: "group-rule", Count: 1, Rows: []landings.Row{buildRow()}}
	}, func(g landings.RevertGroup) string {
		return render(func(w io.Writer) {
			renderLandings(w, landingsOutput{Listing: buildListing(), Groups: []landings.RevertGroup{g}}, false, true)
		})
	}, map[string][]string{
		"Rule": {"group-rule  1\n"}, "Count": {"group-rule  1\n"}, "Rows": {indented},
	}, rowType)

	// The trace's Says is empty here, so its row prints; the second render
	// says, and prints the name it was given, as a trace that names no commit
	// does.
	landingsCovers(t, "a trace", func() landings.Trace {
		r, other := buildRow(), buildRow()
		other.Commit = sha("8")
		return landings.Trace{Commit: sha("c"), Row: &r, Reverts: []landings.Row{other}, Tip: "tip-ref", To: sha("7"), From: sha("6"),
			RangeNote: "a range note", ForgeUnread: "a forge note", CacheUnwritten: "a cache note", ModeUnread: "a mode note"}
	}, func(tr landings.Trace) string {
		return render(func(w io.Writer) {
			renderTrace(w, "c", tr)
			renderTrace(w, "c", landings.Trace{Says: "is ambiguous here", Tip: tr.Tip, To: tr.To})
		})
	}, map[string][]string{
		"Commit": {sha("c")[:12] + " landed in:"}, "Says": {"c: is ambiguous here"}, "Row": {indented},
		"Reverts": {"it reverts:\n  " + sha("8")[:12]}, "Tip": {"read tip-ref"}, "To": {"at " + sha("7")[:12]}, "From": {"listed after " + sha("6")[:12]},
		"RangeNote": {"range: a range note"}, "ForgeUnread": {"forge: a forge note"}, "CacheUnwritten": {"cache: a cache note"}, "ModeUnread": {"mode: a mode note"},
	}, rowType)
}

// TestLandingsTextNamesWhyARevertIsUntraceable: a trace of a reverting
// commit whose line marks no row prints the line and its reason, and under
// --reverted the untraceable group's rows print theirs, through the verb.
func TestLandingsTextNamesWhyARevertIsUntraceable(t *testing.T) {
	w := newLandingsWorld(t, "auto")
	w.r.Git("tag", "start", w.head())
	rv := w.push("Revert \"feat: gone\"")
	text, errOut, code := runLandings(t, rv)
	if code != 0 || !strings.Contains(text, "marks no row: ") {
		t.Fatalf("trace of an untraceable revert (exit %d, %s):\n%s", code, errOut, text)
	}
	l := landingsJSON(t, "--since", "start")
	why := l.Rows[0].Reverts[0].Unmarked
	if why == "" || !strings.Contains(text, why) {
		t.Fatalf("the trace does not print the reason %q:\n%s", why, text)
	}
	text, _, _ = runLandings(t, "--since", "start", "--reverted")
	if !strings.HasPrefix(text, landings.GroupUntraceable+"  1\n  "+rv[:12]) || !strings.Contains(text, why) {
		t.Fatalf("the untraceable group:\n%s", text)
	}
}

// TestLandingsRowTextIsBounded: a row prints at most landingsMaxIDs of its
// ids and of its Assisted-by: values, landingsMaxUnmarkedLines of its revert
// lines that mark no row and landingsMaxRevertLines of those that mark one,
// then how many more of each, so a row carrying thousands prints no more than
// one carrying fifty but for the counts' digits.
func TestLandingsRowTextIsBounded(t *testing.T) {
	rowOf := func(n int) landings.Row {
		r, _ := landingsAllRow()
		r.Resolves, r.Delivers, r.AssistedBy, r.Reverts = nil, nil, nil, nil
		for i := range n {
			r.Resolves = append(r.Resolves, fmt.Sprintf("iss-%d", 1000000+i))
			r.Delivers = append(r.Delivers, fmt.Sprintf("itd-%d", 1000000+i))
			r.AssistedBy = append(r.AssistedBy, fmt.Sprintf("Vendor:model-%d", 1000000+i))
			r.Reverts = append(r.Reverts,
				landings.RevertLine{Commit: fmt.Sprintf("%040x", i), Kind: landings.RevertGit, Names: fmt.Sprintf("%040x", i), Row: fmt.Sprintf("%040x", i)},
				landings.RevertLine{Commit: fmt.Sprintf("%040x", i), Kind: landings.RevertGit, Names: fmt.Sprintf("%040x", i), Unmarked: "names a commit outside the listed range"})
		}
		return r
	}
	small := strings.Join(landingRow(rowOf(50)), "\n")
	big := strings.Join(landingRow(rowOf(5000)), "\n")
	hidden := 2*5000 - landingsMaxRevertLines - landingsMaxUnmarkedLines
	for _, more := range []string{fmt.Sprintf("and %d more", 2*5000-landingsMaxIDs), fmt.Sprintf("and %d more", 5000-landingsMaxIDs),
		fmt.Sprintf("and %d more revert lines: %d marking a row, %d with no row marked", hidden, 5000-landingsMaxRevertLines, 5000-landingsMaxUnmarkedLines)} {
		if !strings.Contains(big, more) {
			t.Fatalf("the row does not say %q:\n%s", more, big)
		}
	}
	if d := len(big) - len(small); d < 0 || d > 10 {
		t.Fatalf("a row of 5000 ids and lines prints %d bytes, one of 50 prints %d", len(big), len(small))
	}
	if n := strings.Count(big, "\n  reverts ("); n != landingsMaxRevertLines+landingsMaxUnmarkedLines {
		t.Fatalf("the row prints %d revert lines", n)
	}
}

// TestLandingsNoCapHidesWhyARevertMarksNoRow: a row whose ninth and later
// revert lines are the ones that mark no row prints every one of them, the
// untraceable one first, before the lines that mark a row, in a listing,
// under the reverted filter (where the untraceable line is why the row is in
// its group) and in a trace; and what the cap hides is counted by whether it
// marks a row.
func TestLandingsNoCapHidesWhyARevertMarksNoRow(t *testing.T) {
	r, _ := landingsAllRow()
	r.Reverts = nil
	for i := range landingsMaxRevertLines + 3 {
		r.Reverts = append(r.Reverts, landings.RevertLine{Commit: fmt.Sprintf("%040x", i), Kind: landings.RevertGit,
			Names: fmt.Sprintf("%040x", i), Target: fmt.Sprintf("%040x", i+100), Row: fmt.Sprintf("%040x", i+200)})
	}
	reasons := []string{"reason one: names a commit outside the listed range", "reason two: 3 more revert lines past the 64 a commit is read for"}
	for _, why := range reasons {
		r.Reverts = append(r.Reverts, landings.RevertLine{Commit: strings.Repeat("e", 40), Kind: landings.RevertGit, Unmarked: why})
	}
	untraceable := "reason three: its subject reads as git's revert, but no line names what it reverts"
	r.Reverts = append(r.Reverts, landings.RevertLine{Commit: strings.Repeat("f", 40), Kind: landings.RevertUntraceable, Unmarked: untraceable})
	reasons = append(reasons, untraceable)

	l := landings.Listing{Tip: "main", To: strings.Repeat("7", 40), Rows: []landings.Row{r}}
	var listing, grouped, traced bytes.Buffer
	renderLandings(&listing, landingsView(l, false, false), false, false)
	view := landingsView(l, false, true)
	renderLandings(&grouped, view, false, true)
	renderTrace(&traced, "c", landings.Trace{Commit: strings.Repeat("c", 40), Row: &r, Reverts: []landings.Row{r}, Tip: "main", To: l.To})
	if len(view.Groups) == 0 || view.Groups[len(view.Groups)-1].Rule != landings.GroupUntraceable {
		t.Fatalf("the row is not in the untraceable group: %+v", view.Groups)
	}
	for mode, text := range map[string]string{"a listing": listing.String(), "the reverted groups": grouped.String(), "a trace": traced.String()} {
		for _, why := range reasons {
			if !strings.Contains(text, "marks no row: "+why) {
				t.Fatalf("%s does not print %q:\n%s", mode, why, text)
			}
		}
		if u, first := strings.Index(text, untraceable), strings.Index(text, "marks no row: "+reasons[0]); u < 0 || u > first || first > strings.Index(text, ": marks row ") {
			t.Fatalf("%s does not print the untraceable line, then the others marking no row, before those marking one:\n%s", mode, text)
		}
		if want := "and 3 more revert lines: 3 marking a row, 0 with no row marked"; !strings.Contains(text, want) {
			t.Fatalf("%s does not say %q:\n%s", mode, want, text)
		}
	}
}

// landingsHostile is text carrying every kind of rune the terminal mask
// masks, a line break that would forge a line, and a byte that is no UTF-8.
const landingsHostile = "x\x1b[2J\u009b31m\u202e\u200b\x07\x7f\U000E0041\r\nFORGED\t\xff"

// landingsMasked are the runes landingsHostile carries that must never reach
// the text raw.
var landingsMasked = []string{"\x1b", "\u009b", "\u202e", "\u200b", "\x07", "\x7f", "\U000E0041", "\r", "\t", "\xff"}

// landingsFill sets every string v holds to s, every int to 7, every bool to
// true and every time to one instant, giving each nil pointer and empty slice
// one element first. A kind it does not know fails t, so a field of a kind
// added later is not left unset and unchecked.
func landingsFill(t *testing.T, v reflect.Value, s string) {
	t.Helper()
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Int:
		v.SetInt(7)
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Pointer:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		landingsFill(t, v.Elem(), s)
	case reflect.Slice:
		if v.Len() == 0 {
			v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		}
		for i := range v.Len() {
			landingsFill(t, v.Index(i), s)
		}
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[time.Time]() {
			v.Set(reflect.ValueOf(time.Date(2026, 11, 3, 4, 5, 6, 0, time.UTC)))
			return
		}
		for i := range v.NumField() {
			landingsFill(t, v.Field(i), s)
		}
	default:
		t.Fatalf("landingsFill: a field of kind %s %s is not filled", v.Kind(), v.Type())
	}
}

// landingsStrings is every string v holds, in the order landingsFill fills
// them, each settable.
func landingsStrings(v reflect.Value) []reflect.Value {
	switch v.Kind() {
	case reflect.String:
		return []reflect.Value{v}
	case reflect.Pointer:
		if v.IsNil() {
			return nil
		}
		return landingsStrings(v.Elem())
	case reflect.Slice:
		var out []reflect.Value
		for i := range v.Len() {
			out = append(out, landingsStrings(v.Index(i))...)
		}
		return out
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[time.Time]() {
			return nil
		}
		var out []reflect.Value
		for i := range v.NumField() {
			out = append(out, landingsStrings(v.Field(i))...)
		}
		return out
	}
	return nil
}

// landingsFilled is a T with every field filled with s by landingsFill, and
// then its string number empty set to "" (none for empty < 0), and how many
// strings it holds.
func landingsFilled[T any](t *testing.T, s string, empty int) (T, int) {
	t.Helper()
	var v T
	landingsFill(t, reflect.ValueOf(&v).Elem(), s)
	strs := landingsStrings(reflect.ValueOf(&v).Elem())
	if empty >= 0 {
		strs[empty].SetString("")
	}
	return v, len(strs)
}

// landingsNoRaw fails t if text carries a masked rune raw, is not UTF-8, or
// holds a line the hostile text's line break forged.
func landingsNoRaw(t *testing.T, mode, text string) {
	t.Helper()
	if !utf8.ValidString(text) {
		t.Fatalf("%s: the text is not UTF-8: %q", mode, text)
	}
	for _, bad := range landingsMasked {
		if strings.Contains(text, bad) {
			t.Fatalf("%s: the text carries %q raw:\n%q", mode, bad, text)
		}
	}
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "FORGED") {
			t.Fatalf("%s: a line was forged:\n%q", mode, text)
		}
	}
}

// landingsCheckMasked is landingsNoRaw, and fails t too unless the masked
// hostile text is printed.
func landingsCheckMasked(t *testing.T, mode, text string) {
	t.Helper()
	landingsNoRaw(t, mode, text)
	if !strings.Contains(text, "x?[2J") {
		t.Fatalf("%s: the masked text is not printed:\n%q", mode, text)
	}
}

// TestLandingsRenderMasksEveryString: every string of the listing, a row, a
// revert mark, a revert line, a trace and a group, filled by walking the
// structs (so a field added later is filled too) with text carrying control
// and escape runes, reaches no output raw: not the listing, the unattended
// filter, the reverted groups or the trace in text, whatever the reader
// handed the render, and not --json, whose \u escapes are the mask there and
// read back every string it was given. The text is checked again with each
// string emptied in turn, so a value printed only when another is empty (a
// person's push's committer where the rule would be, a revert line that marks
// no row, a trace's name where it names no commit) is reached too, whatever
// fields are added later.
func TestLandingsRenderMasksEveryString(t *testing.T) {
	text := func(f func(io.Writer)) string {
		var b bytes.Buffer
		f(&b)
		return b.String()
	}
	_, nOut := landingsFilled[landingsOutput](t, landingsHostile, -1)
	for empty := -1; empty < nOut; empty++ {
		out, _ := landingsFilled[landingsOutput](t, landingsHostile, empty)
		views := map[string]landingsOutput{
			"the listing":         landingsView(out.Listing, false, false),
			"the unattended rows": landingsView(out.Listing, true, false),
			"the reverted groups": landingsView(out.Listing, false, true),
			"the groups handed":   out,
		}
		for mode, v := range views {
			landingsCheckMasked(t, fmt.Sprintf("%s, string %d empty", mode, empty), text(func(w io.Writer) {
				renderLandings(w, v, mode == "the unattended rows", mode != "the listing" && mode != "the unattended rows")
			}))
		}
		if g := views["the reverted groups"].Groups; empty < 0 && len(g) == 0 {
			t.Fatal("the filled listing groups no row")
		}
	}
	_, nTrace := landingsFilled[landings.Trace](t, landingsHostile, -1)
	for empty := -1; empty < nTrace; empty++ {
		tr, _ := landingsFilled[landings.Trace](t, landingsHostile, empty)
		landingsCheckMasked(t, fmt.Sprintf("a trace that says, string %d empty", empty), text(func(w io.Writer) { renderTrace(w, landingsHostile, tr) }))
		tr.Says = ""
		landingsCheckMasked(t, fmt.Sprintf("a trace, string %d empty", empty), text(func(w io.Writer) { renderTrace(w, landingsHostile, tr) }))
	}

	valid := strings.ToValidUTF8(landingsHostile, "\ufffd")
	out, _ := landingsFilled[landingsOutput](t, landingsHostile, -1)
	tr, _ := landingsFilled[landings.Trace](t, landingsHostile, -1)
	for mode, v := range map[string]any{"--json listing": landingsView(out.Listing, false, false), "--json unattended": landingsView(out.Listing, true, false),
		"--json reverted": landingsView(out.Listing, false, true), "--json trace": tr} {
		var b bytes.Buffer
		if err := landingsRender(&b, true, v, nil); err != nil {
			t.Fatal(err)
		}
		for _, r := range b.String() {
			if r != '\n' && termsafe.Sanitize(string(r)) != string(r) {
				t.Fatalf("%s carries %U raw:\n%s", mode, r, b.String())
			}
		}
		// Read back into the type written, every string it holds is the one
		// written (each holds the hostile text: a group's rule is made of the
		// row rule's two strings), and it holds as many as were written.
		back := reflect.New(reflect.TypeOf(v))
		if err := json.Unmarshal(b.Bytes(), back.Interface()); err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
		got, sent := landingsStrings(back.Elem()), landingsStrings(reflect.ValueOf(&v).Elem().Elem())
		if len(got) != len(sent) {
			t.Fatalf("%s reads back %d strings, %d were written", mode, len(got), len(sent))
		}
		for i, s := range got {
			if want := strings.ToValidUTF8(sent[i].String(), "\ufffd"); s.String() != want || !strings.Contains(want, valid) {
				t.Fatalf("%s reads back string %d as %q, not %q", mode, i, s.String(), want)
			}
		}
	}
}

// TestLandingsClipIsExact: the clip never prints more than its bound, never
// cuts inside a rune, and prints only masked UTF-8, on any input, invalid
// UTF-8 included.
func TestLandingsClipIsExact(t *testing.T) {
	for _, s := range []string{strings.Repeat("\xff", 50), strings.Repeat("é", 30) + "\xff\xfe" + strings.Repeat("語", 20), "ab\xe8\xaa", landingsHostile} {
		for n := 0; n <= len(s)+2; n++ {
			got := landingsClip(s, n)
			if len(got) > n || !utf8.ValidString(got) || termsafe.Sanitize(got) != got {
				t.Fatalf("clip(%q, %d) = %q (%d bytes)", s, n, got, len(got))
			}
		}
	}
}

// landingsJSONKeys are the keys --json prints, by object, as the command
// page lists them.
var landingsJSONKeys = map[string][]string{
	"The listing":             {"tip", "from", "to", "since", "since_note", "rows", "forge_unread", "cache_unwritten", "mode_unread", "groups"},
	"A row":                   {"commit", "when", "pr", "merged_head", "committer", "class", "rule", "lane", "approval", "mode", "resolves", "delivers", "assisted_by", "forge_says", "forge_unreadable", "unattended", "reverted", "reverts"},
	"A row's rule":            {"kind", "id", "path", "at"},
	"A row's approval":        {"seen", "by", "at", "source"},
	"A row's reverted mark":   {"by", "by_row", "kind", "withdrawn_by"},
	"Each of a row's reverts": {"commit", "kind", "names", "target", "row", "unmarked"},
	"A group":                 {"rule", "count", "rows"},
	"A trace":                 {"commit", "says", "row", "reverts", "tip", "to", "from", "range_note", "forge_unread", "cache_unwritten", "mode_unread"},
}

// landingsKeys is the sorted keys of the JSON object raw.
func landingsKeys(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("%v: %s", err, raw)
	}
	keys := slices.Collect(maps.Keys(m))
	slices.Sort(keys)
	return keys
}

// TestLandingsJSONKeys: the full key set of the listing, a row and its
// sub-objects, a group and a trace, every field filled, is the list this
// test holds, and the command page lists exactly those keys; under
// --reverted with nothing reverted, groups is an empty list, and without it
// there is no groups key.
func TestLandingsJSONKeys(t *testing.T) {
	var out landingsOutput
	landingsFill(t, reflect.ValueOf(&out).Elem(), "v")
	var tr landings.Trace
	landingsFill(t, reflect.ValueOf(&tr).Elem(), "v")
	raw := func(v any) json.RawMessage {
		var b bytes.Buffer
		if err := landingsRender(&b, true, v, nil); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	var listing struct {
		Rows   []json.RawMessage `json:"rows"`
		Groups []json.RawMessage `json:"groups"`
	}
	lraw := raw(out)
	if err := json.Unmarshal(lraw, &listing); err != nil || len(listing.Rows) != 1 || len(listing.Groups) != 1 {
		t.Fatalf("%v: %s", err, lraw)
	}
	var row struct {
		Rule, Approval, Reverted json.RawMessage
		Reverts                  []json.RawMessage
	}
	if err := json.Unmarshal(listing.Rows[0], &row); err != nil || len(row.Reverts) != 1 {
		t.Fatalf("%v: %s", err, listing.Rows[0])
	}
	got := map[string][]string{
		"The listing": landingsKeys(t, lraw), "A row": landingsKeys(t, listing.Rows[0]), "A row's rule": landingsKeys(t, row.Rule),
		"A row's approval": landingsKeys(t, row.Approval), "A row's reverted mark": landingsKeys(t, row.Reverted),
		"Each of a row's reverts": landingsKeys(t, row.Reverts[0]), "A group": landingsKeys(t, listing.Groups[0]), "A trace": landingsKeys(t, raw(tr)),
	}
	page, err := os.ReadFile(filepath.Join(repoRootForTest(t), "commands", "landings.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, section, _ := strings.Cut(string(page), "\n## What --json prints\n")
	section, _, _ = strings.Cut(section, "\n## ")
	onPage := map[string][]string{}
	keyRe := regexp.MustCompile("`([a-z_]+)`")
	for _, l := range strings.Split(strings.ReplaceAll(section, "\n  ", " "), "\n") {
		label, keys, ok := strings.Cut(strings.TrimPrefix(l, "- "), ": ")
		if !ok || !strings.HasPrefix(l, "- ") {
			continue
		}
		for _, m := range keyRe.FindAllStringSubmatch(keys, -1) {
			onPage[label] = append(onPage[label], m[1])
		}
	}
	for obj, want := range landingsJSONKeys {
		want = slices.Sorted(slices.Values(want))
		if !slices.Equal(got[obj], want) {
			t.Fatalf("%s carries %v, the test holds %v", obj, got[obj], want)
		}
		if page := slices.Sorted(slices.Values(onPage[obj])); !slices.Equal(page, want) {
			t.Fatalf("the page lists %s as %v, the test holds %v", obj, page, want)
		}
	}
	if len(onPage) != len(landingsJSONKeys) {
		t.Fatalf("the page lists %d objects, the test holds %d: %v", len(onPage), len(landingsJSONKeys), onPage)
	}

	w := newLandingsWorld(t, "auto")
	w.r.Git("tag", "start", w.head())
	w.merge(1, "", "feat: one")
	r, _, code := runLandings(t, "--json", "--since", "start", "--reverted")
	if code != 0 || !strings.Contains(r, `"groups": []`) {
		t.Fatalf("--reverted --json with nothing reverted (exit %d):\n%s", code, r)
	}
	if r, _, _ := runLandings(t, "--json", "--since", "start"); strings.Contains(r, `"groups"`) {
		t.Fatalf("--json without --reverted carries groups:\n%s", r)
	}
}

// landingsCollidingBlobs writes two blobs to r whose object names share
// their first seven hex characters, and returns that prefix.
func landingsCollidingBlobs(t *testing.T, r *gittest.Repo) string {
	t.Helper()
	blobID := func(c string) string {
		h := sha1.Sum([]byte(fmt.Sprintf("blob %d\x00%s", len(c), c)))
		return hex.EncodeToString(h[:])
	}
	seen := map[string]string{}
	for i := 0; ; i++ {
		c := strconv.Itoa(i)
		id := blobID(c)
		if other, ok := seen[id[:7]]; ok {
			for _, x := range []string{other, c} {
				cmd := exec.Command("git", "-C", r.Root(), "hash-object", "-w", "--stdin")
				cmd.Env = r.Env()
				cmd.Stdin = strings.NewReader(x)
				got, err := cmd.Output()
				if err != nil || strings.TrimSpace(string(got)) != blobID(x) {
					t.Fatalf("git wrote %q as %q (%v), not %s", x, got, err, blobID(x))
				}
			}
			return id[:7]
		}
		seen[id[:7]] = c
	}
}

// TestLandingsExitCodes: every exit the help and the page state, at the verb.
func TestLandingsExitCodes(t *testing.T) {
	w := newLandingsWorld(t, "auto")
	w.merge(1, "", "feat: one")
	w.r.Git("tag", "v0.1.0", w.head())
	w.push("docs: after the tag")
	tree := w.r.Git("rev-parse", "HEAD^{tree}")
	ambiguous := landingsCollidingBlobs(t, w.r)
	for _, tc := range []struct {
		name string
		args []string
		fail []string
		code int
		says string
	}{
		{"unreadable history", nil, []string{"--timestamp"}, 1, "first-parent line"},
		{"unreadable tags", nil, []string{"describe"}, 1, "release tag"},
		{"a date that is no day", []string{"--since", "2030-02-30"}, nil, 2, "is not a date"},
		{"both filters", []string{"--unattended", "--reverted"}, nil, 2, "not taken together"},
		{"an ambiguous name", []string{ambiguous}, nil, 2, "is ambiguous"},
		{"a name of a tree", []string{tree}, nil, 2, "names a tree, not a commit"},
		{"a listing read", nil, nil, 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.fail != nil {
				landingsGit(t, "", tc.fail...)
			}
			_, errOut, code := runLandings(t, tc.args...)
			if code != tc.code || !strings.Contains(errOut, tc.says) {
				t.Fatalf("exit %d, want %d: %q", code, tc.code, errOut)
			}
		})
	}
	out, errOut, code := runLandings(t, "--json", tree)
	var tr landings.Trace
	if code != 2 || json.Unmarshal([]byte(out), &tr) != nil || !strings.Contains(tr.Says, "names a tree") || !strings.Contains(errOut, "names a tree") {
		t.Fatalf("--json on a refused trace: exit %d, %q, %q", code, out, errOut)
	}

	t.Run("outside a checkout", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if _, errOut, code := runLandings(t); code != 2 || !strings.Contains(errOut, "nothing read") {
			t.Fatalf("exit %d: %q", code, errOut)
		}
	})
	t.Run("no default branch", func(t *testing.T) {
		r := gittest.NewRepo(t)
		r.Commit("chore: one")
		r.Git("branch", "-m", "feature")
		t.Chdir(r.Root())
		if _, errOut, code := runLandings(t, "--since", "2020-01-01"); code != 2 || !strings.Contains(errOut, landings.ErrNoDefaultBranch.Error()) {
			t.Fatalf("exit %d: %q", code, errOut)
		}
	})
}

// TestLandingsForgeUnreadStaysExitZeroUnderEveryFilter: a listing whose forge
// answers were not read exits 0, with its rows unattended and the forge note
// saying how many, bare, under each filter and in a trace.
func TestLandingsForgeUnreadStaysExitZeroUnderEveryFilter(t *testing.T) {
	w := newLandingsWorld(t, "auto")
	w.r.Git("tag", "start", w.head())
	m, branch := w.merge(1, "alice-reviewer", "feat: one")
	w.r.Git("revert", "--no-edit", branch[0])
	w.forge.fail = "the forge is away"
	for _, args := range [][]string{{"--since", "start"}, {"--since", "start", "--unattended"}, {"--since", "start", "--reverted"}, {m}} {
		text, errOut, code := runLandings(t, args...)
		if code != 0 || !strings.Contains(text, "\nforge: the forge's answer was not read for 1 row") {
			t.Fatalf("landings %v: exit %d, %s\n%s", args, code, errOut, text)
		}
	}
	if l := landingsJSON(t, "--since", "start", "--unattended"); !strings.Contains(l.ForgeUnread, "not read for 1 row") || len(l.Rows) != 2 {
		t.Fatalf("--unattended --json: %+v", l)
	}
}

// TestLandingsMasksTheOperandItIsGiven: the name a trace is given and the
// --since value reach the refusal at exit 2 masked, and a release tag whose
// name carries runes the mask masks (git takes them in a tag name) reaches the
// listing's text masked: in the row's range line and the tip read.
func TestLandingsMasksTheOperandItIsGiven(t *testing.T) {
	w := newLandingsWorld(t, "auto")
	tag := "v1\u009b31m‮​\U000E0041"
	w.r.Git("tag", tag, w.head())
	w.merge(1, "", "feat: one")
	for _, args := range [][]string{{landingsHostile}, {"--json", landingsHostile}, {"--since", landingsHostile}, {"--json", "--since", landingsHostile}} {
		out, errOut, code := runLandings(t, args...)
		if code != 2 || !strings.Contains(errOut, "x?[2J") {
			t.Fatalf("landings %q: exit %d, %q", args, code, errOut)
		}
		landingsNoRaw(t, fmt.Sprintf("the refusal of %q", args), errOut)
		landingsNoRaw(t, fmt.Sprintf("the output of %q", args), out)
	}
	for _, args := range [][]string{{"--since", tag}, nil} {
		text, errOut, code := runLandings(t, args...)
		if code != 0 || !strings.Contains(text, "since v1?31m???") {
			t.Fatalf("landings %q: exit %d, %s\n%q", args, code, errOut, text)
		}
		landingsNoRaw(t, fmt.Sprintf("the listing of %q", args), text)
	}
}

// TestLandingsARenamedReleaseTagNamesTheFlag: git describes an annotated
// tag by the name inside it, so a release tag whose ref was renamed is
// described under a name no ref holds; the verb refuses at exit 2 and tells
// the person to name --since, as it does with no release tag.
func TestLandingsARenamedReleaseTagNamesTheFlag(t *testing.T) {
	w := newLandingsWorld(t, "auto")
	w.r.Git("tag", "-a", "v1.0.0", "-m", "release", w.head())
	w.r.Git("update-ref", "refs/tags/v2.0.0", "refs/tags/v1.0.0")
	w.r.Git("update-ref", "-d", "refs/tags/v1.0.0")
	w.merge(1, "", "feat: one")
	out, errOut, code := runLandings(t)
	if code != 2 || !strings.Contains(errOut, "--since") || out != "" {
		t.Fatalf("a renamed release tag: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
}

// TestLandingsUnattendedSaysWhenNoRowIs: with the filter and no unattended
// row, the text says no row in the range is unattended, not that nothing
// landed.
func TestLandingsUnattendedSaysWhenNoRowIs(t *testing.T) {
	w := newLandingsWorld(t, "human")
	w.r.Git("tag", "start", w.head())
	w.merge(1, "alice-reviewer", "feat: one")
	text, _, code := runLandings(t, "--since", "start", "--unattended")
	if code != 0 || !strings.HasPrefix(text, "no row in the range is unattended\n") {
		t.Fatalf("exit %d:\n%s", code, text)
	}
}
