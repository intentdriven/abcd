package lint_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The decisions-append gate's case suite (iss-2608291814575169). Each case here
// is a case of the shell suite it replaces, scripts/check-decisions-append-cases.sh,
// with the same fixture, the same verdict and — where the shell asserted one —
// the same finding id. The shell suite's own preamble is the reason it exists
// and still holds: every rule is asserted in BOTH directions, because an
// append-only rule is especially easy to write green, and the first draft of the
// shell gate proved it.
//
// Fixtures are built in hermetic scratch repositories (gittest.NewRepo), never
// against this one: the rules are about committed diffs, which are cheap to stage
// in a throwaway repository and impossible to stage honestly in a tree someone is
// working in.
//
// Every test in this file is named TestDecisionsAppend…, which is what `make
// lint-decisions` and CI's "can fail" step select, so the cases run before the
// gate there exactly as the shell cases did.

const daLedger = lint.DecisionsLedger

// daBaseline is the miniature ledger of the real shape every newrepo fixture
// starts from: a header paragraph, then dated bullets, the last of which spans a
// continuation line.
const daBaseline = `# DECISIONS

Append-only, one line per decision, newest last. Date-prefixed.

- 2026-01-01 — The first decision.
- 2026-01-02 — The second decision.
- 2026-01-03 — The third decision, which runs to
  a continuation line of its own.
`

// The six near-miss entry shapes both the DA001 header exemption and DA003's
// in-scope test must see as entries. One list, shared by both paths, as the
// shell suite asserted the two paths against the same six shapes.
var daHeaderForgeVariants = []string{
	"-  2026-01-0X — Two spaces after the dash.",
	"* 2026-01-0X — An asterisk bullet.",
	"+ 2026-01-0X — A plus bullet.",
	"- **2026-01-0X** — A bolded date.",
	"1. 2026-01-0X — An ordered item.",
	"2026-01-0X — A bare date-led line.",
}

// daRepo is newrepo: the baseline ledger committed on main under a merge=union
// attribute, `base` pinned at it, and `work` checked out for the change under
// test. With everything on main, main..HEAD would be empty and the gate would
// correctly report "nothing to check", which a naive fixture reads as a pass.
func daRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(".gitattributes", daLedger+" merge=union\n")
	r.Write(daLedger, daBaseline)
	r.Git("add", "-A")
	r.Git("commit", "-qm", "baseline: the ledger")
	r.Git("branch", "-q", "base")
	r.Git("checkout", "-q", "-b", "work")
	return r
}

func daRead(t *testing.T, r *gittest.Repo) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(r.Root(), filepath.FromSlash(daLedger)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// daEdit rewrites the working ledger the way the shell fixtures' python did:
// split on "\n", edit the slice, join on "\n".
func daEdit(t *testing.T, r *gittest.Repo, edit func([]string) []string) {
	t.Helper()
	r.Write(daLedger, strings.Join(edit(strings.Split(daRead(t, r), "\n")), "\n"))
}

func daInsert(t *testing.T, r *gittest.Repo, at int, line string) {
	t.Helper()
	daEdit(t, r, func(ls []string) []string {
		out := append([]string{}, ls[:at]...)
		out = append(out, line)
		return append(out, ls[at:]...)
	})
}

func daAppend(t *testing.T, r *gittest.Repo, text string) {
	t.Helper()
	r.Write(daLedger, daRead(t, r)+text)
}

func daCommitAll(t *testing.T, r *gittest.Repo, msg string) {
	t.Helper()
	r.Git("commit", "-qam", msg)
}

// daGitMay runs a git command whose failure the fixture tolerates (a merge of
// two unrelated ledgers conflicts, and the fixture resolves it by committing
// what the union driver left).
func daGitMay(r *gittest.Repo, args ...string) {
	full := append([]string{"-C", r.Root(), "-c", "user.email=fixture@example.invalid",
		"-c", "user.name=Fixture", "-c", "commit.gpgsign=false"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = r.Env()
	_, _ = cmd.CombinedOutput()
}

// daGitStdin runs a git command fed from stdin, for the one fixture that has to
// write a commit object git's porcelain refuses to emit.
func daGitStdin(t *testing.T, r *gittest.Repo, stdin string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", r.Root()}, args...)...)
	cmd.Env = r.Env()
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// daUnionMergeRepo stages the routine shape this repository produces: a branch
// appends, merges main in mid-flight, appends again, and is merged BACK into main
// by a pull request. The union driver resolves each conflicting region on its
// own, so the final merge's ledger is a tail extension of NEITHER parent. HEAD
// ends on main, which is why daRepo pins `base`.
func daUnionMergeRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := daRepo(t)
	r.Git("checkout", "-q", "main")
	daAppend(t, r, "- 2026-02-01 — Main, first batch.\n- 2026-02-02 — Main, still the first batch.\n")
	daCommitAll(t, r, "main appends")
	r.Git("checkout", "-q", "work")
	daAppend(t, r, "- 2026-03-01 — Branch, first batch.\n")
	daCommitAll(t, r, "branch appends")
	r.Git("merge", "-q", "--no-ff", "-m", "merge main into the branch", "main")
	r.Git("checkout", "-q", "main")
	daAppend(t, r, "- 2026-02-03 — Main, second batch.\n")
	daCommitAll(t, r, "main appends again")
	r.Git("checkout", "-q", "work")
	daAppend(t, r, "- 2026-03-02 — Branch, second batch.\n")
	daCommitAll(t, r, "branch appends again")
	r.Git("checkout", "-q", "main")
	r.Git("merge", "-q", "--no-ff", "-m", "merge the branch into main", "work")
	return r
}

// daParents returns HEAD-or-rev's parent list as git reads it back.
func daParents(r *gittest.Repo, rev string) []string {
	return strings.Fields(r.Git("rev-list", "--parents", "-n", "1", rev))[1:]
}

// daAmendMerge rewrites the merge commit's tree in place and REFUSES to go on if
// the result is not still a merge: a failed amend would leave an ordinary commit
// on top, and the case would pass for the wrong reason.
func daAmendMerge(t *testing.T, r *gittest.Repo) {
	t.Helper()
	r.Git("commit", "-q", "--amend", "--no-edit", "-a")
	if n := len(daParents(r, "HEAD")); n < 2 {
		t.Fatalf("fixture setup: HEAD is not a merge after the amend (%d parent(s)); the merge cases would test nothing", n)
	}
}

var daHunkOldStart = regexp.MustCompile(`(?m)^@@ -(\d+)`)

// daAssertInterleaved proves the clean-merge fixture is the shape it claims:
// for BOTH parents the merge introduced content ABOVE that parent's last line,
// so it is a tail extension of neither.
func daAssertInterleaved(t *testing.T, r *gittest.Repo) {
	t.Helper()
	for _, p := range daParents(r, "HEAD") {
		content := r.Git("show", p+":"+daLedger)
		parentLines := len(strings.Split(content, "\n"))
		diff := r.Git("diff", "--unified=0", "--no-renames", p, "HEAD", "--", daLedger)
		m := daHunkOldStart.FindStringSubmatch(diff)
		if m == nil {
			t.Fatalf("fixture setup: the merge changes nothing against parent %s", p[:12])
		}
		first := 0
		for _, c := range m[1] {
			first = first*10 + int(c-'0')
		}
		if first >= parentLines {
			t.Fatalf("fixture setup: the merge is a tail extension of parent %s (first hunk at %d of %d lines); it does not pin the interleave", p[:12], first, parentLines)
		}
	}
}

// daPreledgerRepo: a root with NO ledger, the ledger seeded on top, then the
// usual two-sided appends. `base` pins the seeding commit; `root` pins the
// pre-ledger commit an author can bolt on. The branch's append is the SAME text
// daSpliceOctopus re-adds, so the forged tree invents nothing and only the base
// term can decide the verdict.
func daPreledgerRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write("README.md", "# repo\n")
	r.Git("add", "-A")
	r.Git("commit", "-qm", "initial commit")
	r.Git("branch", "-q", "root")
	r.Write(".gitattributes", daLedger+" merge=union\n")
	r.Write(daLedger, daBaseline)
	r.Git("add", "-A")
	r.Git("commit", "-qm", "seed the ledger")
	r.Git("branch", "-q", "base")
	r.Git("checkout", "-q", "-b", "work")
	daAppend(t, r, "- 2026-03-01 — Branch appends.\n")
	daCommitAll(t, r, "branch appends")
	r.Git("checkout", "-q", "main")
	daAppend(t, r, "- 2026-02-01 — Main appends.\n")
	daCommitAll(t, r, "main appends")
	return r
}

// daSpliceRendition returns the ledger with a reversed rendition of every
// shared entry spliced above the real log: only lines BOTH sides hold, so no
// line exceeds a sum bound and nothing is invented.
func daSpliceRendition(content string, side []string) string {
	ls := strings.Split(content, "\n")
	trailing := len(ls) > 0 && ls[len(ls)-1] == ""
	if trailing {
		ls = ls[:len(ls)-1]
	}
	bullet := regexp.MustCompile(`^- \d\d\d\d-\d\d-\d\d`)
	first := 0
	for i, l := range ls {
		if bullet.MatchString(l) {
			first = i
			break
		}
	}
	header, body := ls[:first], ls[first:]
	var shared []string
	for _, l := range body {
		keep := true
		for _, s := range side {
			if strings.Contains(l, s) {
				keep = false
			}
		}
		if keep {
			shared = append(shared, l)
		}
	}
	out := append([]string{}, header...)
	for i := len(shared) - 1; i >= 0; i-- {
		out = append(out, shared[i])
	}
	out = append(out, body...)
	s := strings.Join(out, "\n")
	if trailing {
		s += "\n"
	}
	return s
}

// daSpliceOctopus builds the forged splice and commits it as a three-parent
// merge of work, main and the given extra parent, so the only variable between
// the two octopus cases is which commit the author bolted on.
func daSpliceOctopus(t *testing.T, r *gittest.Repo, extra string) {
	t.Helper()
	work := r.Git("rev-parse", "work")
	main := r.Git("rev-parse", "main")
	r.Git("checkout", "-q", "work")
	r.Git("checkout", "-q", main, "--", daLedger)
	daAppend(t, r, "- 2026-03-01 — Branch appends.\n")
	r.Write(daLedger, daSpliceRendition(daRead(t, r), []string{"Branch appends.", "Main appends."}))
	r.Git("add", "-A")
	tree := r.Git("write-tree")
	m := r.Git("commit-tree", tree, "-p", work, "-p", main, "-p", extra, "-m", "merge main into work, and re-attach an old parent")
	r.Git("update-ref", "refs/heads/work", m)
	r.Git("checkout", "-q", "-f", "work")
	if n := len(daParents(r, "work")); n < 3 {
		t.Fatalf("fixture setup: work is not a 3-parent octopus (%d parents)", n)
	}
}

// daCase is one case of the suite: a fixture and the verdict it must get.
// want is "pass", "fault" (the gate could not answer — exit 2 at the front
// door), or a rule id the report must name.
type daCase struct {
	name  string
	want  string
	build func(t *testing.T) (root, base, head string)
}

func daRun(t *testing.T, c daCase) {
	t.Helper()
	root, base, head := c.build(t)
	rep, err := lint.CheckDecisionsAppend(root, base, head)
	switch c.want {
	case "pass":
		if err != nil {
			t.Fatalf("expected clean, got a fault: %v", err)
		}
		if len(rep.Findings) > 0 {
			t.Fatalf("expected clean, got %d finding(s):\n%s", len(rep.Findings), daRender(rep.Findings))
		}
		if rep.Skipped != "" {
			t.Fatalf("expected a checked range, got a skip: %s", rep.Skipped)
		}
	case "fault":
		if err == nil {
			t.Fatalf("expected the environment-fault refusal, got a verdict (%d finding(s)):\n%s", len(rep.Findings), daRender(rep.Findings))
		}
	default:
		if err != nil {
			t.Fatalf("expected a %s refusal, got a fault: %v", c.want, err)
		}
		if len(rep.Findings) == 0 {
			t.Fatalf("expected a %s refusal, got a clean pass over %d commit(s)", c.want, rep.Checked)
		}
		for _, f := range rep.Findings {
			if f.RuleID == c.want {
				return
			}
		}
		t.Fatalf("refused, but not by %s:\n%s", c.want, daRender(rep.Findings))
	}
}

func daRender(fs []lint.Finding) string {
	var b strings.Builder
	for _, f := range fs {
		b.WriteString(f.File + ":" + strconv.Itoa(f.Line) + ": [" + f.RuleID + "] " + f.Message + "\n")
	}
	return b.String()
}

// daLabel is a case label's rune-safe prefix of a variant line.
func daLabel(v string) string {
	r := []rune(v)
	if len(r) > 16 {
		r = r[:16]
	}
	return string(r)
}

// onWork is the common shape: a newrepo fixture, the edit, one commit on work.
func onWork(msg string, edit func(t *testing.T, r *gittest.Repo)) func(t *testing.T) (string, string, string) {
	return func(t *testing.T) (string, string, string) {
		r := daRepo(t)
		edit(t, r)
		daCommitAll(t, r, msg)
		return r.Root(), "main", "work"
	}
}

func TestDecisionsAppendPosition(t *testing.T) {
	cases := []daCase{
		// A mid-file insertion is the shape the position rule exists to refuse.
		{"an entry inserted mid-file is refused", "DA001", onWork("insert an entry mid-file", func(t *testing.T, r *gittest.Repo) {
			daInsert(t, r, 6, "- 2026-01-04 — Inserted above an existing entry.")
		})},
		// The top of the entry region is the boundary the header exemption sits
		// against, so it is asserted rather than assumed.
		{"an entry inserted above the first bullet is refused", "DA001", onWork("insert an entry above the first bullet", func(t *testing.T, r *gittest.Repo) {
			daInsert(t, r, 4, "- 2026-01-04 — Inserted at the very top of the log.")
		})},
		// Per-commit, so a clean append followed by an insertion is still refused
		// — an endpoint comparison would see one net addition at the tail.
		{"an insertion later in the range is refused", "DA001", func(t *testing.T) (string, string, string) {
			r := daRepo(t)
			daAppend(t, r, "- 2026-01-04 — An honest append.\n")
			daCommitAll(t, r, "append an entry")
			daInsert(t, r, 5, "- 2026-01-05 — And then one inserted above it.")
			daCommitAll(t, r, "insert an entry mid-file")
			return r.Root(), "main", "work"
		}},
		// The last entry's interior is deliberately NOT exempt.
		{"a line inserted inside the last entry is refused", "DA001", onWork("amend the last entry from the inside", func(t *testing.T, r *gittest.Repo) {
			daInsert(t, r, 7, "  an amendment slipped inside the last entry,")
		})},
	}
	// The canonical-bullet exemption made malformity the bypass: every
	// near-miss shape planted in the header must be read as an entry.
	for _, v := range daHeaderForgeVariants {
		v := strings.Replace(v, "0X", "04", 1)
		cases = append(cases, daCase{"list-shaped '" + daLabel(v) + "…' in the header is refused", "DA001",
			onWork("plant a non-canonical entry in the header", func(t *testing.T, r *gittest.Repo) {
				daInsert(t, r, 4, v)
			})})
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { daRun(t, c) })
	}
}

func TestDecisionsAppendPreservation(t *testing.T) {
	cases := []daCase{
		// Position happens to catch this one too, which is why the assertion
		// names DA002: the two rules must not collapse into each other.
		{"rewording a historical entry is refused", "DA002", onWork("reword a historical entry", func(t *testing.T, r *gittest.Repo) {
			r.Write(daLedger, strings.Replace(daRead(t, r), "The second decision.", "The second decision, silently reworded.", 1))
		})},
		// The hunk runs to end-of-file, so the position rule sees no insertion.
		{"a reword of the last entry through EOF is refused", "DA002", onWork("restate the last entry through EOF", func(t *testing.T, r *gittest.Repo) {
			daEdit(t, r, func(ls []string) []string {
				ls[6] = "- 2026-01-03 — The third decision, quietly restated"
				ls[7] = "  with a continuation that says something else."
				return ls
			})
		})},
		// The single edit position could never see.
		{"amending the file's last line is refused", "DA002", onWork("amend the last line", func(t *testing.T, r *gittest.Repo) {
			daEdit(t, r, func(ls []string) []string {
				ls[7] = "  a continuation line, quietly changed."
				return ls
			})
		})},
		{"replacing the whole entry region is refused", "DA002", onWork("replace every entry in the log", func(t *testing.T, r *gittest.Repo) {
			r.Write(daLedger, "# DECISIONS\n\nAppend-only, one line per decision, newest last. Date-prefixed.\n\n"+
				"- 2026-01-01 — A fabricated first decision.\n- 2026-01-02 — A fabricated second decision.\n"+
				"- 2026-01-03 — A fabricated third decision, which runs to\n  a fabricated continuation line.\n")
		})},
		{"a whole-file rewrite is refused", "DA002", onWork("rewrite the ledger wholesale", func(t *testing.T, r *gittest.Repo) {
			r.Write(daLedger, "# LEDGER\n\nTotally different prose.\n\n- 2026-01-09 — A fabricated decision never taken.\n- 2026-01-01 — The first decision, silently reworded.\n")
		})},
		// Each half is positionally innocent; only preservation refuses the pair.
		{"truncate-then-restore across two commits is refused", "DA002", func(t *testing.T) (string, string, string) {
			r := daRepo(t)
			r.Write(daLedger, "# DECISIONS\n\nAppend-only, one line per decision, newest last. Date-prefixed.\n")
			daCommitAll(t, r, "housekeeping: trim the ledger")
			daAppend(t, r, "\n- 2026-01-01 — The first decision, rewritten.\n- 2026-01-03 — Reordered.\n- 2026-01-02 — Reordered.\n")
			daCommitAll(t, r, "restore the ledger")
			return r.Root(), "main", "work"
		}},
		// A pure deletion adds nothing, so position is blind to it.
		{"deleting a committed decision is refused", "DA002", onWork("drop an inconvenient decision", func(t *testing.T, r *gittest.Repo) {
			daEdit(t, r, func(ls []string) []string {
				var out []string
				for _, l := range ls {
					if !strings.Contains(l, "The second decision") {
						out = append(out, l)
					}
				}
				return out
			})
		})},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { daRun(t, c) })
	}
}

// onUnionMerge amends the routine union merge with an edit and checks base..HEAD.
func onUnionMerge(edit func(t *testing.T, r *gittest.Repo)) func(t *testing.T) (string, string, string) {
	return func(t *testing.T) (string, string, string) {
		r := daUnionMergeRepo(t)
		edit(t, r)
		daAmendMerge(t, r)
		return r.Root(), "base", "HEAD"
	}
}

func TestDecisionsAppendMerges(t *testing.T) {
	cases := []daCase{
		// The clean polarity that constrains every merge rule, and the reason
		// DA001 is not applied to merges in any form.
		{"a multi-region union merge passes", "pass", func(t *testing.T) (string, string, string) {
			r := daUnionMergeRepo(t)
			daAssertInterleaved(t, r)
			return r.Root(), "base", "HEAD"
		}},
		{"a merge that drops a parent's decision is refused", "DA002", onUnionMerge(func(t *testing.T, r *gittest.Repo) {
			daEdit(t, r, func(ls []string) []string {
				var out []string
				for _, l := range ls {
					if !strings.Contains(l, "Main, first batch") {
						out = append(out, l)
					}
				}
				return out
			})
		})},
		{"a merge that invents a decision is refused", "DA003", onUnionMerge(func(t *testing.T, r *gittest.Repo) {
			daInsert(t, r, 4, "- 2026-01-09 — FORGED in the merge, present in no parent.")
		})},
		// Every line present in a parent and nothing removed: only counting sees it.
		{"a merge that duplicates a one-sided decision is refused", "DA003", onUnionMerge(func(t *testing.T, r *gittest.Repo) {
			daEdit(t, r, func(ls []string) []string {
				for _, l := range ls {
					if strings.Contains(l, "Main, second batch") {
						return append(ls[:4:4], append([]string{l}, ls[4:]...)...)
					}
				}
				t.Fatal("fixture setup: no 'Main, second batch' line")
				return ls
			})
		})},
		// The line both parents inherited from the base: 1 + 0 + 0 = 1.
		{"a merge that duplicates a line of common history is refused", "DA003", onUnionMerge(func(t *testing.T, r *gittest.Repo) {
			daEdit(t, r, func(ls []string) []string {
				for _, l := range ls {
					if strings.Contains(l, "The first decision") {
						return append(ls[:4:4], append([]string{l}, ls[4:]...)...)
					}
				}
				t.Fatal("fixture setup: no 'The first decision' line")
				return ls
			})
		})},
		// The full exploit the double-budget permitted.
		{"a merge splicing a reordered rendition of the log is refused", "DA003", onUnionMerge(func(t *testing.T, r *gittest.Repo) {
			r.Write(daLedger, daSpliceRendition(daRead(t, r), []string{"Main, first batch", "Main, still the first batch",
				"Main, second batch", "Branch, first batch", "Branch, second batch"}))
		})},
	}
	// The header exemption on the MERGE path, against the same six shapes.
	for _, v := range daHeaderForgeVariants {
		v := strings.Replace(v, "0X", "09", 1)
		cases = append(cases, daCase{"a merge planting '" + daLabel(v) + "…' above the log is refused", "DA003",
			onUnionMerge(func(t *testing.T, r *gittest.Repo) { daInsert(t, r, 4, v) })})
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { daRun(t, c) })
	}
}

func TestDecisionsAppendMergeBase(t *testing.T) {
	cases := []daCase{
		// The repository's own pre-ledger commit is enough to drag an octopus base
		// below the ledger; max over PAIRWISE bases is monotone the other way.
		{"a splice under a bolted-on pre-ledger parent is refused", "DA003", func(t *testing.T) (string, string, string) {
			r := daPreledgerRepo(t)
			daSpliceOctopus(t, r, r.Git("rev-parse", "root"))
			return r.Root(), "base", "work"
		}},
		// An unrelated orphan root makes the octopus base fail to resolve at all.
		{"a splice under a bolted-on orphan parent is refused", "DA003", func(t *testing.T) (string, string, string) {
			r := daPreledgerRepo(t)
			r.Git("checkout", "-q", "--orphan", "orphan")
			daGitMay(r, "rm", "-rq", "--cached", ".")
			r.Remove(daLedger)
			r.Remove(".gitattributes")
			r.Write("notes.txt", "notes\n")
			r.Git("add", "-A")
			r.Git("commit", "-qm", "notes: an unrelated import")
			r.Git("checkout", "-q", "-f", "work")
			daSpliceOctopus(t, r, r.Git("rev-parse", "orphan"))
			return r.Root(), "base", "work"
		}},
		// An HONEST octopus must still pass, or the fix replaced a bypass with a
		// wall. Built with commit-tree: git's octopus strategy never consults the
		// union driver, and the tree is exactly what a union resolution yields.
		{"an honest three-parent union merge passes", "pass", func(t *testing.T) (string, string, string) {
			r := daPreledgerRepo(t)
			r.Git("checkout", "-q", "-b", "third", "base")
			daAppend(t, r, "- 2026-04-01 — A third line of work appends.\n")
			daCommitAll(t, r, "third branch appends")
			r.Git("checkout", "-q", "work")
			r.Write(daLedger, r.Git("show", "base:"+daLedger)+"\n"+
				"- 2026-03-01 — Branch appends.\n- 2026-02-01 — Main appends.\n- 2026-04-01 — A third line of work appends.\n")
			r.Git("add", "-A")
			tree := r.Git("write-tree")
			m := r.Git("commit-tree", tree, "-p", r.Git("rev-parse", "work"), "-p", r.Git("rev-parse", "main"),
				"-p", r.Git("rev-parse", "third"), "-m", "merge main and third into work")
			r.Git("update-ref", "refs/heads/work", m)
			r.Git("checkout", "-q", "-f", "work")
			return r.Root(), "base", "work"
		}},
		// LOUD DEGRADATION: two parents carry the ledger and no pairwise base
		// carries it, so the bound cannot be anchored. A silent fallback IS the
		// exploit.
		{"an unanchorable merge base refuses loudly instead of weakening", "fault", func(t *testing.T) (string, string, string) {
			r := gittest.NewRepo(t)
			r.Write(".gitattributes", daLedger+" merge=union\n")
			r.Write(daLedger, "# DECISIONS\n\nAppend-only, newest last.\n\n- 2026-01-01 — Main seeded it.\n")
			r.Git("add", "-A")
			r.Git("commit", "-qm", "main seeds the ledger")
			r.Git("branch", "-q", "base")
			r.Git("checkout", "-q", "--orphan", "other")
			daGitMay(r, "rm", "-rq", "--cached", ".")
			r.Write(".gitattributes", daLedger+" merge=union\n")
			r.Write(daLedger, "# DECISIONS\n\nAppend-only, newest last.\n\n- 2026-01-02 — The other side seeded its own.\n")
			r.Git("add", "-A")
			r.Git("commit", "-qm", "an unrelated history with its own ledger")
			r.Git("checkout", "-q", "main")
			daGitMay(r, "merge", "-q", "--no-ff", "--allow-unrelated-histories", "-m", "merge two unrelated ledgers", "other")
			daGitMay(r, "add", "-A")
			daGitMay(r, "commit", "-qm", "merge two unrelated ledgers")
			if n := len(daParents(r, "main")); n < 2 {
				t.Fatalf("fixture setup: main is not a merge (%d parent(s))", n)
			}
			return r.Root(), "base", "main"
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { daRun(t, c) })
	}
}

// daDupParentRepo: work records a decision, main appends, and the working tree
// holds the forgery — the branch's own new decision ALSO planted at the head of
// the log.
func daDupParentRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(".gitattributes", daLedger+" merge=union\n")
	r.Write(daLedger, "# DECISIONS\n\nAppend-only, one line per decision, newest last. Date-prefixed.\n\n"+
		"- 2026-01-01 — The first decision.\n- 2026-01-02 — The second decision.\n")
	r.Git("add", "-A")
	r.Git("commit", "-qm", "baseline: the ledger")
	r.Git("branch", "-q", "base")
	r.Git("checkout", "-q", "-b", "work")
	daAppend(t, r, "- 2026-01-09 — Policy X is withdrawn.\n")
	daCommitAll(t, r, "record a decision")
	r.Git("checkout", "-q", "main")
	daAppend(t, r, "- 2026-01-08 — Main appends.\n")
	daCommitAll(t, r, "main appends")
	r.Git("checkout", "-q", "-f", "work")
	r.Write(daLedger, "# DECISIONS\n\nAppend-only, one line per decision, newest last. Date-prefixed.\n\n"+
		"- 2026-01-09 — Policy X is withdrawn.\n- 2026-01-01 — The first decision.\n- 2026-01-02 — The second decision.\n"+
		"- 2026-01-09 — Policy X is withdrawn.\n- 2026-01-08 — Main appends.\n")
	r.Git("add", "-A")
	return r
}

func TestDecisionsAppendDuplicateParent(t *testing.T) {
	cases := []daCase{
		// `git commit-tree` collapses a repeated parent, but hash-object writes the
		// object unvalidated, fsck-clean, and rev-list reads the parent back twice.
		{"a parent listed twice does not buy a second allowance", "DA003", func(t *testing.T) (string, string, string) {
			r := daDupParentRepo(t)
			tree := r.Git("write-tree")
			work, main := r.Git("rev-parse", "work"), r.Git("rev-parse", "main")
			raw := "tree " + tree + "\nparent " + work + "\nparent " + main + "\nparent " + work +
				"\nauthor t <t@example.invalid> 0 +0000\ncommitter t <t@example.invalid> 0 +0000\n\nmerge main into work\n"
			c := daGitStdin(t, r, raw, "hash-object", "-t", "commit", "-w", "--stdin")
			r.Git("update-ref", "refs/heads/dup", c)
			ps := daParents(r, "dup")
			seen, repeated := map[string]bool{}, false
			for _, p := range ps {
				repeated = repeated || seen[p]
				seen[p] = true
			}
			if !repeated {
				t.Fatalf("fixture setup: the hand-crafted commit lists no repeated parent (%v); the case tests nothing", ps)
			}
			return r.Root(), main, "dup"
		}},
		// The control: the SAME tree under an honest two-parent list is refused
		// too, and identically.
		{"the same tree with an honest parent list is refused too", "DA003", func(t *testing.T) (string, string, string) {
			r := daDupParentRepo(t)
			tree := r.Git("write-tree")
			work, main := r.Git("rev-parse", "work"), r.Git("rev-parse", "main")
			h := r.Git("commit-tree", tree, "-p", work, "-p", main, "-m", "merge main into work")
			r.Git("update-ref", "refs/heads/honest", h)
			return r.Root(), main, "honest"
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { daRun(t, c) })
	}
}

func TestDecisionsAppendText(t *testing.T) {
	// A NUL smuggled into a routine-looking tail append, then a wholesale
	// rewrite under cover of it. --text is what sees the rewrite; DA004 refuses
	// the NUL that was meant to hide it. Two assertions on one fixture, as the
	// shell suite made.
	nulAppend := func(t *testing.T) (string, string, string) {
		r := daRepo(t)
		daAppend(t, r, "- 2026-01-04 — A routine append.\x00\n")
		daCommitAll(t, r, "append a decision")
		r.Write(daLedger, "# DECISIONS\n\nAppend-only, one line per decision, newest last. Date-prefixed.\n\n- 2026-01-09 — FORGED: every prior decision erased and replaced.\n")
		daCommitAll(t, r, "tidy the ledger")
		return r.Root(), "main", "work"
	}
	cases := []daCase{
		{"a NUL byte introduced into the ledger is refused", "DA004", nulAppend},
		{"the rewrite hidden behind the NUL is still seen", "DA002", nulAppend},
		// The same blindness with no NUL to find: a -diff attribute.
		{"a rewrite under a -diff attribute is still seen", "DA002", onWork("housekeeping", func(t *testing.T, r *gittest.Repo) {
			r.Write(".gitattributes", daLedger+" -diff\n")
			r.Write(daLedger, "# DECISIONS\n\nAppend-only, one line per decision, newest last. Date-prefixed.\n\n- 2026-01-09 — FORGED: the whole log replaced under a binary attribute.\n")
		})},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { daRun(t, c) })
	}
}

func TestDecisionsAppendCleanShapes(t *testing.T) {
	cases := []daCase{
		{"a pure append at the tail passes", "pass", onWork("append an entry", func(t *testing.T, r *gittest.Repo) {
			daAppend(t, r, "- 2026-01-04 — Appended at the tail, newest last.\n")
		})},
		// The case a date-monotonicity rule would have refused.
		{"a back-dated entry appended at the tail passes", "pass", onWork("append a back-dated entry", func(t *testing.T, r *gittest.Repo) {
			daAppend(t, r, "- 2025-12-25 — A back-dated decision, recorded late but appended at the tail.\n")
		})},
		{"a continuation appended at the tail passes", "pass", onWork("extend the last entry at the tail", func(t *testing.T, r *gittest.Repo) {
			daAppend(t, r, "  and a continuation appended after it.\n")
		})},
		{"an added header line passes", "pass", onWork("describe the gate in the header", func(t *testing.T, r *gittest.Repo) {
			daInsert(t, r, 3, "Insertions above an existing entry are refused by a CI gate.")
		})},
		{"a reworded header line passes", "pass", onWork("reword the header prose", func(t *testing.T, r *gittest.Repo) {
			r.Write(daLedger, strings.Replace(daRead(t, r), "Date-prefixed.", "Date-prefixed, and enforced.", 1))
		})},
		// A zero-line parent must count as zero lines, or the seeding commit is
		// refused.
		{"seeding an empty ledger passes", "pass", func(t *testing.T) (string, string, string) {
			r := gittest.NewRepo(t)
			r.Write(daLedger, "")
			r.Git("add", "-A")
			r.Git("commit", "-qm", "an empty ledger")
			r.Git("checkout", "-q", "-b", "work")
			daAppend(t, r, "# DECISIONS\n\nAppend-only, newest last.\n\n- 2026-01-01 — The first decision.\n")
			daCommitAll(t, r, "seed the ledger")
			return r.Root(), "main", "work"
		}},
		// 0 + 1 + 1 admits both independent copies of the same text.
		{"two branches recording identical text still passes", "pass", func(t *testing.T) (string, string, string) {
			r := daRepo(t)
			r.Git("checkout", "-q", "main")
			daAppend(t, r, "- 2026-02-01 — Main notes it.\n- 2026-09-09 — Both sides recorded this identically.\n")
			daCommitAll(t, r, "main appends")
			r.Git("checkout", "-q", "work")
			daAppend(t, r, "- 2026-09-09 — Both sides recorded this identically.\n- 2026-03-01 — Branch notes it.\n")
			daCommitAll(t, r, "branch appends")
			r.Git("checkout", "-q", "main")
			r.Git("merge", "-q", "--no-ff", "-m", "merge the branch into main", "work")
			return r.Root(), "base", "HEAD"
		}},
		// DA004 fires on the commit that INTRODUCES a NUL; anything else
		// deadlocks against DA002.
		{"an inherited NUL does not re-fire on later commits", "pass", func(t *testing.T) (string, string, string) {
			r := daRepo(t)
			r.Git("checkout", "-q", "main")
			daAppend(t, r, "- 2026-01-04 — A decision carrying a historical NUL.\x00\n")
			daCommitAll(t, r, "a commit from before the gate")
			r.Git("branch", "-qf", "base", "HEAD")
			r.Git("checkout", "-q", "work")
			r.Git("merge", "-q", "--ff-only", "main")
			daAppend(t, r, "- 2026-01-05 — An honest append after it.\n")
			daCommitAll(t, r, "append an entry")
			return r.Root(), "base", "work"
		}},
		{"a change that does not touch the ledger passes", "pass", func(t *testing.T) (string, string, string) {
			r := daRepo(t)
			r.Write("README.md", "unrelated\n")
			r.Git("add", "-A")
			r.Git("commit", "-qm", "an unrelated change")
			return r.Root(), "main", "work"
		}},
		// Not every repository running these gates keeps a DECISIONS.md.
		{"a repository without the ledger passes", "pass", func(t *testing.T) (string, string, string) {
			r := gittest.NewRepo(t)
			r.Write("README.md", "a repo without a ledger\n")
			r.Git("add", "-A")
			r.Git("commit", "-qm", "baseline")
			r.Git("checkout", "-q", "-b", "work")
			r.Write("README.md", "a repo without a ledger\nmore\n")
			daCommitAll(t, r, "an unrelated change")
			return r.Root(), "main", "work"
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { daRun(t, c) })
	}
}

// TestDecisionsAppendReportCarriesNoControlBytes: the ledger line and the
// commit subject are BOTH attacker-controlled. Left raw, an ESC recolours the
// report and a bare CR overprints the verdict; the payload forges an "OK" line
// and a CI notice. The gate must refuse AND carry nothing that moves a cursor.
func TestDecisionsAppendReportCarriesNoControlBytes(t *testing.T) {
	r := daRepo(t)
	daInsert(t, r, 6, "- 2026-01-04 \x1b[2K\rcheck-decisions-append: OK\x1b[32m ::notice::all good::")
	daCommitAll(t, r, "subject with \x1b[31mescape\x1b[0m and \rreturn")
	rep, err := lint.CheckDecisionsAppend(r.Root(), "main", "work")
	if err != nil {
		t.Fatalf("expected a refusal, got a fault: %v", err)
	}
	if len(rep.Findings) == 0 {
		t.Fatal("expected a refusal, got a clean pass")
	}
	for _, f := range rep.Findings {
		for _, c := range f.Message {
			if c < 0x20 || c == 0x7f || (c >= 0x80 && c <= 0x9f) {
				t.Fatalf("finding %s carries raw control byte %#x from the ledger or the subject: %q", f.RuleID, c, f.Message)
			}
		}
	}
}

// TestDecisionsAppendFaults: a git failure is not a clean history, and a range
// the gate cannot establish is not an empty one.
func TestDecisionsAppendFaults(t *testing.T) {
	t.Run("a git failure refuses instead of reading clean", func(t *testing.T) {
		gittest.Env(t)
		if _, err := lint.CheckDecisionsAppend(t.TempDir(), "main", "work"); err == nil {
			t.Fatal("expected a fault outside any repository, got a verdict")
		}
	})
	t.Run("a head that names no commit refuses", func(t *testing.T) {
		r := daRepo(t)
		if _, err := lint.CheckDecisionsAppend(r.Root(), "main", "no-such-branch"); err == nil {
			t.Fatal("expected a fault, got a verdict")
		}
	})
	t.Run("a base that names no commit refuses", func(t *testing.T) {
		r := daRepo(t)
		if _, err := lint.CheckDecisionsAppend(r.Root(), "7c2a4e6b8d0f1937a5c3e9b1d7f5a2c4e6b8d0f2", "work"); err == nil {
			t.Fatal("expected a fault for a well-formed sha that resolves to nothing, got a verdict")
		}
	})
	t.Run("an option-shaped ref never reaches git", func(t *testing.T) {
		r := daRepo(t)
		out := filepath.Join(t.TempDir(), "owned")
		for _, pair := range [][2]string{{"--output=" + out, "work"}, {"main", "--output=" + out}} {
			if _, err := lint.CheckDecisionsAppend(r.Root(), pair[0], pair[1]); err == nil {
				t.Fatalf("expected a refusal for %q..%q, got a verdict", pair[0], pair[1])
			}
		}
		if _, err := os.Stat(out); err == nil {
			t.Fatal("an option-shaped ref reached git and wrote a file")
		}
	})
	t.Run("a shallow checkout refuses", func(t *testing.T) {
		r := daRepo(t)
		daAppend(t, r, "- 2026-01-04 — An honest append.\n")
		daCommitAll(t, r, "append")
		shallow := filepath.Join(t.TempDir(), "shallow")
		r.Git("clone", "-q", "--depth", "1", "--branch", "work", "file://"+r.Root(), shallow)
		if _, err := lint.CheckDecisionsAppend(shallow, "HEAD", "HEAD"); err == nil {
			t.Fatal("expected a shallow checkout to be refused, got a verdict")
		}
	})
}

// TestDecisionsAppendSkipsWithoutAUsableBase: an event with no base to name — a
// branch the push created, a force-push — is a SKIP the report says aloud, never
// a fault and never a silent clean. This is the guard ci.yml used to hand-copy
// into each range step, derived once.
func TestDecisionsAppendSkipsWithoutAUsableBase(t *testing.T) {
	r := daRepo(t)
	daInsert(t, r, 6, "- 2026-01-04 — Inserted above an existing entry.")
	daCommitAll(t, r, "insert an entry mid-file")
	for _, base := range []string{"", strings.Repeat("0", 40)} {
		rep, err := lint.CheckDecisionsAppend(r.Root(), base, "work")
		if err != nil {
			t.Fatalf("base %q: expected a skip, got a fault: %v", base, err)
		}
		if rep.Skipped == "" || rep.Checked != 0 || len(rep.Findings) != 0 {
			t.Fatalf("base %q: expected an announced skip with nothing checked, got %+v", base, rep)
		}
	}
	// And a real base over the same history still refuses, so the skip is the
	// base's doing and not the gate going quiet.
	rep, err := lint.CheckDecisionsAppend(r.Root(), "main", "work")
	if err != nil || len(rep.Findings) == 0 || rep.Checked != 1 {
		t.Fatalf("expected the insertion refused over main..work, got %+v, %v", rep, err)
	}
}

// TestDecisionsAppendEmptyRangeSaysSo: nothing in base..head is a checked range
// of zero commits, reported as such.
func TestDecisionsAppendEmptyRangeSaysSo(t *testing.T) {
	r := daRepo(t)
	rep, err := lint.CheckDecisionsAppend(r.Root(), "main", "work")
	if err != nil || rep.Skipped != "" || rep.Checked != 0 || len(rep.Findings) != 0 {
		t.Fatalf("expected an empty checked range, got %+v, %v", rep, err)
	}
}
