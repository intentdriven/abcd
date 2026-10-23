package checklist

import (
	"fmt"
	"strings"
)

// render.go — the injection text (spc-2610230805429429 step 3, "Re-injection"):
// the one render both `abcd checklist render` and the session-start hook print,
// so the two say the same thing for the same budget.

// renderIntro is the render's first line over a checklist it could show.
const renderIntro = "This working copy's checklist (`abcd checklist` lists every field):"

// Render is the checklist as the text a session's context is given, within
// budget bytes: the reader's reason first when it did not read checklist.json
// clean, then the open items (working, waiting), then the done and dropped
// ones, each group newest first, one line an item.
//
// "Newest" is file order read backwards: an item carries no time of its own,
// and the verbs append each new item to the end of the list.
//
// Over budget, the render keeps the reason and the newest open items, drops
// the rest from the end (the closed items before any open one), and ends with
// a line saying how many it left out and that `abcd checklist` shows them all.
// The budget is in bytes, which never undercounts a host that measures in
// characters. When not even the reason fits, the render is empty; a FromNothing
// result renders as the reason line alone.
//
// Every field the reader passed is one line with no control character
// (Validate), so each prints as it stands.
func Render(res Result, budget int) string {
	var head []string
	if res.Source != FromFile {
		head = append(head, res.Reason)
	}
	if res.Source == FromNothing {
		return fit(budget, head)
	}
	head = append(head, renderIntro)
	if len(res.Checklist.Items) == 0 {
		return fit(budget, append(head, "No items."))
	}

	var open, closed []string
	items := res.Checklist.Items
	for i := len(items) - 1; i >= 0; i-- {
		if line := itemLine(items[i]); items[i].State == Working || items[i].State == Waiting {
			open = append(open, line)
		} else {
			closed = append(closed, line)
		}
	}
	const openHeader, closedHeader = "Open:", "Done or dropped:"
	ordered := append(append([]string{}, open...), closed...)

	// size is the bytes of the render that shows the first k of ordered.
	headBytes := linesBytes(head)
	prefix := make([]int, len(ordered)+1)
	for i, l := range ordered {
		prefix[i+1] = prefix[i] + len(l) + 1
	}
	size := func(k int) int {
		n := headBytes + prefix[k]
		if k > 0 && len(open) > 0 {
			n += len(openHeader) + 1
		}
		if k > len(open) {
			n += len(closedHeader) + 1
		}
		if left := len(ordered) - k; left > 0 {
			n += len(omission(left)) + 1
		}
		return n
	}
	for k := len(ordered); k >= 0; k-- {
		if size(k) > budget {
			continue
		}
		lines := append([]string{}, head...)
		shownOpen := min(k, len(open))
		if shownOpen > 0 {
			lines = append(append(lines, openHeader), open[:shownOpen]...)
		}
		if k > len(open) {
			lines = append(append(lines, closedHeader), closed[:k-len(open)]...)
		}
		if left := len(ordered) - k; left > 0 {
			lines = append(lines, omission(left))
		}
		return strings.Join(lines, "\n") + "\n"
	}
	// Not even the head and the count fit: the reason alone, or nothing.
	return fit(budget, head[:len(head)-1])
}

// itemLine is one item's line: its state, id and title, and what it waits on.
func itemLine(it Item) string {
	line := fmt.Sprintf("- [%s] %s: %s", it.State, it.ID, it.Title)
	if it.WaitsOn != "" {
		line += " (waits on " + it.WaitsOn + ")"
	}
	return line
}

// omission is the line that ends a cut render.
func omission(left int) string {
	noun := "items"
	if left == 1 {
		noun = "item"
	}
	return fmt.Sprintf("%d more %s left out; run `abcd checklist` to see them all.", left, noun)
}

// fit is lines as text when it fits budget, and empty otherwise.
func fit(budget int, lines []string) string {
	if len(lines) == 0 || linesBytes(lines) > budget {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func linesBytes(lines []string) int {
	n := 0
	for _, l := range lines {
		n += len(l) + 1
	}
	return n
}
