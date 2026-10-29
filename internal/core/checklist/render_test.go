package checklist_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/checklist"
)

// render_test.go — the injection text (spc-2610230805429429 step 3): open
// items first, newest first, then the done and dropped ones; over its budget
// the render cuts and says how many it left out and how to see them.

func fromFile(items ...checklist.Item) checklist.Result {
	c := sample()
	c.Items = items
	return checklist.Result{Checklist: c, Source: checklist.FromFile, Reason: "Read checklist.json, which validated and scanned clean."}
}

func item(id string, s checklist.State) checklist.Item {
	return checklist.Item{ID: id, Title: "Title of " + id, State: s}
}

// order answers the ids in the order the text names them.
func order(text string, ids ...string) []string {
	type at struct {
		id  string
		pos int
	}
	var found []at
	for _, id := range ids {
		if p := strings.Index(text, "] "+id+":"); p >= 0 {
			found = append(found, at{id, p})
		}
	}
	for i := 1; i < len(found); i++ {
		for j := i; j > 0 && found[j].pos < found[j-1].pos; j-- {
			found[j], found[j-1] = found[j-1], found[j]
		}
	}
	out := make([]string, len(found))
	for i, f := range found {
		out[i] = f.id
	}
	return out
}

func TestRenderPutsOpenItemsFirstNewestFirst(t *testing.T) {
	res := fromFile(
		item("first-open", checklist.Working),
		item("first-done", checklist.Done),
		item("second-open", checklist.Waiting),
		item("dropped-one", checklist.Dropped),
		item("third-open", checklist.Working),
	)
	text := checklist.Render(res, 10_000)
	got := order(text, "first-open", "first-done", "second-open", "dropped-one", "third-open")
	want := []string{"third-open", "second-open", "first-open", "dropped-one", "first-done"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("order = %v, want %v:\n%s", got, want, text)
	}
	if strings.Contains(text, "left out") {
		t.Errorf("a list within its budget says it left something out:\n%s", text)
	}
	if strings.Contains(text, res.Reason) {
		t.Errorf("a clean read repeats the reader's reason:\n%s", text)
	}
}

// bigList is n open items, then n closed ones, each with a long title.
func bigList(n int) checklist.Result {
	var items []checklist.Item
	for i := 0; i < n; i++ {
		items = append(items, checklist.Item{ID: fmt.Sprintf("open-%d", i), Title: strings.Repeat("o", 150), State: checklist.Working})
	}
	for i := 0; i < n; i++ {
		items = append(items, checklist.Item{ID: fmt.Sprintf("closed-%d", i), Title: strings.Repeat("c", 150), State: checklist.Done})
	}
	return fromFile(items...)
}

func TestRenderCutsOverBudgetAndSaysSo(t *testing.T) {
	res := bigList(40)
	const budget = 2_000
	text := checklist.Render(res, budget)
	if len(text) > budget {
		t.Fatalf("the render is %d bytes, over its budget of %d", len(text), budget)
	}
	shown := strings.Count(text, "\n- [")
	if strings.HasPrefix(text, "- [") {
		shown++
	}
	left := len(res.Checklist.Items) - shown
	if left <= 0 {
		t.Fatalf("nothing was cut from an over-budget list:\n%s", text)
	}
	want := fmt.Sprintf("%d more items left out; run `abcd checklist` to see them all.", left)
	if !strings.Contains(text, want) {
		t.Errorf("the render does not say %q:\n%s", want, text)
	}
	// The newest open items are the ones kept, and no closed item is shown
	// while an open one is left out.
	if !strings.Contains(text, "] open-39:") || strings.Contains(text, "] closed-") {
		t.Errorf("the cut did not keep the newest open items first:\n%s", text)
	}
}

func TestRenderNeverExceedsItsBudget(t *testing.T) {
	res := bigList(10)
	for budget := 0; budget <= 4_000; budget += 7 {
		if text := checklist.Render(res, budget); len(text) > budget {
			t.Fatalf("budget %d: the render is %d bytes", budget, len(text))
		}
	}
}

func TestRenderFallbackLeadsWithTheReason(t *testing.T) {
	res := fromFile(item("kept", checklist.Working))
	res.Source = checklist.FromLastGood
	res.Reason = "Showing the last good copy, because checklist.json fails the schema: items[0].state must be one of working."
	text := checklist.Render(res, 10_000)
	if first, _, _ := strings.Cut(text, "\n"); first != res.Reason {
		t.Errorf("the first line is %q, want the reason", first)
	}
	if !strings.Contains(text, "] kept:") {
		t.Errorf("the fallback does not show the last good items:\n%s", text)
	}
}

func TestRenderOfNothingIsTheLineAlone(t *testing.T) {
	res := checklist.Result{Source: checklist.FromNothing, Reason: "No checklist to show: checklist.json is absent, and checklist.last-good.json fails the schema."}
	if text := checklist.Render(res, 10_000); text != res.Reason+"\n" {
		t.Errorf("render of nothing = %q, want the reason line alone", text)
	}
}

// TestRenderKeepsAFallbackReasonOverAnEmptyList: the reason is the line the
// render drops last, so a tight budget drops "No items." and the intro first.
func TestRenderKeepsAFallbackReasonOverAnEmptyList(t *testing.T) {
	res := fromFile()
	res.Source = checklist.FromLastGood
	res.Reason = "Showing the last good copy, because checklist.json is not a regular file; the copy was written by a checked write and validated and scanned clean again."
	if text := checklist.Render(res, len(res.Reason)+1); text != res.Reason+"\n" {
		t.Errorf("render within the reason's own bytes = %q, want the reason line alone", text)
	}
	full := checklist.Render(res, 10_000)
	if !strings.HasPrefix(full, res.Reason+"\n") || !strings.HasSuffix(full, "No items.\n") {
		t.Errorf("an ample budget = %q, want the reason, the intro and No items.", full)
	}
}
