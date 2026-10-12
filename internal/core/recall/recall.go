package recall

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/rules"
)

// Request is one prompt's recall.
type Request struct {
	// Items are every note and page in reach (ReadItems).
	Items []Item
	// Prompt is the prompt the hook was handed.
	Prompt string
	// Ledger is the session's recalled ledger (rules.SessionState.Recalled):
	// item key to the content signature last sent.
	Ledger map[string]string
	// Budget is what the rules left of the loader's per-turn budget,
	// rules.InjectionBudgetBytes less the rules text's length.
	Budget int
	// Disabled is the loader's kill switch: recall is the loader's, so the
	// switch silences it too.
	Disabled bool
}

// Result is one prompt's recall. Text is empty when nothing new comes back.
// Recalled names each item Text carries, in order; Omitted each new
// candidate that did not fit this turn's budget, not recorded and so retried
// on the next turn; TooLarge each new candidate whose block alone is larger
// than the whole per-turn budget (rules.InjectionBudgetBytes), which no turn
// can carry, so it is set aside without stopping the take and never
// recorded. Ledger is the updated recalled ledger, a copy.
type Result struct {
	Text     string
	Recalled []string
	Omitted  []string
	TooLarge []string
	Ledger   map[string]string
}

// Recall matches the items against the prompt, ranks the candidates (Rank),
// drops every one already sent this session with unchanged content, and
// takes the rest in ranked order while they fit the budget: the first that
// does not fit stops the take, so an item never comes back ahead of one
// ranked above it that a later turn can still carry. An item larger than the
// whole per-turn budget is set aside as TooLarge instead, so it cannot hold
// back every item below it on every turn. It performs no I/O and never
// reflects prompt bytes into the block.
func Recall(req Request) Result {
	ledger := map[string]string{}
	for k, v := range req.Ledger {
		ledger[k] = v
	}
	res := Result{Ledger: ledger}
	if req.Disabled {
		return res
	}
	var fresh []Item
	for _, c := range Rank(req.Items, req.Prompt) {
		if ledger[c.Item.Key] == signature(c.Item) {
			continue
		}
		fresh = append(fresh, c.Item)
	}
	if len(fresh) == 0 {
		return res
	}
	var b strings.Builder
	size := len(headerLine)
	stopped := false
	for _, it := range fresh {
		block := renderItem(it)
		switch {
		case len(headerLine)+len(block) > rules.InjectionBudgetBytes:
			res.TooLarge = append(res.TooLarge, it.Key)
		case stopped || size+len(block) > req.Budget:
			stopped = true
			res.Omitted = append(res.Omitted, it.Key)
		default:
			b.WriteString(block)
			size += len(block)
			ledger[it.Key] = signature(it)
			res.Recalled = append(res.Recalled, it.Key)
		}
	}
	if len(res.Recalled) > 0 {
		res.Text = headerLine + b.String()
	}
	return res
}

// signature is the ledger's key for an item's content, never for the form it
// was rendered in, so a later change of form never sends an item again while
// a change of its words does. FNV-1a, as the rules ledger: dedup, not
// security.
func signature(it Item) string {
	h := fnv.New64a()
	for _, part := range append([]string{it.Key, it.Title, it.Body}, it.Recall...) {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return fmt.Sprintf("%016x", h.Sum64())
}
