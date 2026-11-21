package matrix

import (
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"
)

// A reviewer is handed a copy's normalised diff (adr-2610161733400551): the
// diff is computed here from the two files' bytes alone, so nothing of git's
// (an index line, a commit, a configured driver) enters it.

func TestTheNormalisedDiffOfAChangedFile(t *testing.T) {
	old := "package a\n\nfunc A() int {\n\treturn 1\n}\n"
	new := "package a\n\nfunc A() int {\n\treturn 2\n}\n"
	got := unifiedDiff("a.go", []byte(old), []byte(new))
	want := "--- a/a.go\n+++ b/a.go\n@@ -1,5 +1,5 @@\n package a\n \n func A() int {\n-\treturn 1\n+\treturn 2\n }\n"
	if got != want {
		t.Fatalf("diff:\n%s\nwant:\n%s", got, want)
	}
}

func TestTheNormalisedDiffKeepsThreeLinesOfContextAndSplitsDistantHunks(t *testing.T) {
	var a, b []string
	for i := 1; i <= 20; i++ {
		a = append(a, "line "+strconv.Itoa(i))
		b = append(b, "line "+strconv.Itoa(i))
	}
	b[1] = "changed 2"
	b[18] = "changed 19"
	got := unifiedDiff("f.txt", []byte(strings.Join(a, "\n")+"\n"), []byte(strings.Join(b, "\n")+"\n"))
	if n := strings.Count(got, "\n@@ "); n != 2 {
		t.Fatalf("two changes sixteen lines apart are two hunks, got %d:\n%s", n, got)
	}
	if !strings.Contains(got, "@@ -1,5 +1,5 @@\n line 1\n-line 2\n+changed 2\n line 3\n line 4\n line 5\n") {
		t.Fatalf("the first hunk holds three lines of context after the change:\n%s", got)
	}
	if !strings.Contains(got, "@@ -16,5 +16,5 @@\n line 16\n line 17\n line 18\n-line 19\n+changed 19\n line 20\n") {
		t.Fatalf("the second hunk holds three lines of context before the change:\n%s", got)
	}
}

func TestTheNormalisedDiffOfAnAddedAndADeletedFile(t *testing.T) {
	if got, want := unifiedDiff("n.go", nil, []byte("x\ny\n")), "--- /dev/null\n+++ b/n.go\n@@ -0,0 +1,2 @@\n+x\n+y\n"; got != want {
		t.Fatalf("added:\n%s\nwant:\n%s", got, want)
	}
	if got, want := unifiedDiff("d.go", []byte("x\n"), nil), "--- a/d.go\n+++ /dev/null\n@@ -1 +0,0 @@\n-x\n"; got != want {
		t.Fatalf("deleted:\n%s\nwant:\n%s", got, want)
	}
}

// Line endings are normalised, so a copy whose editor wrote CRLF differs from
// its neighbours only where its lines do; a missing final newline is marked.
func TestTheNormalisedDiffIgnoresLineEndingsAndMarksAMissingFinalNewline(t *testing.T) {
	if got := unifiedDiff("c.go", []byte("a\nb\n"), []byte("a\r\nb\r\n")); got != "" {
		t.Fatalf("a change of line endings alone is no change:\n%s", got)
	}
	got := unifiedDiff("e.go", []byte("a\nb\n"), []byte("a\nb"))
	if !strings.Contains(got, "-b\n+b\n\\ No newline at end of file\n") {
		t.Fatalf("a dropped final newline is shown and marked:\n%s", got)
	}
}

// A rewrite too large for the edit search is shown whole, removed then added,
// rather than searched without bound.
func TestARewritePastTheEditBoundIsShownWhole(t *testing.T) {
	var a, b strings.Builder
	for i := 0; i < maxDiffEdits+10; i++ {
		a.WriteString("old " + strconv.Itoa(i) + "\n")
		b.WriteString("new " + strconv.Itoa(i) + "\n")
	}
	got := unifiedDiff("big.txt", []byte(a.String()), []byte(b.String()))
	n := maxDiffEdits + 10
	if !strings.HasPrefix(got, "--- a/big.txt\n+++ b/big.txt\n@@ -1,"+strconv.Itoa(n)+" +1,"+strconv.Itoa(n)+" @@\n-old 0\n") {
		t.Fatalf("the rewrite is one hunk, every old line removed then every new one added:\n%.200s", got)
	}
	if strings.Count(got, "\n-old ") != n || strings.Count(got, "\n+new ") != n {
		t.Fatal("every line of both sides is shown")
	}
}

// The edit script is a shortest one: over many small random pairs it rebuilds
// both sides and keeps as many lines as their longest common subsequence.
func TestTheEditScriptIsAShortestOne(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for round := 0; round < 2000; round++ {
		a := make([]string, rng.IntN(9))
		b := make([]string, rng.IntN(9))
		for i := range a {
			a[i] = string(rune('a' + rng.IntN(3)))
		}
		for i := range b {
			b[i] = string(rune('a' + rng.IntN(3)))
		}
		ops := editScript(a, b)
		var gotA, gotB []string
		kept := 0
		for _, op := range ops {
			switch op.kind {
			case opKeep:
				if a[op.a] != b[op.b] {
					t.Fatalf("%v %v: kept unequal lines", a, b)
				}
				gotA, gotB = append(gotA, a[op.a]), append(gotB, b[op.b])
				kept++
			case opDel:
				gotA = append(gotA, a[op.a])
			case opAdd:
				gotB = append(gotB, b[op.b])
			}
		}
		if strings.Join(gotA, "") != strings.Join(a, "") || strings.Join(gotB, "") != strings.Join(b, "") {
			t.Fatalf("%v %v: the script does not rebuild both sides: %v", a, b, ops)
		}
		if want := lcs(a, b); kept != want {
			t.Fatalf("%v %v: kept %d lines, the longest common subsequence is %d", a, b, kept, want)
		}
	}
}

func lcs(a, b []string) int {
	dp := make([][]int, len(a)+1)
	for i := range dp {
		dp[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else {
				dp[i][j] = max(dp[i+1][j], dp[i][j+1])
			}
		}
	}
	return dp[0][0]
}
