package scaffold

import (
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// hostileBytes is how large the attacks that grow with the file are written: all the
// read accepts, less a little for what surrounds each.
const hostileBytes = maxWorkflowBytes - 4<<10

// hostileAllocPerByte bounds what one read may allocate for each byte of the file,
// and hostileAllocFloor what it may allocate whatever the file's size, which is what
// the glob budget can reach from a small file.
const (
	hostileAllocPerByte = 400
	hostileAllocFloor   = 64 << 20
)

// hostileReadSeconds is a hard stop on one read: a bounded read takes well under a
// second even on a loaded machine, and one that is not would otherwise hold the test
// until its timeout.
const hostileReadSeconds = 60

// Words of the refusals, each naming the limit that stopped a file.
const (
	refusedVisits    = "visits more"
	refusedNodes     = "holds more"
	refusedDepth     = "nests deeper"
	refusedDuplicate = "already holds"
	refusedRootMerge = "merge key at the top"
	refusedJobKey    = "key abcd does not read"
	refusedIndent    = "indents deeper"
	refusedComments  = "comment lines"
	refusedDirective = "directive lines"
	refusedEncoding  = "not UTF-8"
)

// hostileWorkflows are files written to cost far more to read than their size, each
// within the size the read accepts, with the words of the limit that refuses it, or
// nothing when the read takes it. Each stays inside every limit ahead of the one it
// names, so it reaches the code it is named for.
func hostileWorkflows() map[string]struct{ src, why string } {
	const good = "on: pull_request\njobs:\n  build:\n    runs-on: x\n"
	lines := func(n int, format string) string {
		var b strings.Builder
		for i := range n {
			fmt.Fprintf(&b, format, i)
		}
		return b.String()
	}
	repeat := func(item string, room int) string { return strings.Repeat(item, room/len(item)) }
	list := func(item string, n int) string { return strings.Repeat(item+",", n-1) + item }
	const wide = maxMatrixValues
	glob := strings.Repeat("?", maxGlobBytes)

	type file = struct{ src, why string }
	files := map[string]file{
		"a job body behind many aliases": {why: refusedVisits, src: "on: pull_request\njobs:\n  a: &x\n" +
			lines(maxWorkflowNodes*9/20, "    k%06d:\n") + lines(3000, "  j%05d: *x\n")},
		"a matrix list behind many aliases": {why: refusedVisits, src: "on: pull_request\njobs:\n  a: &b\n    strategy:\n" +
			"      matrix:\n        k: [" + list("a", wide) + "]\n" + lines(2000, "  j%05d: *b\n")},
		"a strategy behind many aliases": {why: refusedVisits, src: "x-s: &s {matrix: {k: [" + list("a", wide) +
			"]}}\non: pull_request\njobs:\n" + lines(2000, "  j%05d: {strategy: *s}\n")},
		"a needs list behind many aliases": {why: refusedVisits, src: "x-n: &n [" + list("a", maxWorkflowNodes*2/5) +
			"]\non: pull_request\njobs:\n  a: {}\n" + lines(100, "  j%05d: {needs: *n}\n")},
		"long text behind many aliases": {why: refusedVisits, src: "x-l: &l \"" + strings.Repeat("y", maxWorkflowNodes/2) +
			"\"\non: pull_request\njobs:\n" + lines(200, "  j%05d: {name: *l, if: *l, strategy: {matrix: {k: [*l]}}}\n")},

		"one key repeated at the top": {why: refusedDuplicate, src: "{" + repeat("a,", hostileBytes) + "a}"},
		"one key repeated in a job": {why: refusedDuplicate, src: "on: pull_request\njobs:\n  a: {" +
			repeat("k: 1,", hostileBytes) + "k: 1}\n"},
		"one key repeated where the read does not look": {why: refusedDuplicate, src: good +
			"    env: {" + repeat("k: 1,", hostileBytes) + "k: 1}\n"},

		"more nodes than the read takes": {why: refusedNodes, src: "x: [" +
			strings.Repeat("a,", maxWorkflowNodes) + "a]\n" + good},
		"more depth than the read takes": {why: refusedDepth, src: "x: " +
			strings.Repeat("[", maxWorkflowDepth+1) + strings.Repeat("]", maxWorkflowDepth+1) + "\n" + good},

		"a merge key at the top": {why: refusedRootMerge, src: "x-m: &m {x: 1}\n<<: *m\n" + good},
		"merge keys in a job": {why: refusedJobKey, src: "x-m: &m {a: 1}\non: pull_request\njobs:\n  a:\n    <<: [" +
			list("*m", maxWorkflowNodes*2/5) + "]\n"},

		"a glob behind many aliases": {why: refusedVisits, src: "x-g: &g \"" + glob + "\"\non: {pull_request: {branches: [" +
			list("*g", 2000) + "]}}\n" + "jobs: {a: {runs-on: x}}\n"},
		"many globs written out": {why: refusedVisits, src: "on: {pull_request: {branches: [" +
			list(`"`+glob+`"`, 2000) + "]}}\njobs: {a: {runs-on: x}}\n"},
		"globs up to what the read compiles": {src: "x-g: &g \"" + glob + "\"\non: {pull_request: {branches: [" +
			list("*g", maxWorkflowNodes/(2*maxGlobBytes+2)-2) + "]}}\n" + "jobs: {a: {runs-on: x}}\n"},
		"a glob longer than a branch name can be": {src: "on: {pull_request: {branches: [\"" +
			repeat("?", hostileBytes) + "\"]}}\njobs: {a: {runs-on: x}}\n"},

		"comments under a deep block": {why: refusedIndent, src: strings.Repeat("- ", 10_000) + "x\n" +
			repeat("#\n #\n", hostileBytes-20_000) + "- y\n"},
		"comments under a block as deep as the prescan takes": {why: refusedDepth, src: strings.Repeat("- ", maxLineLead) + "x\n" +
			strings.Repeat("#\n #\n", maxCommentLines/2) + "- y\n"},
		"a deep block written line by line": {why: refusedIndent, src: func() string {
			var b strings.Builder
			for i := 0; i <= maxLineLead+1; i++ {
				b.WriteString(strings.Repeat(" ", i) + "a:\n")
			}
			return b.String()
		}()},
		"more comment lines than the read takes": {why: refusedComments, src: good +
			strings.Repeat("#\n #\n", maxCommentLines/2+1)},
		"comment lines ended by carriage returns": {why: refusedComments, src: good +
			strings.Repeat("#\r #\r", maxCommentLines/2+1)},
		"comment lines ended by line separators": {why: refusedComments, src: good +
			strings.Repeat("#\u2028 #\u2028", maxCommentLines/2+1)},
		"comment lines ended by next-line marks": {why: refusedComments, src: good +
			strings.Repeat("#\u0085 #\u0085", maxCommentLines/2+1)},
		"more directives than the read takes": {why: refusedDirective, src: lines(hostileBytes/20, "%%TAG !x%06d! b\n") + "---\n" + good},
		"text in another encoding":            {why: refusedEncoding, src: "\xff\xfe" + repeat("#\x00\n\x00", 1024)},

		"an alias inside what it names": {src: "x-s: &s [*s]\non: pull_request\njobs:\n  build:\n    runs-on: x\n" +
			"  loop:\n    strategy: &t {matrix: *t}\n"},
		"aliases of aliases where the read does not look": {src: func() string {
			var b strings.Builder
			b.WriteString("x-0: &a0 [l,l,l,l,l,l,l,l,l,l]\n")
			for i := 1; i <= 30; i++ {
				fmt.Fprintf(&b, "x-%d: &a%d [%s*a%d]\n", i, i, strings.Repeat(fmt.Sprintf("*a%d,", i-1), 9), i-1)
			}
			return b.String() + good
		}()},
	}
	// Many anchors followed a few times each, so that only the count of everything
	// the read visits stops it.
	var many strings.Builder
	const anchors, items = 40, 2000
	for a := range anchors {
		fmt.Fprintf(&many, "x-%d: &a%d [%sa]\n", a, a, strings.Repeat("a,", items))
	}
	many.WriteString("on: pull_request\njobs:\n  a: {}\n")
	for a := range anchors {
		for j := range 64 {
			fmt.Fprintf(&many, "  j%d-%d: {needs: *a%d}\n", a, j, a)
		}
	}
	files["many anchors, each followed a few times"] = file{why: refusedVisits, src: many.String()}
	return files
}

// TestAHostileWorkflowIsReadInBoundedWork: whatever a file holds, reading it costs
// bounded work, and a file past the read's limits is refused, loudly and with nothing
// proposed, as an unreadable one is, for the limit it was written to reach. Both
// readers are held. The cost is measured in bytes allocated, with a hard stop on time.
func TestAHostileWorkflowIsReadInBoundedWork(t *testing.T) {
	for name, c := range hostileWorkflows() {
		t.Run(name, func(t *testing.T) {
			if len(c.src) > maxWorkflowBytes {
				t.Fatalf("the fixture is %d bytes, past the %d the read accepts, so it would test only that", len(c.src), maxWorkflowBytes)
			}
			repo := writeWorkflows(t, map[string]string{"ci.yml": "on: pull_request\njobs:\n  fine:\n    runs-on: x\n", "hostile.yml": c.src})
			limit := uint64(len(c.src))*hostileAllocPerByte + hostileAllocFloor

			var checks []string
			var err error
			bounded(t, limit, func() { checks, _, err = RequiredChecks(repo, "main", false) })
			if c.why != "" {
				if err == nil {
					t.Fatalf("the read went ahead, with checks %q", checks)
				}
				if len(checks) != 0 {
					t.Errorf("checks = %q; a refused read proposes none", checks)
				}
				if !strings.Contains(err.Error(), "hostile.yml") {
					t.Errorf("the refusal does not name the file: %v", err)
				}
				if !strings.Contains(err.Error(), c.why) {
					t.Errorf("the refusal does not say %q, the limit this file is written to reach: %v", c.why, err)
				}
			} else if err != nil {
				t.Errorf("the read was refused: %v", err)
			} else if !slices.Contains(checks, "fine") {
				t.Errorf("checks = %q, which leaves out the other file's job", checks)
			}
			bounded(t, limit, func() {
				if _, unread := DeriveCIChecks(repo); c.why != "" && !slices.Contains(unread, "hostile.yml") {
					t.Errorf("the scaffold's read does not name the file: %q", unread)
				}
			})
		})
	}
}

// TestRequiredChecksReadsAMergeKeyByWhereItIs: a merge key could add any key where it
// stands, and the read does not expand one. At the top or in a job it could hide what
// decides the answer, so it refuses the read. In a strategy or a matrix it can only
// hide the job's names, so that job is left out and named.
func TestRequiredChecksReadsAMergeKeyByWhereItIs(t *testing.T) {
	const anchor = "x-m: &m\n  if: github.event_name != 'pull_request'\n"
	for name, c := range map[string]struct{ body, why string }{
		"at the top": {anchor + "<<: *m\non: pull_request\njobs:\n  a:\n    runs-on: x\n", refusedRootMerge},
		"in a job":   {anchor + "on: pull_request\njobs:\n  a:\n    <<: *m\n    runs-on: x\n", refusedJobKey},
	} {
		t.Run(name, func(t *testing.T) {
			checks, _, err := RequiredChecks(writeWorkflows(t, map[string]string{"merged.yml": c.body}), "main", false)
			if err == nil {
				t.Fatalf("the read went ahead, with checks %q", checks)
			}
			if !strings.Contains(err.Error(), "merged.yml") || !strings.Contains(err.Error(), c.why) {
				t.Errorf("the refusal does not name the file and say %q: %v", c.why, err)
			}
		})
	}
	repo := writeWorkflows(t, map[string]string{"ci.yml": `x-s: &s {fail-fast: false}
on: pull_request
jobs:
  plain:
    runs-on: x
  in-strategy:
    strategy:
      <<: *s
      matrix:
        os: [a, b]
  in-matrix:
    strategy:
      matrix:
        <<: [a, b]
`})
	wantChecks(t, repo, []string{"plain"}, "in-strategy", "in-matrix")
}

// TestAScaffoldSaysWhichWorkflowItCouldNotRead: the scaffold writes a list an operator
// is told to require, so a file it could not read is named in the runbook, in
// release.yml and in its report, and neither file says no CI was found. A name that
// could carry anything is held out of what it writes.
func TestAScaffoldSaysWhichWorkflowItCouldNotRead(t *testing.T) {
	const broken = "on: pull_request\njobs:\n  a:\n\t\truns-on: x\n"
	render := func(t *testing.T, files map[string]string) (Report, string, string) {
		t.Helper()
		dir := t.TempDir()
		mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/x\n\ngo 1.22\n")
		for name, body := range files {
			mustWrite(t, filepath.Join(dir, ".github", "workflows", name), body)
		}
		gitInit(t, dir, "trunk")
		declarePlugin(t, dir)
		rep, err := Scaffold(Request{RepoRoot: dir})
		if err != nil {
			t.Fatal(err)
		}
		return rep, readFile(t, filepath.Join(dir, filepath.FromSlash(RunbookPath))),
			readFile(t, filepath.Join(dir, filepath.FromSlash(ReleaseYMLPath)))
	}
	t.Run("the only workflow", func(t *testing.T) {
		rep, book, rel := render(t, map[string]string{"broken.yml": broken})
		if !slices.Equal(rep.UnreadWorkflows, []string{"broken.yml"}) {
			t.Errorf("report unread = %q", rep.UnreadWorkflows)
		}
		for what, doc := range map[string]string{"runbook": book, "release.yml": rel} {
			if !strings.Contains(doc, "broken.yml") {
				t.Errorf("the %s does not name the file", what)
			}
			for _, claim := range []string{"found no workflow", "none found", "nothing but review"} {
				if strings.Contains(doc, claim) {
					t.Errorf("the %s says %q of a repository whose workflow it could not read", what, claim)
				}
			}
		}
	})
	t.Run("beside one it read", func(t *testing.T) {
		rep, book, rel := render(t, map[string]string{"broken.yml": broken, "ci.yml": "on: pull_request\njobs:\n  lint:\n    runs-on: x\n"})
		if !slices.Equal(rep.CIChecks, []string{"lint"}) || !slices.Equal(rep.UnreadWorkflows, []string{"broken.yml"}) {
			t.Errorf("report checks %q, unread %q", rep.CIChecks, rep.UnreadWorkflows)
		}
		for what, doc := range map[string]string{"runbook": book, "release.yml": rel} {
			if !strings.Contains(doc, "broken.yml") || !strings.Contains(doc, "lint") {
				t.Errorf("the %s does not name both the file and the check it read", what)
			}
		}
	})
	t.Run("a name that could carry anything", func(t *testing.T) {
		const name = "a\u202eb`\n$(x).yml"
		rep, book, rel := render(t, map[string]string{name: broken})
		if len(rep.UnreadWorkflows) != 1 || rep.UnreadWorkflows[0] == name {
			t.Errorf("report unread = %q", rep.UnreadWorkflows)
		}
		for what, doc := range map[string]string{"runbook": book, "release.yml": rel} {
			for _, bad := range []string{"\u202e", "$(x)", "b`"} {
				if strings.Contains(doc, bad) {
					t.Errorf("the %s carries %q from a file's name", what, bad)
				}
			}
			if strings.Contains(doc, "found no workflow") || strings.Contains(doc, "none found") {
				t.Errorf("the %s claims no workflow was found", what)
			}
		}
	})
	t.Run("nothing unread renders as before", func(t *testing.T) {
		rep, book, rel := render(t, map[string]string{"ci.yml": "on: pull_request\njobs:\n  lint:\n    runs-on: x\n"})
		if len(rep.UnreadWorkflows) != 0 || strings.Contains(book, "could not read") || strings.Contains(rel, "could not read") {
			t.Errorf("a repository read whole speaks of an unread file: %q", rep.UnreadWorkflows)
		}
	})
}

// TestRequiredChecksNamesAWorkflowNamedLikeAScaffoldFile: abcd does not count a file
// with the name of one it scaffolds, so one that runs on a pull request is named: it
// may be the repository's own CI. One that does not run on a pull request has nothing
// to require, and says nothing.
func TestRequiredChecksNamesAWorkflowNamedLikeAScaffoldFile(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{
		"ci.yml":                  "on: pull_request\njobs:\n  lint:\n    runs-on: x\n",
		"release.yml":             "on: pull_request\njobs:\n  own-ci:\n    runs-on: x\n",
		"auto-release.yml":        "on:\n  push:\n    branches: [main]\njobs:\n  roll:\n    runs-on: x\n",
		"dependency-reauthor.yml": "on: pull_request\njobs:\n  [broken\n",
	})
	checks, notes := requiredChecksOf(t, repo, false)
	if !slices.Equal(checks, []string{"lint"}) {
		t.Errorf("checks = %q, want only lint", checks)
	}
	joined := strings.Join(notes, "\n")
	for _, name := range []string{"release.yml", "dependency-reauthor.yml"} {
		if !strings.Contains(joined, name) {
			t.Errorf("the notes do not name %s:\n%s", name, joined)
		}
	}
	if strings.Contains(joined, "auto-release.yml") {
		t.Errorf("a scaffold-named file that runs on no pull request is named:\n%s", joined)
	}
}

// bounded runs read, failing the test if it allocates past limit bytes or outlasts
// the hard stop.
func bounded(t *testing.T, limit uint64, read func()) {
	t.Helper()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	done := make(chan struct{})
	go func() {
		defer close(done)
		read()
	}()
	select {
	case <-done:
	case <-time.After(hostileReadSeconds * time.Second):
		t.Fatalf("the read was still going after %d seconds", hostileReadSeconds)
	}
	runtime.ReadMemStats(&after)
	got := after.TotalAlloc - before.TotalAlloc
	t.Logf("allocated %d KiB", got>>10)
	if got > limit {
		t.Errorf("the read allocated %d MiB, past its %d MiB bound", got>>20, limit>>20)
	}
}

// TestRequiredChecksReadsTheTriggerFromTheParsedFile: a trigger reads the same however
// its YAML is laid out, so a workflow is judged by what GitHub reads, whichever column
// its keys start in and whatever its text says that looks like a key. One with no
// `on:` at all is named rather than dropped.
func TestRequiredChecksReadsTheTriggerFromTheParsedFile(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{
		"json.yml":     `{"on": "pull_request", "jobs": {"json": {"runs-on": "x"}}}`,
		"bom.yml":      "\ufeffon: pull_request\njobs:\n  bom:\n    runs-on: x\n",
		"anchored.yml": "&t on: pull_request\njobs:\n  anchored:\n    runs-on: x\n",
		"aliased.yml":  "name: &e pull_request\non: *e\njobs:\n  aliased:\n    runs-on: x\n",
		"indented.yml": "  on: pull_request\n  jobs:\n    indented:\n      runs-on: x\n",
		"decoy.yml":    "name: \"a\n\n  on: push\n\n  b\"\non: pull_request\njobs:\n  decoy:\n    runs-on: x\n",
		"push.yml":     "\ufeffon: push\njobs:\n  not-a-gate:\n    runs-on: x\n",
		"no-on.yml":    "jobs:\n  no-on:\n    runs-on: x\n",
	})
	checks, notes := requiredChecksOf(t, repo, false)
	if want := []string{"aliased", "anchored", "bom", "decoy", "indented", "json"}; !slices.Equal(checks, want) {
		t.Errorf("checks = %q, want %q; notes %q", checks, want, notes)
	}
	if !strings.Contains(strings.Join(notes, "\n"), "no-on.yml") {
		t.Errorf("the notes do not name the file with no trigger: %q", notes)
	}
}

// TestRequiredChecksRefusesAKeyDefinedTwice: GitHub refuses a workflow that defines a
// key twice in any mapping, so the read does too, wherever in the file it is.
func TestRequiredChecksRefusesAKeyDefinedTwice(t *testing.T) {
	for name, body := range map[string]string{
		"on":        "on: pull_request\non: push\njobs:\n  a:\n    runs-on: x\n",
		"event":     "on:\n  pull_request:\n  pull_request:\njobs:\n  a:\n    runs-on: x\n",
		"filter":    "on:\n  pull_request:\n    types: [opened]\n    types: [opened, synchronize, reopened]\njobs:\n  a:\n    runs-on: x\n",
		"job":       "on: pull_request\njobs:\n  a:\n    runs-on: x\n  a:\n    runs-on: y\n",
		"job's key": "on: pull_request\njobs:\n  a:\n    if: always()\n    if: false\n    runs-on: x\n",
		"matrix":    "on: pull_request\njobs:\n  a:\n    strategy:\n      matrix:\n        os: [a]\n        os: [b]\n",
		"a step's":  "on: pull_request\njobs:\n  a:\n    runs-on: x\n    steps:\n      - run: a\n        run: b\n",
		"an unused": "x: {a: 1, a: 2}\non: pull_request\njobs:\n  a:\n    runs-on: x\n",
	} {
		t.Run(name, func(t *testing.T) {
			if checks, _, err := RequiredChecks(writeWorkflows(t, map[string]string{"twice.yml": body}), "main", false); err == nil {
				t.Errorf("the read went ahead, with checks %q", checks)
			} else if !strings.Contains(err.Error(), "twice.yml") {
				t.Errorf("the refusal does not name the file: %v", err)
			}
		})
	}
}

// TestRequiredChecksEncodesWhatAFileCarries: a file's name and what the parser says
// about it reach a JSON surface as well as the terminal, so neither carries a rune a
// reader cannot see, on a note or on a refusal.
func TestRequiredChecksEncodesWhatAFileCarries(t *testing.T) {
	const hidden = "\u202e"
	name := "a" + hidden + "b.yml"
	repo := writeWorkflows(t, map[string]string{name: "on:\n  pull_request:\n    paths: [x]\njobs:\n  a:\n    runs-on: x\n"})
	_, notes := requiredChecksOf(t, repo, false)
	if joined := strings.Join(notes, "\n"); !strings.Contains(joined, "b.yml") || strings.Contains(joined, hidden) {
		t.Errorf("the note should name the file with its hidden rune encoded: %q", joined)
	}

	for body, why := range map[string]string{
		"on: [\n":              "a file that does not parse",
		"\"\u009b2J\u202e\"\n": "a file that is one hidden-rune scalar",
	} {
		repo := writeWorkflows(t, map[string]string{name: body})
		_, _, err := RequiredChecks(repo, "main", false)
		if err == nil {
			t.Fatalf("%s: the read went ahead", why)
		}
		if strings.ContainsAny(err.Error(), "\u202e\u009b") {
			t.Errorf("%s: the refusal carries a hidden rune: %q", why, err)
		}
		if !strings.Contains(err.Error(), "b.yml") {
			t.Errorf("%s: the refusal does not name the file: %q", why, err)
		}
	}
}

// TestALargeOrdinaryWorkflowIsRead: the limits leave a large real workflow alone: a
// thousand jobs sharing an `if:` and a `needs:` through anchors, a matrix as long as
// GitHub allows, and as many comment lines as the read takes.
func TestALargeOrdinaryWorkflowIsRead(t *testing.T) {
	var b strings.Builder
	b.WriteString("x-if: &cond ${{ always() }}\nx-needs: &gates [first]\non: pull_request\njobs:\n  first:\n    runs-on: x\n")
	want := []string{"first"}
	for i := range 1000 {
		fmt.Fprintf(&b, "  job-%d:\n    runs-on: x\n    steps:\n      - run: make\n", i)
		want = append(want, fmt.Sprintf("job-%d", i))
		b.WriteString("    if: *cond\n    needs: *gates\n")
	}
	b.WriteString("  wide:\n    strategy:\n      matrix:\n        n: [")
	for i := range maxMatrixValues {
		fmt.Fprintf(&b, "%d,", i)
		want = append(want, fmt.Sprintf("wide (%d)", i))
	}
	b.WriteString("]\n" + strings.Repeat("# a comment\n", maxCommentLines))
	checks, notes := requiredChecksOf(t, writeWorkflows(t, map[string]string{"ci.yml": b.String()}), false)
	slices.Sort(want)
	if !slices.Equal(checks, want) {
		t.Errorf("read %d checks, want %d; notes %q", len(checks), len(want), notes)
	}

	// One value past GitHub's limit is a matrix abcd does not expand.
	over := strings.Replace(b.String(), "n: [", "n: [over,", 1)
	_, notes = requiredChecksOf(t, writeWorkflows(t, map[string]string{"ci.yml": over}), false)
	if !strings.Contains(strings.Join(notes, "\n"), "job wide ") {
		t.Errorf("a matrix past the limit is not named: %q", notes)
	}
}

// FuzzReadWorkflow: whatever the file holds, the read answers, and a job it reports
// has an id GitHub accepts.
func FuzzReadWorkflow(f *testing.F) {
	f.Add("on: pull_request\njobs:\n  a:\n    needs: [b]\n    if: always()\n    strategy:\n      matrix:\n        os: [x, y]\n  b: {}\n")
	f.Add("x: &a [*a]\non: &o {pull_request: *o}\njobs: &j {a: *j, <<: *j}\n")
	f.Add("? [a, b]\n: c\n*x: 1\n")
	f.Add("\ufeff%YAML 1.1\n---\non: !!binary x\n--- \njobs: !t\n")
	f.Fuzz(func(t *testing.T, src string) {
		if len(src) > maxWorkflowBytes {
			return
		}
		wf, err := readWorkflow(src)
		if err != nil {
			return
		}
		for _, job := range wf.jobs {
			if !jobIDRe.MatchString(job.id) {
				t.Errorf("read a job with the id %q", job.id)
			}
		}
		if skippable, err := skippableJobs(wf.jobs); err == nil && len(skippable) != len(wf.jobs) {
			t.Errorf("judged %d of %d jobs", len(skippable), len(wf.jobs))
		}
	})
}
