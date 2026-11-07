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
	// Budget is what the rules and the blocks read before them left of the
	// loader's per-turn budget, rules.InjectionBudgetBytes.
	Budget int
	// Disabled is the loader's kill switch: recall is the loader's, so the
	// switch silences it too.
	Disabled bool
	// Root answers the repository's root commit, for the items tied to one;
	// asked only once a tied item matched (matching).
	Root RootFunc
	// Bracket is the context bracket the session is in (rules.BracketFor
	// its fill), whose row says how many candidates come back in full and
	// how many as one line each. The zero value is rules.BracketUnknown:
	// one line each.
	Bracket rules.Bracket
}

// Result is one prompt's recall. Text is empty when nothing new comes back.
// Recalled names each item Text carries, in order; Omitted each new
// candidate that did not fit this turn's budget, not recorded and so retried
// on the next turn; TooLarge each new candidate whose one line alone is larger
// than the whole per-turn budget (rules.InjectionBudgetBytes), which no turn
// can carry, so it is set aside without stopping the take and never
// recorded. Ledger is the updated recalled ledger, a copy. RootErr is the
// root-commit lookup's error (git did not answer in time) when a tied item
// matched and was left out for it — unrecorded, so retried on the next turn.
type Result struct {
	Text     string
	Recalled []string
	Omitted  []string
	TooLarge []string
	Ledger   map[string]string
	RootErr  error

	// layers is each candidate's layer, by key, for Labels.
	layers map[string]string
	// forms is the form each recalled item was sent in, by key, for Form.
	forms map[string]string
}

// Form is the form the recall sent an item in this turn: FormFull or
// FormLine, and "" for an item it did not send.
func (r Result) Form(key string) string { return r.forms[key] }

// Lines names, in the order Recalled does, each item sent as one line.
func (r Result) Lines() []string {
	var out []string
	for _, k := range r.Recalled {
		if r.forms[k] == FormLine {
			out = append(out, k)
		}
	}
	return out
}

// Labels names keys as the hook's diagnostic does, each with its layer —
// "mem-… (project)", "mem-… (machine)" — so the out-of-band log says whose
// words went into the context, as the rules line's "(repo override)" does.
// A key this recall did not take or omit is named bare.
func (r Result) Labels(keys []string) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if layer := r.layers[k]; layer != "" {
			out = append(out, k+" ("+layer+")")
		} else {
			out = append(out, k)
		}
	}
	return out
}

// Recall matches the items against the prompt, judges each tied note's root
// (matching), ranks the candidates (Rank), drops every one already sent this
// session with unchanged content, and takes the rest in ranked order in the
// forms the context bracket's row gives them (take, rules.BracketRow): in
// full, then as one line each, within the row's bounds and the budget. The
// first that does not fit a form ends that form's take, so an item never
// comes back ahead of, or larger than, one ranked above it that a later turn
// can still carry. An item larger than the whole per-turn budget is sent as
// one line, so it cannot hold back every item below it on every turn. The ledger records an
// item's content whatever its form, so a change of bracket never sends it
// again. Its only I/O is req.Root, asked when a tied item matched, and it
// never reflects prompt bytes into the block.
func Recall(req Request) Result {
	ledger := map[string]string{}
	for k, v := range req.Ledger {
		ledger[k] = v
	}
	res := Result{Ledger: ledger}
	if req.Disabled {
		return res
	}
	ranked, rootErr := Rank(req.Items, req.Prompt, req.Root)
	res.RootErr = rootErr
	var fresh []Item
	for _, c := range ranked {
		if ledger[c.Item.Key] == signature(c.Item) {
			continue
		}
		fresh = append(fresh, c.Item)
	}
	if len(fresh) == 0 {
		return res
	}
	res.layers = make(map[string]string, len(fresh))
	for _, it := range fresh {
		res.layers[it.Key] = it.Layer
	}
	res.forms = map[string]string{}
	t := newTake(req.Bracket.Row(), req.Budget)
	var b strings.Builder
	for _, it := range fresh {
		block, form, verdict := t.next(it)
		switch verdict {
		case takeTooLarge:
			res.TooLarge = append(res.TooLarge, it.Key)
		case takeOmitted:
			res.Omitted = append(res.Omitted, it.Key)
		default:
			b.WriteString(block)
			ledger[it.Key] = signature(it)
			res.forms[it.Key] = form
			res.Recalled = append(res.Recalled, it.Key)
		}
	}
	if len(res.Recalled) > 0 {
		res.Text = headerLine + b.String()
	}
	return res
}

// What the take decides for one candidate.
type takeVerdict int

const (
	takeSent takeVerdict = iota
	takeOmitted
	takeTooLarge
)

// take is one turn's choice, in ranked order, of the form each new candidate
// comes back in, under the bracket's row (rules.BracketRow): in full while
// the row's full bound and the per-turn budget hold, then as one line each
// while the row's line bound and the budget hold, then not this turn. Each
// phase ends at the first candidate it cannot carry, so no candidate comes
// back in a larger form than one ranked above it, and none comes back at all
// ahead of one ranked above it that a later turn can still carry.
//
// Two exceptions keep a candidate from holding back the rest. A candidate
// whose full block alone is larger than the whole per-turn budget, which no
// turn can carry, is passed to the lines without ending the full phase. And where the row has lines, the first candidate the take
// reaches, the top-ranked, comes back as one line at least whatever the
// row's line bound, so the recall never sends nothing (the facilitator's
// ruling of 2026-10-11) — though never past the per-turn budget, the
// loader's flood guard, which bounds every form.
type take struct {
	budget   int  // the per-turn budget left to the recall, the header included
	size     int  // the block so far, the header included
	fullLeft int  // the row's full bound left; rules.MemoryAll for none
	lineLeft int  // the row's line bound left; rules.MemoryAll for none
	fullOpen bool // the full phase still takes
	lineOpen bool // the line phase still takes
	first    bool // no candidate reached yet
}

func newTake(row rules.BracketRow, budget int) *take {
	return &take{
		budget:   budget,
		size:     len(headerLine),
		fullLeft: row.MemoryFullBytes,
		lineLeft: row.MemoryLineBytes,
		fullOpen: row.MemoryFullBytes != 0,
		lineOpen: row.MemoryLineBytes != 0,
		first:    true,
	}
}

// within reports whether n bytes fit a bound left, rules.MemoryAll being
// no bound.
func within(n, left int) bool { return left == rules.MemoryAll || n <= left }

// spend takes n bytes off a bound left, never below 0; rules.MemoryAll
// stays itself.
func spend(left, n int) int {
	if left == rules.MemoryAll {
		return left
	}
	return max(0, left-n)
}

// tooLarge reports whether a block or line alone is larger than the whole per-turn
// budget, which no turn can carry.
func tooLarge(block string) bool { return len(headerLine)+len(block) > rules.InjectionBudgetBytes }

// next decides one candidate, the next in ranked order.
func (t *take) next(it Item) (block, form string, v takeVerdict) {
	top := t.first
	t.first = false
	full := renderItem(it)
	if t.fullOpen && !tooLarge(full) {
		if within(len(full), t.fullLeft) && t.size+len(full) <= t.budget {
			t.size += len(full)
			t.fullLeft = spend(t.fullLeft, len(full))
			return full, FormFull, takeSent
		}
		t.fullOpen = false
	}
	if !t.lineOpen {
		return "", "", takeOmitted
	}
	line := renderLine(it)
	switch {
	case tooLarge(line):
		return "", "", takeTooLarge
	case t.size+len(line) > t.budget, !within(len(line), t.lineLeft) && !top:
		t.lineOpen = false
		return "", "", takeOmitted
	}
	t.size += len(line)
	t.lineLeft = spend(t.lineLeft, len(line))
	return line, FormLine, takeSent
}

// signature is the ledger's key for an item's content, never for the form it
// was rendered in, so a later change of form never sends an item again while
// a change of its words does. The layer is part of the content: it is whose
// words these are, and the rendered label says so, so a note promoted from
// the machine's file into the project's is sent again under its new label.
// FNV-1a, as the rules ledger: dedup, not security.
func signature(it Item) string {
	h := fnv.New64a()
	for _, part := range append([]string{it.Key, it.Layer, it.Title, it.Body}, it.Recall...) {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return fmt.Sprintf("%016x", h.Sum64())
}
