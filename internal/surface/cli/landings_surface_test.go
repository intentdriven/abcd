package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/landings"
	"github.com/intentdriven/abcd/internal/gittest"
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
	if !strings.Contains(text, "#1  person-approved  facilitator.mode: auto") || !strings.Contains(text, "approval forge: alice-reviewer, 2026-11-01T09:00:00Z  iss-1") ||
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
	if strings.Contains(text, m1[:12]) || !strings.Contains(text, m2[:12]) || !strings.Contains(text, p[:12]) {
		t.Fatalf("--unattended text:\n%s", text)
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
	if !strings.Contains(text, "pushed by Pat ?[31mPusher?2J?") || !strings.Contains(text, "\nforge: ") {
		t.Fatalf("the masked committer or the forge note is missing:\n%q", text)
	}
	for _, l := range strings.Split(text, "\n") {
		if len(l) > 2*landingsNoteBytes {
			t.Fatalf("a line of %d bytes: %q", len(l), l)
		}
	}
}

// TestLandingsFetchesNothingAndWritesOnlyItsCache: the listing and the trace
// make no git call that fetches or pushes, ask the forge only GraphQL
// documents, and leave the tree as it was but for the forge cache in the
// local tier.
func TestLandingsFetchesNothingAndWritesOnlyItsCache(t *testing.T) {
	w := newLandingsWorld(t, "auto")
	w.r.Git("tag", "start", w.head())
	_, branch := w.merge(1, "", "feat: one")
	w.push("docs: pushed")
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	script := "#!/bin/sh\n{ printf '%s' \"$*\" | tr '\\n' ' '; echo; } >> '" + log + "'\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	before := w.r.Git("status", "--porcelain", "--ignored", "--untracked-files=all")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, args := range [][]string{{"--since", "start"}, {"--json", "--since", "start", "--reverted"}, {branch[0]}} {
		if out, errOut, code := runLandings(t, args...); code != 0 {
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

// TestLandingsRenderMasksAndBoundsWhatTheReaderHandsIt: the render masks and
// bounds every value it prints itself, whatever the reader handed it, so a
// value a later reader left raw or long still reaches the terminal safe.
func TestLandingsRenderMasksAndBoundsWhatTheReaderHandsIt(t *testing.T) {
	long := strings.Repeat("y", 5000)
	r := landings.Row{Commit: strings.Repeat("a", 40), PR: 7, Class: "person-push\x1b[2J", Committer: "Pat\x1b]0;title\x07" + long,
		Lane: "drain‮", Approval: landings.Approval{Seen: true, By: "who\u009b", Source: "forge"},
		Resolves: []string{"iss-1\x1b[1m"}, Delivers: []string{"itd-2​"}}
	out := landingsOutput{Listing: landings.Listing{Tip: "main\x1b[2J", To: strings.Repeat("b", 40), Since: "v1\x1b", Rows: []landings.Row{r},
		ForgeUnread: "boom\x1b[2J" + long, SinceNote: "note\u009b", CacheUnwritten: "c\x07", ModeUnread: "m‮"}}
	var b bytes.Buffer
	renderLandings(&b, out, false)
	text := b.String()
	for _, bad := range []string{"\x1b", "\x07", "\u009b", "‮", "​"} {
		if strings.Contains(text, bad) {
			t.Fatalf("the render printed %q raw:\n%q", bad, text)
		}
	}
	for _, l := range strings.Split(text, "\n") {
		if len(l) > 2*landingsNoteBytes {
			t.Fatalf("a line of %d bytes", len(l))
		}
	}
	b.Reset()
	renderTrace(&b, "abc\x1b1234", landings.Trace{Says: "is \x1b[2Jnot", Tip: "main\x1b", To: strings.Repeat("c", 40)})
	if strings.Contains(b.String(), "\x1b") {
		t.Fatalf("the trace printed ESC raw: %q", b.String())
	}
}
