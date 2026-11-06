package landing

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// namingReturn is a return whose verdict line opens with verdict and names sha.
func namingReturn(sha, who, verdict string) string {
	return "Reviewed head: " + sha[:9] + "\n\n# " + who + "\n\n### Verdict\n\n" + verdict + " " + sha + " — nothing survived.\n"
}

// A verdict names the head it judges: a line that opens as one and names no head, or
// another, refuses the return, and nothing is recorded.
func TestAVerdictNamesTheHeadItJudges(t *testing.T) {
	repo, sha := reviewRepo(t)
	other := strings.Repeat("0", len(sha))
	first := "Reviewed head: " + sha[:9] + "\n\n"
	for name, text := range map[string]string{
		"a return that ends on an indented heading and a passing word": first + "### Analysis\n\nThe diff's context reads:\n\n ### Verdict\n \n SHIP\n",
		"a return that ends on a heading and a passing word":           first + "### Analysis\n\nThe fixture reads:\n\n### Verdict\n\nSHIP\n",
		"a refusal under a qualified heading, beside an indented one":  first + "### Final verdict\n\n**BLOCK**\n\nThe diff's context reads:\n\n ### Verdict\n \n SHIP\n",
		"a refusal under a qualified heading, beside a heading":        first + "### Final verdict\n\n**BLOCK**\n\nThe fixture reads:\n\n### Verdict\n\nSHIP\n",
		"a verdict naming another head":                                first + "### Verdict\n\nSHIP " + other + "\n",
		"a verdict naming the head by its abbreviation":                first + "### Verdict\n\nSHIP " + sha[:9] + "\n",
		"a head named above a verdict that names another":              first + "### Verdict\n\nHead: " + sha + "\nSHIP " + other + "\n",
		"another head named directly under the heading":                first + "### Verdict\n\nHead: " + other + "\nSHIP\n",
		"a head named below the verdict":                               first + "### Verdict\n\nSHIP\n\nHead: " + sha + "\n",
		"a head named on the first line alone":                         "Reviewed head: " + sha + "\n\n### Verdict\n\nSHIP\n",
		"a bare name under the heading":                                first + "### Verdict\n\n" + sha + "\nSHIP\n",
		"a head named under the heading and no verdict after it":       first + "### Verdict\n\nHead: " + sha + "\n",
		"a passing word two lines under the head":                      first + "### Verdict\n\nHead: " + sha + "\nThe fixture reads SHIP\nSHIP\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := RecordReview(repo.Root(), sha, []ReviewReturn{
				{Name: "a.md", Text: text},
				{Name: "b.md", Text: namingReturn(sha, "b", "APPROVE")},
			})
			if err == nil || !strings.Contains(err.Error(), "a.md") {
				t.Fatalf("err = %v; want a.md refused", err)
			}
			if got, _ := reviewReceipt(repo.Root(), sha); got != "" {
				t.Fatalf("a refused pair left a receipt at %s", got)
			}
		})
	}
}

// The head goes on the verdict line or directly under the heading, in the markup a
// reviewer writes.
func TestAVerdictNamesItsHeadOnItsLineOrDirectlyUnderTheHeading(t *testing.T) {
	repo, sha := reviewRepo(t)
	for name, section := range map[string]string{
		"on the line":                  "SHIP " + sha,
		"emphasis around the word":     "**SHIP** " + sha,
		"emphasis around both":         "**SHIP " + sha + "**",
		"code around the name":         "**SHIP** `" + sha + "`",
		"a reason after":               "**SHIP** " + sha + " — nothing survived.",
		"a list item":                  "- **SHIP** " + sha + " — nothing survived.",
		"a colon before the name":      "SHIP: " + sha,
		"a dash before the name":       "**SHIP** — " + sha,
		"brackets around the name":     "SHIP (" + sha + ")",
		"labelled under the heading":   "Head: " + sha + "\nSHIP",
		"emphasised label, then a gap": "**Head:** `" + sha + "`\n\n**SHIP** — nothing survived.",
		"under the heading and on it":  "Head: " + sha + "\n**SHIP** " + sha,
		"abbreviated under the name":   "Head: " + sha + "\n**SHIP** " + sha[:9],
	} {
		t.Run(name, func(t *testing.T) {
			text := "Reviewed head: " + sha[:9] + "\n\n### Findings\n\nNone.\n\n### Verdict\n\n" + section + "\n"
			if _, err := RecordReview(repo.Root(), sha, []ReviewReturn{
				{Name: "a.md", Text: text},
				{Name: "b.md", Text: namingReturn(sha, "b", "APPROVE")},
			}); err != nil {
				t.Fatalf("the return was refused: %v", err)
			}
		})
	}
}

// A line that opens a verdict in any common shape counts, a qualifier included, so a
// return holding one beside its Verdict section is refused, naming it.
func TestAQualifiedVerdictLineCounts(t *testing.T) {
	repo, sha := reviewRepo(t)
	for _, line := range []string{
		"### Final verdict",
		"### Final Verdict: BLOCK",
		"## Overall verdict",
		"# My verdict",
		"#### The reviewers' final overall verdict",
		"> ### Final verdict",
		"**Verdict:**",
		"**Final verdict**",
		"**Final verdict (round 2):** BLOCK",
		"**Final verdict:** BLOCK",
		"**Final verdict**: BLOCK",
		"__Overall verdict__ BLOCK",
		"*My verdict*: BLOCK",
		"- **Final verdict:** BLOCK",
		"> **Overall verdict** — BLOCK",
		"1. **Final verdict:** BLOCK",
		"Final verdict: BLOCK",
		"final verdict - BLOCK",
		"Overall verdict — BLOCK",
		"Review verdict = BLOCK",
		"| Final verdict | BLOCK |",
		"| Reviewer | Verdict |",
		"My verdict is BLOCK",
		"The final verdict is FIX FIRST",
		"Overall verdict (round 2): BLOCK",
		"Final verdict\n=====",
		"Security verdict\n-----",
		"Final verdict",
		"### Verdicts",
		"1. Verdict: BLOCK",
		" ### Verdict",
		"   ### Verdict",
	} {
		t.Run(line, func(t *testing.T) {
			_, err := RecordReview(repo.Root(), sha, []ReviewReturn{
				{Name: "a.md", Text: namingReturn(sha, "a", "SHIP") + "\n" + line + "\n"},
				{Name: "b.md", Text: namingReturn(sha, "b", "APPROVE")},
			})
			if err == nil || !strings.Contains(err.Error(), "a.md") || !strings.Contains(err.Error(), "Verdict") {
				t.Fatalf("err = %v; want a.md refused for a second Verdict line", err)
			}
			if got, _ := reviewReceipt(repo.Root(), sha); got != "" {
				t.Fatalf("a refused pair left a receipt at %s", got)
			}
		})
		t.Run("alone "+line, func(t *testing.T) {
			text := "Reviewed head: " + sha[:9] + "\n\n" + line + "\n\nSHIP " + sha + "\n"
			if _, err := RecordReview(repo.Root(), sha, []ReviewReturn{
				{Name: "a.md", Text: text},
				{Name: "b.md", Text: namingReturn(sha, "b", "APPROVE")},
			}); err == nil || !strings.Contains(err.Error(), "### Verdict") {
				t.Fatalf("err = %v; want no exact heading named", err)
			}
		})
	}
}

// A line that only mentions the word opens nothing, and a line break of any kind ends
// a line.
func TestAVerdictLineIsReadLineByLine(t *testing.T) {
	repo, sha := reviewRepo(t)
	for _, prose := range []string{
		"The verdict is plain, and **a verdict** in a sentence is none.",
		"An earlier note: the loop records the verdict",
		"Whether the verdict stands is the gate's to say.",
		"- The verdict reader was read line by line.",
		"The reader takes a verdict from one section, and a reviewer writes it there.",
	} {
		t.Run(prose, func(t *testing.T) {
			if _, err := RecordReview(repo.Root(), sha, []ReviewReturn{
				{Name: "a.md", Text: "Reviewed head: " + sha[:9] + "\n\n### Analysis\n\n" + prose + "\n\n### Verdict\n\nSHIP " + sha + "\n"},
				{Name: "b.md", Text: namingReturn(sha, "b", "APPROVE")},
			}); err != nil {
				t.Fatalf("a line that only mentions the word was refused: %v", err)
			}
		})
	}
	for name, brk := range map[string]string{"a carriage return": "\r", "a form feed": "\f", "a line separator": " ", "a next line": "\u0085"} {
		t.Run(name, func(t *testing.T) {
			_, err := RecordReview(repo.Root(), sha, []ReviewReturn{
				{Name: "a.md", Text: namingReturn(sha, "a", "SHIP") + "A note." + brk + "### Final verdict" + brk + "BLOCK\n"},
				{Name: "b.md", Text: namingReturn(sha, "b", "APPROVE")},
			})
			if err == nil || !strings.Contains(err.Error(), "a.md") {
				t.Fatalf("err = %v; want a.md refused for a second Verdict line", err)
			}
		})
	}
	t.Run("windows line ends", func(t *testing.T) {
		if _, err := RecordReview(repo.Root(), sha, []ReviewReturn{
			{Name: "a.md", Text: strings.ReplaceAll(namingReturn(sha, "a", "SHIP"), "\n", "\r\n")},
			{Name: "b.md", Text: namingReturn(sha, "b", "APPROVE")},
		}); err != nil {
			t.Fatalf("a return with windows line ends was refused: %v", err)
		}
	})
}

// A repository whose object names are SHA-256 records and reads the same way: its
// reviewers name the head by its longer name.
func TestAVerdictNamesAHeadOfAnyObjectFormat(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=A Tester", "-c", "user.email=tester@example.com", "-c", "commit.gpgsign=false"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Skipf("git cannot make a SHA-256 repository here: %v (%s)", err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "--object-format=sha256", "--initial-branch=main")
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".abcd/.work.local/\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(TierRelDir)), 0o700); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-q", "-m", "the change")
	sha := git("rev-parse", "HEAD")
	if len(sha) != 64 {
		t.Fatalf("want a SHA-256 name, got %q", sha)
	}
	if _, err := RecordReview(dir, sha, []ReviewReturn{
		{Name: "a.md", Text: namingReturn(sha, "a", "SHIP")},
		{Name: "b.md", Text: namingReturn(sha, "b", "APPROVE")},
	}); err != nil {
		t.Fatalf("the pair naming a SHA-256 head was refused: %v", err)
	}
	if _, err := RecordReview(dir, sha, []ReviewReturn{
		{Name: "a.md", Text: "Reviewed head: " + sha[:9] + "\n\n### Verdict\n\nSHIP " + sha[:40] + "\n"},
		{Name: "b.md", Text: namingReturn(sha, "b", "APPROVE")},
	}); err == nil {
		t.Fatal("a name of SHA-1's length is the whole of no SHA-256 head")
	}
}

// A finding that mentions a verdict, in a reviewer's own format, opens none: emphasis
// makes a line open a verdict only where it wraps the word itself.
func TestAFindingThatMentionsAVerdictOpensNone(t *testing.T) {
	_, sha := reviewRepo(t)
	for _, line := range []string{
		"  - *Failure scenario:* a verdict naming no head is refused.",
		"- **review.go:83** — the verdict reader drops a line.",
		"1. **review.go:83** — the verdict reader drops a line.",
		"_Failure scenario:_ a verdict naming no head is refused.",
		"**The verdict reader drops a line.**",
		"*The verdict reader drops a line.*",
	} {
		text := "Reviewed head: " + sha[:9] + "\n\n### Analysis\n\n" + line + "\n\n### Verdict\n\nSHIP " + sha + "\n"
		if word, err := ReadVerdict(text, sha); err != nil || word != "SHIP" {
			t.Errorf("a return with the finding %q: %q, %v; want SHIP", line, word, err)
		}
	}
}

// A verdict directly under the head's line, with no heading, is refused naming the
// heading it is read under, and the same line under that heading is recorded. The
// reviewers' pages name the heading too.
func TestAVerdictIsReadUnderItsHeadingAndTheRefusalNamesIt(t *testing.T) {
	repo, sha := reviewRepo(t)
	first := "Reviewed head: " + sha[:9] + "\n\n"
	bare := []ReviewReturn{
		{Name: "a.md", Text: first + "SHIP " + sha + "\n"},
		{Name: "b.md", Text: first + "APPROVE " + sha + "\n"},
	}
	if _, err := RecordReview(repo.Root(), sha, bare); err == nil || !strings.Contains(err.Error(), "### Verdict") {
		t.Errorf("a verdict under the head's line: %v; want a refusal naming the heading", err)
	}
	headed := []ReviewReturn{
		{Name: "a.md", Text: first + "### Verdict\n\nSHIP " + sha + "\n"},
		{Name: "b.md", Text: first + "### Verdict\n\nAPPROVE " + sha + "\n"},
	}
	if _, err := RecordReview(repo.Root(), sha, headed); err != nil {
		t.Errorf("the same verdict under the heading: %v", err)
	}
	for _, page := range []string{"ruthless-reviewer.md", "security-reviewer.md"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "agents", page))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "the first line under `### Verdict` opens with the verdict") {
			t.Errorf("%s does not name the heading a verdict is read under", page)
		}
	}
}

// A delta's return holds no Reviewed head line, and its verdict names the head: the
// pages say both, as the reader holds them.
func TestTheDeltaWordingSaysWhatTheReaderHolds(t *testing.T) {
	for _, path := range []string{
		"commands/land.md", "agents/ruthless-reviewer.md", "agents/security-reviewer.md",
	} {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", path))
		if err != nil {
			t.Fatal(err)
		}
		flat := strings.Join(strings.Fields(string(data)), " ")
		if strings.Contains(flat, "no line states a head") || strings.Contains(flat, "no line of it states a head") {
			t.Errorf("%s says no line of a delta's return states a head, and its verdict names one", path)
		}
		if !strings.Contains(flat, "no line reads `Reviewed head:`") {
			t.Errorf("%s does not say a delta's return holds no Reviewed head line", path)
		}
	}
}
