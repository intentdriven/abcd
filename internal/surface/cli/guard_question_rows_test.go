package cli

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/core/question"
)

// The rows limit is the one layout rule the hook does not refuse on
// (iss-2610070637562567). A question whose only finding is its height is
// shown, and the agent is told afterwards, through the host's
// additionalContext, which tab ran over, by how much, and to draft the next
// question through the question-drafter agent. Refusing it made the agent
// redraft a question the person was ready to answer; a question cut at the
// bottom of a short window costs less than that.

// tallQuestion is wellBuilt with more material than fits: every rule holds
// but the rows limit.
func tallQuestion() hostQuestion {
	q := wellBuilt()
	more := []string{
		"The label is the first thing on the screen, above the question text.",
		"It is short, so it never wraps, whatever the window's width.",
		"It names the role first and the number second.",
		"A total after a slash appears only when the interview's length is known.",
		"The number counts the questions put to this person in this interview.",
		"It starts again at one when the next interview opens.",
	}
	q.Question = strings.Join(more, "\n") + "\n\n" + q.Question
	return q
}

// mustBeRowsOnly fails the test unless q breaks the rows limit and nothing else,
// so a test built on it proves the rows path and not another rule's.
func mustBeRowsOnly(t *testing.T, q hostQuestion, who question.Person) []question.Finding {
	t.Helper()
	tab := question.Tab{Header: q.Header, Text: q.Question}
	for _, o := range q.Options {
		tab.Options = append(tab.Options, question.Choice{Label: o.Label, Description: o.Description})
	}
	fs := question.CheckLimits(question.Fields{Tabs: []question.Tab{tab}}, question.Default,
		question.Addressee{Person: who, Verbs: verbsOf(NewRootCommand())})
	if len(fs) != 1 || fs[0].Rule != question.RuleRows {
		t.Fatalf("precondition: the fixture must break the rows limit alone; findings = %v", fs)
	}
	return fs
}

// TestRowsOnlyOverflowIsAdmittedWithANote: a question whose only finding is
// the rows limit runs, is marked open as any admitted question is, and the
// agent is told through additionalContext — never a permission decision,
// which would bypass the host's own permission flow. The note names the tab,
// its rows and the limit, says the question was shown, points at the drafter,
// and never says to ask again: the person may already have answered it.
func TestRowsOnlyOverflowIsAdmittedWithANote(t *testing.T) {
	q := tallQuestion()
	fs := mustBeRowsOnly(t, q, question.ProductThinker)
	limit := fmt.Sprintf("%d rows at %d columns", question.Default.Rows, question.Default.Columns)

	check := func(t *testing.T, note string) {
		t.Helper()
		for _, want := range []string{"tab 1", fs[0].Value, limit, "was shown", "abcd:question-drafter"} {
			if !strings.Contains(note, want) {
				t.Errorf("the note must name %q; note = %q", want, note)
			}
		}
		if strings.Contains(strings.ToLower(note), "ask again") {
			t.Errorf("the note must not tell the agent to ask again; note = %q", note)
		}
	}

	t.Run("managed, mode names the product thinker", func(t *testing.T) {
		root := managedCheckout(t)
		setMode(t, root, mode.ProductThinker)
		stdout, stderr, code := runGuard(askPayload(t, root, q), "guard", "hook")
		check(t, mustAdmitWithNote(t, stdout, stderr, code))
		if !markedOpen(t, root) {
			t.Error("the admitted question was not marked open, so the answer will not reset the mode")
		}
	})
	t.Run("unmanaged, chip alone", func(t *testing.T) {
		repo := unmanagedRepo(t)
		stdout, stderr, code := runGuard(askPayload(t, repo, q), "guard", "hook")
		check(t, mustAdmitWithNote(t, stdout, stderr, code))
	})
}

// TestRowsNoteIsBounded: a call of four tall tabs names each, and the note
// stays one bounded block whatever the payload holds.
func TestRowsNoteIsBounded(t *testing.T) {
	repo := unmanagedRepo(t)
	q := tallQuestion()
	mustBeRowsOnly(t, q, question.ProductThinker)
	qs := []hostQuestion{q, q, q, q}
	for i := range qs {
		qs[i].Header = fmt.Sprintf("Product Q%d", i+1)
	}
	stdout, stderr, code := runGuard(askPayload(t, repo, qs...), "guard", "hook")
	note := mustAdmitWithNote(t, stdout, stderr, code)
	for i := 1; i <= len(qs); i++ {
		if !strings.Contains(note, fmt.Sprintf("tab %d", i)) {
			t.Errorf("the note must name tab %d; note = %q", i, note)
		}
	}
	if len(note) > 4096 {
		t.Errorf("the note is %d bytes; it must stay under 4 KiB", len(note))
	}
}

// TestRowsWithAnotherFindingIsDenied: the rows limit is admitted only alone. A
// tall question that also breaks another rule is refused, and the refusal
// counts and lists only the refusing parts as the parts to fix; the rows
// finding follows them, on a line saying the rows limit does not refuse on its
// own and the question would have been shown (iss-2610100626327722).
func TestRowsWithAnotherFindingIsDenied(t *testing.T) {
	root := managedCheckout(t)
	setMode(t, root, mode.ProductThinker)
	q := tallQuestion()
	q.Options[0].Label = "Keep it (Recommended)"
	stdout, stderr, code := runGuard(askPayload(t, root, q), "guard", "hook")
	reason := mustDeny(t, stdout, stderr, code)
	head, parts, _ := refusal(t, reason)
	if got := rulesNamed(parts); !slices.Equal(got, []string{"never-recommended"}) {
		t.Errorf("want only the never-recommended finding among the parts to fix; got %v:\n%s", got, reason)
	}
	if !strings.Contains(head, ": 1 part(s)") {
		t.Errorf("the head line must count only the refusing part; head = %q", head)
	}
	line, rows := rowsAfterRefusal(reason)
	if line == "" {
		t.Fatalf("the refusal must say the rows limit does not refuse on its own; reason:\n%s", reason)
	}
	if !strings.Contains(line, "would have been shown") {
		t.Errorf("the rows line must say the question would have been shown; line = %q", line)
	}
	if !slices.Equal(rulesNamed(rows), []string{string(question.RuleRows)}) {
		t.Errorf("want the rows finding named after the rows line; got %v:\n%s", rulesNamed(rows), reason)
	}
	if markedOpen(t, root) {
		t.Error("a refused question was marked open")
	}
}

// TestRowsOnlyWhileManagedIsDenied: the mode gate still refuses. A tall
// question asked while the mode reads managed is refused with the mode's line,
// and the rows finding follows it as one that does not refuse on its own.
func TestRowsOnlyWhileManagedIsDenied(t *testing.T) {
	root := managedCheckout(t)
	q := tallQuestion()
	mustBeRowsOnly(t, q, question.ProductThinker)
	stdout, stderr, code := runGuard(askPayload(t, root, q), "guard", "hook")
	reason := mustDeny(t, stdout, stderr, code)
	if !strings.Contains(reason, questionRefusal) {
		t.Errorf("the mode's refusal must be named; reason = %q", reason)
	}
	if _, rows := rowsAfterRefusal(reason); !slices.Equal(rulesNamed(rows), []string{string(question.RuleRows)}) {
		t.Errorf("want the rows finding named after the rows line; got %v:\n%s", rulesNamed(rows), reason)
	}
	if strings.Contains(reason, "part(s) of this question break") {
		t.Errorf("no part refuses but the mode, so no head line may count one; reason:\n%s", reason)
	}
	if markedOpen(t, root) {
		t.Error("a refused question was marked open")
	}
}
