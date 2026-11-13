package spotcheck

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// clone is a checkout with one commit on main and origin at bare.
func clone(t *testing.T, bare string) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("the base")
	r.Git("remote", "add", "origin", bare)
	return r
}

// twoClones are two checkouts of one bare remote.
func twoClones(t *testing.T) (a, b *gittest.Repo, bare string) {
	t.Helper()
	bare = filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	return clone(t, bare), clone(t, bare), bare
}

var at0 = time.Date(2026, 11, 13, 7, 0, 0, 0, time.UTC)

func handBack(issue string, at time.Time) HandBack {
	return HandBack{At: at, Issue: issue, From: "lane", Kind: "stuck", Route: "home", Reason: "the lane kept refusing",
		PR: 12, RunID: "run-2611130703410770", DrainRule: "adr-2609291342092738", Holder: "h_0123456789abcdef0123456789abcdef"}
}

func gateDecision(pr int, head string, arm bool, at time.Time) Decision {
	return Decision{Kind: KindMergeGate, At: at, Site: SiteLane, PR: pr, Head: head, Arm: arm, Method: "--merge",
		Why: "facilitator.mode is auto (repo layer, .abcd/config.json at origin/main), but no review receipt names it"}
}

// write writes b, failing the test on a write that fails.
func write(t *testing.T, root string, b Batch) Written {
	t.Helper()
	w := Write(context.Background(), root, b)
	if w.Err != nil || w.Failed != 0 {
		t.Fatalf("Write = %+v", w)
	}
	return w
}

func hb(hs ...HandBack) Batch { return Batch{HandBacks: hs} }

func readLocal(t *testing.T, root string) Records {
	t.Helper()
	rec, ok, err := read(context.Background(), root, LocalRef)
	if err != nil || !ok {
		t.Fatalf("read(local) = ok %v, %v", ok, err)
	}
	return rec
}

func remoteFiles(t *testing.T, bare string) []string {
	t.Helper()
	return strings.Fields(gitIn(t, bare, "ls-tree", "-r", "--name-only", BranchRef))
}

func localFiles(t *testing.T, r *gittest.Repo) []string {
	t.Helper()
	return strings.Fields(r.Git("ls-tree", "-r", "--name-only", LocalRef))
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	return gittest.OpenRepo(t, dir).Git(args...)
}

// commitFilesAt makes a commit over parent ("" for none) holding parent's
// files and each of files at its path, as another writer would.
func commitFilesAt(t *testing.T, r *gittest.Repo, parent string, files map[string]string) string {
	t.Helper()
	ctx := context.Background()
	var placed []placed
	for p, body := range files {
		sha, err := storeGit(ctx, r.Root(), "", body, "hash-object", "-w", "--stdin")
		if err != nil {
			t.Fatal(err)
		}
		placed = append(placed, placedAt(p, sha))
	}
	base := parent
	if base == "" {
		empty, err := storeGit(ctx, r.Root(), "", "", "mktree")
		if err != nil {
			t.Fatal(err)
		}
		c, err := commitTree(ctx, r.Root(), empty, "", "empty")
		if err != nil {
			t.Fatal(err)
		}
		base = c
	}
	tree, err := treeOver(ctx, r.Root(), base, placed)
	if err != nil {
		t.Fatal(err)
	}
	c, err := commitTree(ctx, r.Root(), tree, parent, "another writer")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func placedAt(path, sha string) placed {
	return placed{path: path, entry: entry{mode: "100644", typ: "blob", sha: sha}}
}

// fileOf is the store's file for one record.
func fileOf(t *testing.T, r interface{ file() (file, error) }) file {
	t.Helper()
	f, err := r.file()
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// A record is one new file on the local copy, named by its time, its issue
// and the hash of its content, written without touching HEAD, the index or
// the working tree, and without reaching origin.
func TestAWriteIsOneFileOnTheLocalCopy(t *testing.T) {
	a, _, bare := twoClones(t)
	head := a.Git("rev-parse", "HEAD")
	if w := write(t, a.Root(), hb(handBack("iss-1", at0))); w.Wrote != 1 {
		t.Fatalf("Write = %+v", w)
	}
	if got := a.Git("rev-parse", "HEAD"); got != head {
		t.Fatalf("HEAD moved: %s", got)
	}
	if st := a.Git("status", "--porcelain"); st != "" {
		t.Fatalf("the working tree changed: %q", st)
	}
	files := localFiles(t, a)
	if len(files) != 1 || !regexp.MustCompile(`^handbacks/20261113T070000Z-iss-1-[0-9a-f]{16}\.json$`).MatchString(files[0]) {
		t.Fatalf("local copy holds %v", files)
	}
	if out := gitIn(t, bare, "for-each-ref", BranchRef); out != "" {
		t.Fatalf("a write reached origin: %s", out)
	}
	rec := readLocal(t, a.Root())
	if len(rec.HandBacks) != 1 || rec.HandBacks[0].PR != 12 || rec.HandBacks[0].RunID != "run-2611130703410770" || len(rec.Unreadable) != 0 {
		t.Fatalf("read back %+v", rec)
	}
}

// A store commit names abcd's fixed identity as author and committer, with
// its dates in UTC, whatever the environment of the writing process holds.
func TestAStoreCommitNamesAbcdWhateverTheEnvironmentHolds(t *testing.T) {
	a, _, _ := twoClones(t)
	t.Setenv("GIT_AUTHOR_NAME", "Some Person")
	t.Setenv("GIT_AUTHOR_EMAIL", "person@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Some Committer")
	t.Setenv("GIT_COMMITTER_EMAIL", "committer@example.invalid")
	t.Setenv("TZ", "Asia/Kolkata")
	write(t, a.Root(), hb(handBack("iss-1", at0)))
	raw := a.Git("cat-file", "commit", LocalRef)
	want := regexp.MustCompile(`^(author|committer) abcd spotcheck <spotcheck@abcd\.invalid> [0-9]+ \+0000$`)
	n := 0
	for _, l := range strings.Split(raw, "\n") {
		if strings.HasPrefix(l, "author ") || strings.HasPrefix(l, "committer ") {
			n++
			if !want.MatchString(l) {
				t.Fatalf("the commit names %q", l)
			}
		}
	}
	if n != 2 || strings.Contains(raw, "Some") || strings.Contains(raw, "example.invalid") {
		t.Fatalf("the commit is\n%s", raw)
	}
}

// Writing the same record twice keeps one file and one commit; a merge-gate
// decision asked again over the same head with the same answer is the one
// record, whenever it is asked, whatever origin's copy holds.
func TestAWriteOfARecordAlreadyKeptWritesNothing(t *testing.T) {
	a, _, _ := twoClones(t)
	write(t, a.Root(), hb(handBack("iss-1", at0)))
	first := a.Git("rev-parse", LocalRef)
	if w := write(t, a.Root(), hb(handBack("iss-1", at0))); w.Wrote != 0 || w.Kept != 1 {
		t.Fatalf("second write = %+v; want it kept", w)
	}
	if a.Git("rev-parse", LocalRef) != first {
		t.Fatal("the local copy moved on a record it already held")
	}
	head := strings.Repeat("a", 40)
	if w := write(t, a.Root(), Batch{Decisions: []Decision{gateDecision(7, head, false, at0)}}); w.Wrote != 1 {
		t.Fatalf("decision = %+v", w)
	}
	if w := write(t, a.Root(), Batch{Decisions: []Decision{gateDecision(7, head, false, at0.Add(time.Hour))}}); w.Wrote != 0 || w.Kept != 1 {
		t.Fatalf("the same decision an hour later = %+v; want it kept", w)
	}
	if w := write(t, a.Root(), Batch{Decisions: []Decision{gateDecision(7, head, true, at0)}}); w.Wrote != 1 {
		t.Fatalf("the gate's other answer = %+v; want it written", w)
	}
	// Origin's copy holding the local copy's files changes nothing: what
	// this checkout wrote is decided from its own copy alone.
	a.Git("update-ref", TrackingRef, LocalRef)
	if w := write(t, a.Root(), Batch{Decisions: []Decision{gateDecision(7, head, false, at0.Add(2*time.Hour))}}); w.Wrote != 0 || w.Kept != 1 {
		t.Fatalf("a decision whose earlier record origin also holds = %+v; want it kept", w)
	}
}

// gateFiles are the merge-gate records of pull request pr among files.
func gateFiles(files []string, pr int) []string {
	var out []string
	for _, f := range files {
		if strings.HasPrefix(f, DecisionsDir+"/") && strings.Contains(f, "-pr"+strconv.Itoa(pr)+"-") {
			out = append(out, f)
		}
	}
	return out
}

// A gate asked five times over one head with one answer keeps one record,
// with a push between asks that lays this checkout's records over a tip
// another writer moved: the local copy holds only what this checkout wrote,
// so nothing a push or a fetch brings makes the record be written again, and
// origin's copy holds the one record too.
func TestAGateAskedAgainAcrossPushesKeepsOneRecord(t *testing.T) {
	a, b, bare := twoClones(t)
	ctx := context.Background()
	head := strings.Repeat("f", 40)
	for i := range 5 {
		write(t, b.Root(), hb(handBack("iss-"+strconv.Itoa(10+i), at0)))
		if _, err := push(ctx, b.Root()); err != nil {
			t.Fatal(err)
		}
		w := write(t, a.Root(), Batch{Decisions: []Decision{gateDecision(7, head, false, at0.Add(time.Duration(i)*time.Hour))}})
		if want := min(i, 1); w.Wrote != 1-want || w.Kept != want {
			t.Fatalf("ask %d = %+v", i+1, w)
		}
		if _, err := push(ctx, a.Root()); err != nil {
			t.Fatal(err)
		}
	}
	if got := localFiles(t, a); len(got) != 1 || len(gateFiles(got, 7)) != 1 {
		t.Fatalf("a's local copy holds %v; want its one record alone", got)
	}
	got := remoteFiles(t, bare)
	if len(gateFiles(got, 7)) != 1 || len(got) != 6 {
		t.Fatalf("origin holds %v; want one gate record and b's five", got)
	}
}

// A copy of origin's that cannot be listed changes nothing a write decides:
// a gate asked again over the same head and answer is kept once, and a
// hand-back is written, whatever the tracking ref holds.
func TestACopyOfOriginsThatCannotBeListedChangesNothing(t *testing.T) {
	head := strings.Repeat("9", 40)
	for name, bad := range map[string]func(r *gittest.Repo) string{
		"a ref naming a blob":              func(r *gittest.Repo) string { return strings.TrimSpace(r.Git("hash-object", "-w", "a.txt")) },
		"a decisions entry that is a file": func(r *gittest.Repo) string { return commitFilesAt(t, r, "", map[string]string{DecisionsDir: "x\n"}) },
	} {
		a, _, _ := twoClones(t)
		a.Git("update-ref", TrackingRef, bad(a))
		write(t, a.Root(), Batch{Decisions: []Decision{gateDecision(7, head, true, at0)}})
		if w := write(t, a.Root(), Batch{Decisions: []Decision{gateDecision(7, head, true, at0.Add(time.Hour))}}); w.Wrote != 0 || w.Kept != 1 {
			t.Fatalf("%s: the gate asked again = %+v; want it kept", name, w)
		}
		if w := write(t, a.Root(), hb(handBack("iss-1", at0))); w.Wrote != 1 {
			t.Fatalf("%s: a hand-back = %+v", name, w)
		}
	}
}

// Every worktree of a checkout shares the local copy: a record written from
// a linked worktree is read from the main one and the reverse, and the
// worktree removed takes no record with it.
func TestEveryWorktreeOfACheckoutSharesTheLocalCopy(t *testing.T) {
	a, _, _ := twoClones(t)
	wt := filepath.Join(t.TempDir(), "lane")
	a.Git("worktree", "add", "-q", "-b", "lane", wt)
	write(t, wt, hb(handBack("iss-1", at0)))
	write(t, a.Root(), hb(handBack("iss-2", at0)))
	for _, root := range []string{a.Root(), wt} {
		rec, ok, err := read(context.Background(), root, LocalRef)
		if err != nil || !ok || len(rec.HandBacks) != 2 {
			t.Fatalf("read from %s = %+v, %v, %v; want both records", filepath.Base(root), rec, ok, err)
		}
	}
	a.Git("worktree", "remove", "--force", wt)
	if rec := readLocal(t, a.Root()); len(rec.HandBacks) != 2 {
		t.Fatalf("after the worktree's removal the local copy holds %+v", rec.HandBacks)
	}
}

// A checkout with no origin keeps no record.
func TestAWriteWithNoOriginWritesNothing(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("the base")
	w := Write(context.Background(), r.Root(), hb(handBack("iss-1", at0)))
	if !errors.Is(w.Err, ErrNoOrigin) || w.Wrote != 0 || w.Failed != 0 {
		t.Fatalf("Write = %+v; want ErrNoOrigin", w)
	}
	if out := r.Git("for-each-ref", LocalRef); out != "" {
		t.Fatalf("a ref was written: %s", out)
	}
}

// The cleaning rule, as a property: in cleaned text every "/" has an ASCII
// letter or digit right before it, every word holding a "/" begins with an
// ASCII letter or digit, and the text holds no "\" and no "~";
// cleaning is the same on every machine, so cleaning again changes nothing;
// and the table pins what each layout leaves.
func TestTheCleaningRule(t *testing.T) {
	home, _ := os.UserHomeDir()
	for _, c := range []struct{ in, want string }{
		{"see internal/core/x.go", "see internal/core/x.go"},
		{"failed in /private/var/tmp/run/y.txt now", "failed in <removed> now"},
		{"a single /tmp segment", "a single <removed> segment"},
		{"under " + home + "/work/x.go", "under <removed>"},
		{"under /Users/someone-else/work/x.go", "under <removed>"},
		{"under ~/work and ~someone/x", "under <removed> and <removed>"},
		{"see `/etc/passwd` or `internal/x.go`", "see <removed> or <removed>"},
		// An option glued to an absolute path, as compiler, linker and git
		// error text prints it, goes whole.
		{"cc -I/usr/include -L/opt/lib/x -C/home/someone/repo -Werror", "cc <removed> <removed> <removed> -Werror"},
		{"-I" + home + "/inc", "<removed>"},
		// A repository-relative path inside brackets, quotes or backticks
		// goes too, and so does one with a line number or punctuation written
		// against it.
		{"(internal/x.go) [internal/x.go] \"internal/x.go\" 'a/b' `a/b` <a/b>", "<removed> <removed> <removed> <removed> <removed> <removed>"},
		{"internal/x.go:12: x/abs/path a/b,", "<removed> x/abs/path <removed>"},
		{"internal/x.go:12", "<removed>"},
		{"internal/x.go,", "<removed>"},
		{"(internal/x.go)", "<removed>"},
		// A bare repository-relative path made only of letters, digits, dots,
		// underscores, dashes and slashes stays.
		{"internal/core/x_test.go and docs/how-to/a-b.md", "internal/core/x_test.go and docs/how-to/a-b.md"},
		// A path that begins with a dot goes, a repository-relative one too.
		{"facilitator.mode is auto (repo layer, .abcd/config.json at origin/main)", "facilitator.mode is auto (repo layer, <removed> at <removed>"},
		// So does any path under a top-level folder whose name begins with a dot.
		{"see .github/workflows/ci.yml and .abcd/work/issues/open/iss-1.md", "see <removed> and <removed>"},
		// An option, a name or a quote written straight against a path goes
		// whole, wherever the path starts: an absolute path with a home in it,
		// a dot folder, a quoted or bracketed path.
		{"CFLAGS=-I/Users/someone/include", "<removed>"},
		{"CFLAGS=-I" + home + "/include", "<removed>"},
		{"file=.github/workflows/ci.yml", "<removed>"},
		{`key="internal/x.go" f(internal/x.go) a=b/c`, "<removed> <removed> <removed>"},
		{"PATH=/usr/bin:/bin", "<removed>"},
		{"-I/usr/include", "<removed>"},
		// A word holding a slash and a byte outside ASCII goes.
		{"caf\u00e9/x.go and internal/\u00fcber.go", "<removed> and <removed>"},
		// A word with no slash keeps whatever it holds.
		{"key=value f(x) host.invalid:8080 caf\u00e9", "key=value f(x) host.invalid:8080 caf\u00e9"},
		{"(/abs/path) [/abs] {/abs} </abs> |/abs @/abs \"/abs\" '/abs' =/abs", "<removed> <removed> <removed> <removed> <removed> <removed> <removed> <removed> <removed>"},
		{"at file:///home/x/y and https://example.invalid/a/b", "at <removed> and <removed>"},
		{"the address http://host.invalid:8080/x", "the address <removed>"},
		{"the bare host.invalid:8080/x", "the bare <removed>"},
		{`C:\Users\x\y.txt and C:/Users/x`, "<removed> and <removed>"},
		{"a//b and a/./b and ../up", "<removed> and <removed> and <removed>"},
		{"/Users/jane doe/notes.txt", "<removed> doe/notes.txt"},
		{"red \x1b[31mtext\x1b[0m\nnext\tline\u200b/x", "red [31mtext [0m next line <removed>"},
	} {
		got := clean(c.in)
		if got != c.want {
			t.Errorf("clean(%q) = %q, want %q", c.in, got, c.want)
		}
		if clean(got) != got {
			t.Errorf("clean is not stable on %q: %q", got, clean(got))
		}
		for i := 0; i < len(got); i++ {
			if got[i] == '/' && (i == 0 || !asciiAlnum(got[i-1])) {
				t.Errorf("clean(%q) = %q keeps a \"/\" after %q", c.in, got, got[max(i-1, 0):i])
			}
		}
		for _, w := range strings.Fields(got) {
			if !strings.Contains(w, "/") {
				continue
			}
			if !asciiAlnum(w[0]) {
				t.Errorf("clean(%q) = %q keeps a word holding \"/\" that begins with %q", c.in, got, w[:1])
			}
			for i := 0; i < len(w); i++ {
				if b := w[i]; !asciiAlnum(b) && b != '.' && b != '_' && b != '-' && b != '/' {
					t.Errorf("clean(%q) = %q keeps a word holding \"/\" and the byte %q", c.in, got, b)
				}
			}
		}
		if strings.ContainsAny(got, "\\~\x1b\n\t\u200b") {
			t.Errorf("clean(%q) = %q keeps a character the rule removes", c.in, got)
		}
	}
	long := clean(strings.Repeat("z", 3*maxTextBytes))
	if len(long) > maxTextBytes || !strings.HasSuffix(long, ellipsis) || clean(long) != long {
		t.Fatalf("a long text is %d bytes", len(long))
	}
}

// No home directory, absolute path, address or control character reaches a
// record, and each text is bounded.
func TestARecordCarriesNoPathAndIsBounded(t *testing.T) {
	a, _, _ := twoClones(t)
	home, _ := os.UserHomeDir()
	h := handBack("iss-1", at0)
	h.Reason = "failed in " + home + "/work/x.go and /private/var/tmp/run/y.txt, see https://example.invalid/a/b and .github/workflows/ci.yml \x1b[31mred\x1b[0m " + strings.Repeat("z", 3*maxTextBytes)
	h.Holder = "someone"
	write(t, a.Root(), hb(h))
	got := readLocal(t, a.Root()).HandBacks[0]
	for _, bad := range []string{home, "/private/var", "example.invalid", ".github", "\x1b", "someone"} {
		if bad != "" && strings.Contains(got.Reason+got.Holder, bad) {
			t.Fatalf("the record carries %q: %q", bad, got.Reason)
		}
	}
	if len(got.Reason) > maxTextBytes {
		t.Fatalf("the reason is %d bytes", len(got.Reason))
	}
}

// A store whose ref names anything but a commit is unreadable: a read and a
// write each refuse, and neither reads it as empty.
func TestAnUnreadableStoreIsNeverEmpty(t *testing.T) {
	a, _, _ := twoClones(t)
	blob := strings.TrimSpace(a.Git("hash-object", "-w", "a.txt"))
	a.Git("update-ref", LocalRef, blob)
	if _, _, err := read(context.Background(), a.Root(), LocalRef); err == nil {
		t.Fatal("a read of a ref naming a blob succeeded")
	}
	w := Write(context.Background(), a.Root(), hb(handBack("iss-1", at0)))
	if w.Err == nil || w.Failed != 1 || w.Wrote != 0 {
		t.Fatalf("a write over an unreadable store = %+v", w)
	}
	if got := a.Git("rev-parse", LocalRef); got != blob {
		t.Fatalf("a write over an unreadable store moved the local copy: %s", got)
	}
}

// A store whose handbacks or decisions entry is not a folder cannot be
// written or read, and says which.
func TestAFolderThatIsNotAFolderIsUnreadable(t *testing.T) {
	for _, folder := range []string{HandBacksDir, DecisionsDir} {
		a, _, _ := twoClones(t)
		a.Git("update-ref", LocalRef, commitFilesAt(t, a, "", map[string]string{folder: "not a folder\n"}))
		if _, _, err := read(context.Background(), a.Root(), LocalRef); err == nil || !strings.Contains(err.Error(), folder+" as a blob") {
			t.Fatalf("%s: read = %v", folder, err)
		}
		w := Write(context.Background(), a.Root(), Batch{HandBacks: []HandBack{handBack("iss-1", at0)}, Decisions: []Decision{gateDecision(7, strings.Repeat("b", 40), false, at0)}})
		if w.Err == nil || !strings.Contains(w.Err.Error(), folder+" as a blob") || w.Failed != 2 {
			t.Fatalf("%s: write = %+v", folder, w)
		}
	}
}

// A file another writer put on the branch is read only when it is exactly
// what this package writes under that name: an extra field, another
// version, a name whose hash its content does not give, a file over the size
// bound or one in a folder of its own is named unreadable, and the rest is
// still read.
func TestAFileNotWrittenAsAbcdWritesItIsUnreadable(t *testing.T) {
	a, _, _ := twoClones(t)
	write(t, a.Root(), hb(handBack("iss-1", at0)))
	good := fileOf(t, handBack("iss-2", at0))
	extra := strings.Replace(string(good.data), `"issue": "iss-2"`, `"issue": "iss-3", "extra": 1`, 1)
	v2 := strings.Replace(strings.Replace(string(good.data), `"issue": "iss-2"`, `"issue": "iss-5"`, 1), `"schema_version": 1`, `"schema_version": 2`, 1)
	other := strings.Replace(string(good.data), "the lane kept refusing", "another reason", 1)
	planted := map[string]string{
		"handbacks/" + named("20261113T070000Z-iss-3", []byte(extra)): extra,
		"handbacks/" + named("20261113T070000Z-iss-5", []byte(v2)):    v2,
		// good's name over other content: the hash in the name is not its.
		"handbacks/" + good.name:                                    other,
		"handbacks/" + named("20261113T070000Z-iss-6", []byte("x")): strings.Repeat("x", maxRecordBytes+1),
		"handbacks/deeper/" + good.name:                             string(good.data),
	}
	a.Git("update-ref", LocalRef, commitFilesAt(t, a, a.Git("rev-parse", LocalRef), planted))
	rec := readLocal(t, a.Root())
	if len(rec.HandBacks) != 1 || rec.HandBacks[0].Issue != "iss-1" || len(rec.Unreadable) != len(planted) {
		t.Fatalf("read %d hand-backs, unreadable %v", len(rec.HandBacks), rec.Unreadable)
	}
	if !nameMatches(good.name, good.data) || nameMatches(good.name, []byte(other)) {
		t.Fatal("a name's hash is not checked against its content")
	}
}

// A read past its bound says how many records the store holds and reads
// none of them.
func TestAReadPastItsBoundNamesTheCount(t *testing.T) {
	a, _, _ := twoClones(t)
	write(t, a.Root(), hb(handBack("iss-1", at0), handBack("iss-2", at0), handBack("iss-3", at0)))
	saved := maxReadFiles
	t.Cleanup(func() { maxReadFiles = saved })
	maxReadFiles = 2
	if _, _, err := read(context.Background(), a.Root(), LocalRef); err == nil || !strings.Contains(err.Error(), "holds 3 records, over the 2") {
		t.Fatalf("read = %v", err)
	}
}

// Nothing origin's copy holds makes a record count as kept: a tracking ref
// holding a file of a decision's old-style name, and one of the record's own
// name with other content, leave the local write to be made.
func TestOriginsCopyNeverMakesARecordCountAsKept(t *testing.T) {
	a, _, _ := twoClones(t)
	d := Decision{Kind: KindDrainJudgement, At: at0, Issue: "iss-1", Question: "Does this remedy change what a person sees?", Answer: "no", Model: "model-a", Reason: "it changes neither"}
	f := fileOf(t, d)
	head := strings.Repeat("c", 40)
	g := fileOf(t, gateDecision(7, head, true, at0))
	a.Git("update-ref", TrackingRef, commitFilesAt(t, a, "", map[string]string{
		"decisions/20261113T070000Z-iss-1.json":                  "{}\n",
		"decisions/20261113T065959Z-pr7-cccccccccccc-armed.json": "{}\n",
		f.path(): "other content\n",
		"decisions/" + named("20261113T060000Z-pr7-cccccccccccc-armed", g.data): string(g.data),
	}))
	if w := write(t, a.Root(), Batch{Decisions: []Decision{d, gateDecision(7, head, true, at0)}}); w.Wrote != 2 {
		t.Fatalf("Write = %+v; want both written", w)
	}
	if rec := readLocal(t, a.Root()); len(rec.Decisions) != 2 || len(rec.Unreadable) != 0 {
		t.Fatalf("the local copy holds %+v", rec)
	}
}

// Two hand-backs of one issue in one second that differ are two records,
// and both are kept.
func TestTwoHandBacksOfOneIssueInOneSecondAreBothKept(t *testing.T) {
	a, _, _ := twoClones(t)
	one, two := handBack("iss-1", at0), handBack("iss-1", at0.Add(300*time.Millisecond))
	two.Kind = "closed"
	write(t, a.Root(), hb(one))
	if w := write(t, a.Root(), hb(two)); w.Wrote != 1 {
		t.Fatalf("the second hand-back = %+v", w)
	}
	if rec := readLocal(t, a.Root()); len(rec.HandBacks) != 2 {
		t.Fatalf("the local copy holds %+v", rec.HandBacks)
	}
}

// One write keeps all its records in one commit, and says which records the
// store cannot keep at all.
func TestOneWriteIsOneCommit(t *testing.T) {
	a, _, _ := twoClones(t)
	b := Batch{HandBacks: []HandBack{handBack("iss-1", at0), handBack("iss-2", at0), handBack("iss-3", at0), {Issue: "iss-4", Reason: "no time"}, handBack("not-an-id", at0)},
		Decisions: []Decision{gateDecision(7, strings.Repeat("d", 40), false, at0), gateDecision(0, strings.Repeat("d", 40), false, at0)}}
	w := Write(context.Background(), a.Root(), b)
	if w.Err != nil || w.Wrote != 4 || len(w.Unkeepable) != 3 {
		t.Fatalf("Write = %+v", w)
	}
	if n := a.Git("rev-list", "--count", LocalRef); n != "1" {
		t.Fatalf("the write made %s commits", n)
	}
	if !strings.Contains(strings.Join(w.Unkeepable, "; "), "carries no time") {
		t.Fatalf("unkeepable %v", w.Unkeepable)
	}
}

// Another writer of the checkout that moves the local copy between this
// write's read of it and its move makes this write read it again: both
// records are kept.
func TestAnotherWriterMovingTheLocalCopyIsReadAgain(t *testing.T) {
	a, _, _ := twoClones(t)
	t.Cleanup(func() { beforeMove = nil })
	tries := 0
	beforeMove = func(attempt int) {
		tries = attempt
		if attempt != 1 {
			return
		}
		saved := beforeMove
		beforeMove = nil
		write(t, a.Root(), hb(handBack("iss-2", at0)))
		beforeMove = saved
	}
	write(t, a.Root(), hb(handBack("iss-1", at0)))
	if tries != 2 {
		t.Fatalf("the write moved the copy on try %d", tries)
	}
	if rec := readLocal(t, a.Root()); len(rec.HandBacks) != 2 {
		t.Fatalf("the local copy holds %+v", rec.HandBacks)
	}
}

// standInGit puts a git first on PATH that runs onUpdateRef (a shell
// fragment) for an update-ref and the real git for everything else, with
// ABCD_TEST_REAL_GIT naming the real one. It returns a func that puts PATH
// back, for the part of a test that needs the real git again.
func standInGit(t *testing.T, onUpdateRef string) (restore func()) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip(err)
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nfor a in \"$@\"; do\n  if [ \"$a\" = update-ref ]; then\n" + onUpdateRef + "\n  fi\ndone\nexec \"$ABCD_TEST_REAL_GIT\" \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	path := os.Getenv("PATH")
	t.Setenv("ABCD_TEST_REAL_GIT", real)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+path)
	return func() { os.Setenv("PATH", path) }
}

// hangDeadline is the deadline the hung-git tests give a write, and
// hangMargin what they allow past the bound the store names for a write cut
// off at it (the deadline, stopGrace and readBackTimeout): time for the
// test's own processes to start and be reaped on a loaded machine.
const (
	hangDeadline = 8 * time.Second
	hangMargin   = 3 * time.Second
)

// writeCutOff is a write of one hand-back under hangDeadline, failing the
// test when it returns later than the store's bound and hangMargin.
func writeCutOff(t *testing.T, root string) Written {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), hangDeadline)
	defer cancel()
	start := time.Now()
	w := Write(ctx, root, hb(handBack("iss-1", at0)))
	if took, bound := time.Since(start), hangDeadline+stopGrace+readBackTimeout; took > bound+hangMargin {
		t.Fatalf("the write took %s; its bound is %s, and the test allows %s more", took, bound, hangMargin)
	}
	return w
}

// A git that hangs while it holds the local copy's lock is stopped at the
// write's deadline in a way that lets it remove the lock: the lock is seen
// to exist before the cut-off, the write returns within its bound, saying it
// was not written, no lock remains, and the next write succeeds.
func TestAGitThatHangsLeavesTheStoreUsable(t *testing.T) {
	a, _, _ := twoClones(t)
	lock := filepath.Join(a.Root(), ".git", filepath.FromSlash(LocalRef)+".lock")
	t.Setenv("ABCD_TEST_LOCK", lock)
	t.Setenv("ABCD_TEST_HANG_ROOT", a.Root())
	t.Setenv("ABCD_TEST_HANG_COMMIT", a.Git("rev-parse", "HEAD"))
	seen := filepath.Join(t.TempDir(), "lock-seen")
	t.Setenv("ABCD_TEST_SEEN", seen)
	restore := standInGit(t, `    printf 'start\ncreate `+LocalRef+` %s\nprepare\n' "$ABCD_TEST_HANG_COMMIT" | { cat; sleep 600; } | "$ABCD_TEST_REAL_GIT" -C "$ABCD_TEST_HANG_ROOT" update-ref --stdin &
    while [ ! -e "$ABCD_TEST_LOCK" ]; do sleep 0.05; done
    : > "$ABCD_TEST_SEEN"
    wait
    exit 1`)
	w := writeCutOff(t, a.Root())
	if w.Failed != 1 || w.Wrote != 0 || !errors.Is(w.Err, context.DeadlineExceeded) {
		t.Fatalf("Write = %+v", w)
	}
	restore()
	if _, err := os.Stat(seen); err != nil {
		t.Fatalf("git held no lock before the cut-off: %v", err)
	}
	if _, err := os.Stat(lock); err == nil {
		t.Fatalf("the cut-off left %s", lock)
	}
	if w := write(t, a.Root(), hb(handBack("iss-1", at0))); w.Wrote != 1 {
		t.Fatalf("the next write = %+v", w)
	}
}

// A git that ignores the signal to stop, and leaves a lock, is killed once
// the grace is over: the write returns within its bound, its failure names
// the lock file, the lock is not removed, and the next write fails naming it
// too.
func TestAGitThatIgnoresTheStopLeavesItsLockNamed(t *testing.T) {
	a, _, _ := twoClones(t)
	lock := filepath.Join(a.Root(), ".git", filepath.FromSlash(LocalRef)+".lock")
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ABCD_TEST_LOCK", lock)
	restore := standInGit(t, `    trap '' TERM
    : > "$ABCD_TEST_LOCK"
    sleep 600
    exit 1`)
	// The store's own words, which git never prints: git's own refusal of a
	// held lock names the lock's path too.
	named := "git left the lock file " + LocalRef + ".lock in the repository's git directory, and it is not removed here"
	w := writeCutOff(t, a.Root())
	restore()
	if w.Failed != 1 || w.Err == nil || !strings.Contains(w.Err.Error(), named) {
		t.Fatalf("Write = %+v; want a failure naming the lock", w)
	}
	if _, err := os.Stat(lock); err != nil {
		t.Fatalf("the lock was removed: %v", err)
	}
	w = Write(context.Background(), a.Root(), hb(handBack("iss-1", at0)))
	if w.Failed != 1 || w.Err == nil || !strings.Contains(w.Err.Error(), named) {
		t.Fatalf("the next write = %+v; want a failure naming the lock", w)
	}
	if _, err := os.Stat(lock); err != nil {
		t.Fatalf("the lock was removed: %v", err)
	}
}

// A git that moved the local copy and was cut off before it exited made the
// write: the write reads back where the copy points before it reports, and
// counts the record written.
func TestAWriteGitFinishedAsTheDeadlinePassedIsWritten(t *testing.T) {
	a, _, _ := twoClones(t)
	restore := standInGit(t, `    "$ABCD_TEST_REAL_GIT" "$@" || exit 1
    sleep 600
    exit 0`)
	w := writeCutOff(t, a.Root())
	restore()
	if w.Wrote != 1 || w.Failed != 0 || w.Err != nil {
		t.Fatalf("Write = %+v; want the record counted written", w)
	}
	if rec := readLocal(t, a.Root()); len(rec.HandBacks) != 1 {
		t.Fatalf("the local copy holds %+v", rec)
	}
}

// A git that moved the local copy, after which another writer moved it on,
// and that was then cut off, leaves a write the read-back cannot confirm: it
// is reported as failed, the record stays on the copy, and the next write
// finds it kept and writes nothing, so it is neither lost nor written twice.
func TestACutOffMoveTheCopyMovedOnFromIsKeptOnce(t *testing.T) {
	a, _, _ := twoClones(t)
	t.Setenv("ABCD_TEST_HANG_ROOT", a.Root())
	restore := standInGit(t, `    "$ABCD_TEST_REAL_GIT" "$@" || exit 1
    c=$("$ABCD_TEST_REAL_GIT" -C "$ABCD_TEST_HANG_ROOT" commit-tree -p `+LocalRef+` -m other `+LocalRef+`^{tree}) || exit 1
    "$ABCD_TEST_REAL_GIT" -C "$ABCD_TEST_HANG_ROOT" update-ref `+LocalRef+` "$c" || exit 1
    sleep 600
    exit 0`)
	w := writeCutOff(t, a.Root())
	restore()
	if w.Wrote != 0 || w.Failed != 1 || w.Err == nil {
		t.Fatalf("Write = %+v; want the write reported failed", w)
	}
	if w := write(t, a.Root(), hb(handBack("iss-1", at0))); w.Wrote != 0 || w.Kept != 1 {
		t.Fatalf("the next write = %+v; want the record found kept", w)
	}
	if rec := readLocal(t, a.Root()); len(rec.HandBacks) != 1 {
		t.Fatalf("the local copy holds %+v", rec)
	}
	if n := strings.TrimSpace(gitIn(t, a.Root(), "rev-list", "--count", LocalRef)); n != "2" {
		t.Fatalf("the local copy has %s commits; want this write's and the other writer's", n)
	}
}

// PushBranch and FetchBranch carry the store's one branch and refuse any
// other.
func TestThePushAndFetchCarryTheStoresBranchAlone(t *testing.T) {
	a, _, _ := twoClones(t)
	ctx := context.Background()
	head := a.Git("rev-parse", "HEAD")
	for _, ref := range []string{"refs/heads/abcd-other", "refs/heads/main", "refs/heads/abcd-spotcheck/x"} {
		if err := claim.PushBranch(ctx, a.Root(), head, ref); err == nil {
			t.Errorf("a push to %s was made", ref)
		}
		if _, err := claim.FetchBranch(ctx, a.Root(), ref, "refs/remotes/origin/"+strings.TrimPrefix(ref, "refs/heads/")); err == nil {
			t.Errorf("a fetch of %s was made", ref)
		}
	}
	if _, err := claim.FetchBranch(ctx, a.Root(), BranchRef, "refs/remotes/origin/other"); err == nil {
		t.Error("a fetch into another tracking ref was made")
	}
}

// Two writers of one remote each push their records, and neither drops the
// other's: the second lays its records over the first's tip.
func TestTwoWritersBothKeepTheirRecords(t *testing.T) {
	a, b, bare := twoClones(t)
	ctx := context.Background()
	write(t, a.Root(), hb(handBack("iss-1", at0)))
	write(t, b.Root(), hb(handBack("iss-2", at0)))
	if res, err := push(ctx, a.Root()); err != nil || res.Pushed != 1 {
		t.Fatalf("a's push = %+v, %v", res, err)
	}
	first := gitIn(t, bare, "rev-parse", BranchRef)
	res, err := push(ctx, b.Root())
	if err != nil || res.Pushed != 1 {
		t.Fatalf("b's push = %+v, %v", res, err)
	}
	if got := remoteFiles(t, bare); len(got) != 2 {
		t.Fatalf("origin holds %v, want both records", got)
	}
	// No force: a's commit is still on the branch's history.
	gitIn(t, bare, "merge-base", "--is-ancestor", first, BranchRef)
	// The laid-over commit names abcd in UTC too.
	if raw := gitIn(t, bare, "cat-file", "commit", BranchRef); !strings.Contains(raw, "committer abcd spotcheck <spotcheck@abcd.invalid>") || !strings.Contains(raw, "+0000\n") {
		t.Fatalf("the laid-over commit is\n%s", raw)
	}
	// A push with nothing new pushes nothing.
	if res, err := push(ctx, b.Root()); err != nil || res.Commit != "" {
		t.Fatalf("a second push = %+v, %v; want nothing pushed", res, err)
	}
}

// A writer that loses the race (another pushed between its fetch and its
// push) fetches again and lays its records over the new tip, once.
func TestALostRaceIsTriedOnceMore(t *testing.T) {
	a, b, bare := twoClones(t)
	ctx := context.Background()
	write(t, b.Root(), hb(handBack("iss-2", at0)))
	t.Cleanup(func() { beforePush = nil })
	races := 0
	beforePush = func(attempt int) {
		if attempt != 1 {
			return
		}
		races++
		beforePush = nil
		write(t, a.Root(), hb(handBack("iss-1", at0)))
		if _, err := push(ctx, a.Root()); err != nil {
			t.Fatal(err)
		}
		beforePush = func(int) {}
	}
	res, err := push(ctx, b.Root())
	if err != nil || res.Pushed != 1 || races != 1 {
		t.Fatalf("b's push = %+v, %v (races %d)", res, err, races)
	}
	if got := remoteFiles(t, bare); len(got) != 2 {
		t.Fatalf("origin holds %v, want both records", got)
	}
}

// A writer that loses the race twice gives up, saying so, with its records
// still on its local copy and origin's untouched.
func TestASecondLostRaceKeepsTheRecordsForTheNextPush(t *testing.T) {
	a, b, bare := twoClones(t)
	ctx := context.Background()
	write(t, b.Root(), hb(handBack("iss-2", at0)))
	t.Cleanup(func() { beforePush = nil })
	n := 0
	beforePush = func(int) {
		n++
		saved := beforePush
		beforePush = nil
		write(t, a.Root(), hb(handBack("iss-1", at0.Add(time.Duration(n)*time.Minute))))
		if _, err := push(ctx, a.Root()); err != nil {
			t.Fatal(err)
		}
		beforePush = saved
	}
	if _, err := push(ctx, b.Root()); err == nil {
		t.Fatal("a push that lost twice succeeded")
	}
	beforePush = nil
	if got := remoteFiles(t, bare); len(got) != 2 {
		t.Fatalf("origin holds %v, want a's two records alone", got)
	}
	if rec := readLocal(t, b.Root()); !slices.ContainsFunc(rec.HandBacks, func(h HandBack) bool { return h.Issue == "iss-2" }) {
		t.Fatalf("b's local copy lost its record: %+v", rec.HandBacks)
	}
	if res, err := push(ctx, b.Root()); err != nil || res.Pushed != 1 {
		t.Fatalf("the next push = %+v, %v", res, err)
	}
	if got := remoteFiles(t, bare); len(got) != 3 {
		t.Fatalf("origin holds %v, want all three", got)
	}
}

// Two records of one issue in one second that differ, one from each
// writer, are both pushed; and a file origin holds under a record's name
// whose content the name does not give is named unreadable and replaced by
// the record, which is then found on origin's copy.
func TestAFileOriginHoldsUnderARecordsNameWithOtherContentIsReplaced(t *testing.T) {
	a, b, bare := twoClones(t)
	ctx := context.Background()
	ha, hb2 := handBack("iss-1", at0), handBack("iss-1", at0)
	hb2.Reason = "another reason"
	write(t, a.Root(), hb(ha))
	write(t, b.Root(), hb(hb2))
	if _, err := push(ctx, a.Root()); err != nil {
		t.Fatal(err)
	}
	if res, err := push(ctx, b.Root()); err != nil || res.Pushed != 1 || len(res.Unreadable) != 0 {
		t.Fatalf("b's push = %+v, %v", res, err)
	}
	if got := remoteFiles(t, bare); len(got) != 2 {
		t.Fatalf("origin holds %v, want both of iss-1's records", got)
	}
	// Another writer puts other content under a record's name on origin.
	mine := fileOf(t, handBack("iss-2", at0))
	write(t, b.Root(), hb(handBack("iss-2", at0)))
	a.Git("fetch", "origin", "+"+BranchRef+":"+TrackingRef)
	planted := commitFilesAt(t, a, a.Git("rev-parse", TrackingRef), map[string]string{mine.path(): "{}\n"})
	a.Git("push", "origin", planted+":"+BranchRef)
	res, err := push(ctx, b.Root())
	if err != nil || res.Pushed != 1 || len(res.Unreadable) != 1 || res.Unreadable[0] != mine.path() {
		t.Fatalf("b's push = %+v, %v", res, err)
	}
	rec, ok, err := read(ctx, b.Root(), TrackingRef)
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	found := false
	for _, h := range rec.HandBacks {
		found = found || (h.Issue == "iss-2" && h.Reason == "the lane kept refusing")
	}
	if !found || len(rec.Unreadable) != 0 || len(rec.HandBacks) != 3 {
		t.Fatalf("origin's copy holds %+v", rec)
	}
}

// A branch gone from origin leaves no tracking ref to decide anything: the
// stale ref is removed and the local copy pushed whole.
func TestABranchGoneFromOriginLeavesNoStaleTrackingRef(t *testing.T) {
	a, _, bare := twoClones(t)
	ctx := context.Background()
	write(t, a.Root(), hb(handBack("iss-1", at0)))
	if _, err := push(ctx, a.Root()); err != nil {
		t.Fatal(err)
	}
	gitIn(t, bare, "update-ref", "-d", BranchRef)
	res, err := push(ctx, a.Root())
	if err != nil || res.Commit == "" || res.Pushed != 1 {
		t.Fatalf("the push over a deleted branch = %+v, %v", res, err)
	}
	if got := remoteFiles(t, bare); len(got) != 1 {
		t.Fatalf("origin holds %v", got)
	}
}

// The lay-over compares files at any depth: a file the local copy holds in a
// folder within a folder that origin lacks is pushed, and origin's own at
// any depth are kept.
func TestALayOverComparesFilesAtAnyDepth(t *testing.T) {
	a, b, bare := twoClones(t)
	ctx := context.Background()
	write(t, a.Root(), hb(handBack("iss-1", at0)))
	if _, err := push(ctx, a.Root()); err != nil {
		t.Fatal(err)
	}
	a.Git("update-ref", LocalRef, commitFilesAt(t, a, a.Git("rev-parse", LocalRef), map[string]string{"items/x/one.json": "one\n"}))
	if _, err := push(ctx, a.Root()); err != nil {
		t.Fatal(err)
	}
	write(t, b.Root(), hb(handBack("iss-2", at0)))
	b.Git("update-ref", LocalRef, commitFilesAt(t, b, b.Git("rev-parse", LocalRef), map[string]string{"items/x/two.json": "two\n"}))
	res, err := push(ctx, b.Root())
	if err != nil || res.Pushed != 2 {
		t.Fatalf("b's push = %+v, %v", res, err)
	}
	got := remoteFiles(t, bare)
	for _, want := range []string{"items/x/one.json", "items/x/two.json"} {
		if !slices.Contains(got, want) {
			t.Fatalf("origin holds %v, want %s", got, want)
		}
	}
	if len(got) != 4 {
		t.Fatalf("origin holds %v", got)
	}
}
