package lint_test

import (
	"errors"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// recordsOnly runs scripts/records-only.sh in r and returns its line and
// whether it classified HEAD records-only (exit 0) or the full run (exit 1).
func recordsOnly(t *testing.T, r *gittest.Repo) (string, bool) {
	t.Helper()
	script, err := filepath.Abs(filepath.Join("..", "..", "..", "scripts", "records-only.sh"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", script)
	cmd.Dir, cmd.Env = r.Root(), r.Env()
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return string(out), true
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		return string(out), false
	}
	t.Fatalf("records-only.sh: %v\n%s", err, out)
	return "", false
}

// recordsRepo is a repository whose origin/main is its first commit, holding
// a code file, a test that names one record by its path, and that record.
func recordsRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write("internal/x/x.go", "package x\n")
	r.Write("internal/x/x_test.go", "package x\n\nconst fixture = \".abcd/work/fixture.md\"\n")
	r.Write(".abcd/work/fixture.md", "a record a test reads\n")
	r.Write(".abcd/work/issues/open/iss-1.md", "one\n")
	r.Commit("base")
	r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
	return r
}

// TestRecordsOnlyClassifiesARecordTierChange (iss-2610251555094324): a change
// whose every path lies in .abcd/development/ or .abcd/work/, named by no
// test, is records-only, and the line says why.
func TestRecordsOnlyClassifiesARecordTierChange(t *testing.T) {
	r := recordsRepo(t)
	r.Write(".abcd/work/issues/open/iss-2.md", "two\n")
	r.Write(".abcd/development/intents/drafts/itd-1.md", "an intent\n")
	r.Write(".abcd/work/issues/open/iss-1.md", "one, edited\n")
	r.Commit("records")
	base := r.Git("rev-parse", "origin/main")
	out, ok := recordsOnly(t, r)
	if want := "records-only: all 3 changed paths since the merge base with origin/main, " + base + ", are in the record tiers, and no test names one\n"; !ok || out != want {
		t.Fatalf("a records change = %v, %q; want records-only, %q", ok, out, want)
	}
	// A deleted record is a changed path in the tiers too.
	r.Remove(".abcd/work/issues/open/iss-2.md")
	r.Commit("drop one")
	if out, ok := recordsOnly(t, r); !ok {
		t.Fatalf("a records change with a delete = full run, %q; want records-only", out)
	}
}

// TestRecordsOnlyRunsEverythingElseInFull: the classifier is fail-closed. A
// code change mixed into a records change, a path outside the two tiers (the
// docs and the conventions file included, which CI's own classifier treats
// differently), a record a test names, a file moved out of code into a tier,
// an empty diff, and a checkout with no origin default branch each take the
// full run, and the line names why.
func TestRecordsOnlyRunsEverythingElseInFull(t *testing.T) {
	for _, c := range []struct {
		name string
		edit func(r *gittest.Repo)
		want string
	}{
		{"code mixed in", func(r *gittest.Repo) {
			r.Write(".abcd/work/issues/open/iss-2.md", "two\n")
			r.Write("internal/x/x.go", "package x\n\nvar Y = 1\n")
		}, "full run: internal/x/x.go is outside the record tiers"},
		{"docs", func(r *gittest.Repo) { r.Write("docs/guide.md", "a page\n") }, "full run: docs/guide.md is outside the record tiers"},
		{"conventions file", func(r *gittest.Repo) { r.Write("AGENTS.md", "rules\n") }, "full run: AGENTS.md is outside the record tiers"},
		{"another .abcd tier", func(r *gittest.Repo) { r.Write(".abcd/config.json", "{}\n") }, "full run: .abcd/config.json is outside the record tiers"},
		{"a record a test names", func(r *gittest.Repo) { r.Write(".abcd/work/fixture.md", "edited\n") },
			"full run: .abcd/work/fixture.md is named by a test (its file name fixture.md), so it is that test's fixture"},
		{"a record a test names by its file name", func(r *gittest.Repo) {
			r.Write("internal/x/y_test.go", "package x\n\nvar parts = []string{\".abcd\", \"work\", \"personas.json\"}\n")
			r.Commit("a test that builds the path")
			r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
			r.Write(".abcd/work/personas.json", "[]\n")
		}, "full run: .abcd/work/personas.json is named by a test (its file name personas.json), so it is that test's fixture"},
		{"a record a test finds by its id", func(r *gittest.Repo) {
			r.Write("internal/x/z_test.go", "package x\n\nconst glob = \"shipped/itd-7-*.md\"\n")
			r.Commit("a test that globs for a record by its id")
			r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
			r.Write(".abcd/development/intents/shipped/itd-7-a-real-intent.md", "an intent\n")
		}, "full run: .abcd/development/intents/shipped/itd-7-a-real-intent.md is named by a test (its record id itd-7), so it is that test's fixture"},
		{"a record a test finds by its bare number", func(r *gittest.Repo) {
			r.Write("internal/x/n_test.go", "package x\n\nconst id = \"adr-261015072933\"\n")
			r.Commit("a test that globs for an ADR by its number")
			r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
			r.Write(".abcd/development/decisions/adrs/261015072933-a-real-decision.md", "a decision\n")
		}, "full run: .abcd/development/decisions/adrs/261015072933-a-real-decision.md is named by a test (its record id 261015072933), so it is that test's fixture"},
		{"a local branch named like origin's", func(r *gittest.Repo) {
			r.Write("internal/x/x.go", "package x\n\nvar Y = 1\n")
			r.Commit("unmerged code")
			r.Git("branch", "origin/main")
			r.Write(".abcd/work/issues/open/iss-2.md", "two\n")
		}, "full run: internal/x/x.go is outside the record tiers"},
		{"moved out of code", func(r *gittest.Repo) { r.Git("mv", "internal/x/x.go", ".abcd/work/x.go") },
			"full run: internal/x/x.go is outside the record tiers"},
		{"empty", func(r *gittest.Repo) { r.Git("commit", "-q", "--allow-empty", "-m", "nothing") },
			"full run: the diff against the merge base with origin/main is empty"},
		{"no origin", func(r *gittest.Repo) {
			r.Write(".abcd/work/issues/open/iss-2.md", "two\n")
			r.Git("update-ref", "-d", "refs/remotes/origin/main")
		}, "full run: there is no origin/main to diff against"},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := recordsRepo(t)
			c.edit(r)
			if r.Git("status", "--porcelain") != "" {
				r.Commit(c.name)
			}
			out, ok := recordsOnly(t, r)
			if ok || !strings.HasPrefix(out, c.want) {
				t.Fatalf("records-only = %v, %q; want the full run, %q", ok, out, c.want)
			}
		})
	}
}

// TestRecordsOnlyDiffsAgainstOriginsDefaultBranch: the merge base is taken
// with origin/HEAD when it names the default branch, so a repository whose
// default branch is not main is judged against it.
func TestRecordsOnlyDiffsAgainstOriginsDefaultBranch(t *testing.T) {
	r := recordsRepo(t)
	r.Git("update-ref", "refs/remotes/origin/trunk", "HEAD")
	r.Git("update-ref", "-d", "refs/remotes/origin/main")
	r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
	r.Write(".abcd/work/issues/open/iss-2.md", "two\n")
	r.Commit("records")
	if out, ok := recordsOnly(t, r); !ok || !strings.Contains(out, "merge base with origin/trunk") {
		t.Fatalf("against origin/trunk = %v, %q; want records-only against it", ok, out)
	}
}

// TestRecordsOnlyKeepsControlBytesOffTheTerminal: the line can name a
// changed path, which a writer chose, so a control byte in it never reaches
// the terminal.
func TestRecordsOnlyKeepsControlBytesOffTheTerminal(t *testing.T) {
	for _, c := range []struct{ name, file, want string }{
		{"a C0 escape", "x\x1b[2Jy.go", "x[2Jy.go is outside"},
		{"a C1 control sequence introducer", "x\u009b2Jy.go", "x2Jy.go is outside"},
		{"a bidirectional override", "x\u202egp.go", "xgp.go is outside"},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := recordsRepo(t)
			r.Write(c.file, "package x\n")
			r.Commit("hostile")
			out, ok := recordsOnly(t, r)
			for _, b := range []byte(strings.TrimSuffix(out, "\n")) {
				if b < 0x20 || b > 0x7e {
					t.Fatalf("a hostile path printed byte %#x: %q", b, out)
				}
			}
			if ok || !strings.Contains(out, c.want) {
				t.Fatalf("a hostile path = %v, %q; want the full run naming %q", ok, out, c.want)
			}
		})
	}
}

// TestRecordsOnlyLetsACitationInACommentPass: a test that cites a record's id
// in prose (itd-8, or itd-8)) does not make that record its fixture; only the
// id followed by a dash, the start of its file name, does.
func TestRecordsOnlyLetsACitationInACommentPass(t *testing.T) {
	r := recordsRepo(t)
	r.Write("internal/x/c_test.go", "package x\n\n// The rule itd-8 states (see itd-8).\n")
	r.Commit("a test that cites an intent")
	r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
	r.Write(".abcd/development/intents/drafts/itd-8-a-cited-intent.md", "an intent\n")
	r.Commit("the intent")
	if out, ok := recordsOnly(t, r); !ok {
		t.Fatalf("a record a test only cites = full run, %q; want records-only", out)
	}
}

// TestRecordsOnlyReadsANewlineInAPathAsOnePath: the changed paths are read
// NUL-delimited, so a path holding a newline is one path, judged whole: a code
// file whose name holds a newline is not split into names that could each look
// like a record. A record whose name holds one cannot be searched for whole
// (git grep splits a pattern at a newline), so it takes the full run too.
func TestRecordsOnlyReadsANewlineInAPathAsOnePath(t *testing.T) {
	r := recordsRepo(t)
	r.Write("x\n.abcd/work/d.md", "package x\n")
	r.Commit("a code file named with a newline")
	if out, ok := recordsOnly(t, r); ok || !strings.HasPrefix(out, "full run: x.abcd/work/d.md is outside") {
		t.Fatalf("a code file named with a newline = %v, %q; want the full run naming it whole", ok, out)
	}
	r = recordsRepo(t)
	r.Write(".abcd/work/a\nb.md", "a record\n")
	r.Commit("a record with a newline in its name")
	if out, ok := recordsOnly(t, r); ok || !strings.HasPrefix(out, "full run: .abcd/work/ab.md holds a newline in its name") {
		t.Fatalf("a record named with a newline = %v, %q; want the full run, saying why", ok, out)
	}
}

// TestRecordsOnlyRunsAnUncommittedEditInFull: the change is judged by its
// commits, so a tree that differs from HEAD (an edited code file, an untracked
// one, an edit an index flag hides) takes the full run, or the edit's tests
// would be skipped.
func TestRecordsOnlyRunsAnUncommittedEditInFull(t *testing.T) {
	for _, c := range []struct {
		name string
		edit func(r *gittest.Repo)
	}{
		{"an edited code file", func(r *gittest.Repo) { r.Write("internal/x/x.go", "package x\n\nvar Y = 1\n") }},
		{"an untracked code file", func(r *gittest.Repo) { r.Write("internal/x/z.go", "package x\n") }},
		{"a hidden edit", func(r *gittest.Repo) {
			r.Git("update-index", "--skip-worktree", "internal/x/x.go")
			r.Write("internal/x/x.go", "package x\n\nvar Y = 1\n")
		}},
		{"an assume-unchanged edit", func(r *gittest.Repo) {
			r.Git("update-index", "--assume-unchanged", "internal/x/x.go")
			r.Write("internal/x/x.go", "package x\n\nvar Y = 1\n")
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := recordsRepo(t)
			r.Write(".abcd/work/issues/open/iss-2.md", "two\n")
			r.Commit("records")
			c.edit(r)
			want := "full run: the working tree differs from HEAD"
			if out, ok := recordsOnly(t, r); ok || !strings.HasPrefix(out, want) {
				t.Fatalf("records-only = %v, %q; want the full run, %q", ok, out, want)
			}
		})
	}
}

// TestPreflightRecordsRuleRunsPrechecksTests: the Makefile carries two
// preflight rules, the full one first, so every reader of the first rule (the
// load gates, the race lane, the receipt) reads the full run. The full rule
// waits at the load gate as one whole run and runs its gates and stages as
// preflight-stages, inside the same branch (itd-2610262127138979). The second,
// taken for a records-only change, runs every gate the full run runs (its
// prerequisites are preflight-stages's) with no load gate, checks the verdict
// again before anything else, runs precheck's build, vet and test lines and
// nothing else, and mints a receipt that says it is records-only.
func TestPreflightRecordsRuleRunsPrechecksTests(t *testing.T) {
	makefile := readRepoFile(t, filepath.Join("..", "..", ".."), "Makefile")
	lines := strings.Split(makefile, "\n")
	var rules []int
	stages := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "preflight:") && !strings.HasPrefix(l, "preflight: export ") {
			rules = append(rules, i)
		}
		if strings.HasPrefix(l, "preflight-stages:") && stages < 0 {
			stages = i
		}
	}
	if len(rules) != 2 {
		t.Fatalf("the Makefile declares %d preflight rules, want 2 (the full run, then the records-only run)", len(rules))
	}
	if stages < rules[0] || stages > rules[1] {
		t.Fatalf("preflight-stages is declared at line %d, want it in the full rule's branch, between the two preflight rules (lines %d and %d)", stages+1, rules[0]+1, rules[1]+1)
	}
	gates := func(l, target string) []string { return strings.Fields(strings.TrimPrefix(l, target+":")) }
	if got, want := gates(lines[rules[1]], "preflight"), gates(lines[stages], "preflight-stages"); !slices.Equal(got, want) {
		t.Errorf("the records-only rule's gates differ from every gate the full run runs (preflight-stages's):\n%s\n%s", lines[rules[1]], lines[stages])
	}
	var body []string
	for _, l := range lines[rules[1]+1:] {
		if !strings.HasPrefix(l, "\t") {
			break
		}
		body = append(body, strings.TrimPrefix(l, "\t"))
	}
	if !slices.Contains(lines[stages:rules[1]], "else") || !slices.Contains(lines[rules[1]:], "endif") {
		t.Fatal("the two preflight rules are not the two branches of one conditional")
	}
	precheck, ok := makeRecipe(makefile, "precheck")
	if !ok {
		t.Fatal("Makefile declares no `precheck:` recipe")
	}
	var want []string
	for _, l := range strings.Split(precheck, "\n") {
		want = append(want, strings.TrimPrefix(l, "\t"))
	}
	if len(body) != len(want)+3 {
		t.Fatalf("the records-only recipe runs %q; want the verdict checked, the skip announced, precheck's %q, then the mint", body, want)
	}
	if body[0] != "@scripts/records-only.sh" || !strings.HasPrefix(body[1], `@echo "preflight: a records-only change, so the unit and race stages are skipped`) {
		t.Errorf("the records-only recipe does not check the verdict and say what it skips before it runs anything: %q", body[:2])
	}
	if got := body[2 : len(body)-1]; !slices.Equal(got, want) {
		t.Errorf("the records-only recipe runs %q; want precheck's lines, %q", got, want)
	}
	if last := body[len(body)-1]; last != `@scripts/preflight-receipt.sh mint "$(PREFLIGHT_BEGAN)" records-only` {
		t.Errorf("the records-only recipe ends %q; want the records-only receipt minted", last)
	}
}
