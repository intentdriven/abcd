package recall

import (
	"fmt"
	"hash/fnv"
	"strings"
)

// Request is one prompt's recall.
type Request struct {
	// Items are what the memory layers yield (noteItems, ReadNotes).
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
	// Root answers the repository's root commit, for the items tied to one;
	// asked only once a tied item matched (Matching).
	Root RootFunc
}

// Result is one prompt's recall. Text is empty when nothing new comes back.
// Recalled names each item Text carries, in order; Omitted each new
// candidate that did not fit the budget, not recorded and so retried on the
// next turn. Ledger is the updated recalled ledger, a copy. RootErr is the
// root-commit lookup's error (git did not answer in time) when a tied item
// matched and was left out for it — unrecorded, so retried on the next turn.
type Result struct {
	Text     string
	Recalled []string
	Omitted  []string
	Ledger   map[string]string
	RootErr  error

	// layers is each candidate's layer, by key, for Labels.
	layers map[string]string
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

// Recall matches the items against the prompt, drops every candidate already
// sent this session with unchanged content, and takes the rest in id order
// while they fit the budget: the first that does not fit stops the take, so
// an item never comes back ahead of one before it in the order. Its only I/O
// is req.Root, asked when a tied item matched, and it never reflects prompt
// bytes into the block.
func Recall(req Request) Result {
	ledger := map[string]string{}
	for k, v := range req.Ledger {
		ledger[k] = v
	}
	res := Result{Ledger: ledger}
	if req.Disabled {
		return res
	}
	cands, rootErr := Matching(req.Items, req.Prompt, req.Root)
	res.RootErr = rootErr
	var fresh []Item
	for _, c := range cands {
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
	var b strings.Builder
	size := len(headerLine)
	for i, it := range fresh {
		block := renderItem(it)
		if size+len(block) > req.Budget {
			for _, rest := range fresh[i:] {
				res.Omitted = append(res.Omitted, rest.Key)
			}
			break
		}
		b.WriteString(block)
		size += len(block)
		ledger[it.Key] = signature(it)
		res.Recalled = append(res.Recalled, it.Key)
	}
	if len(res.Recalled) > 0 {
		res.Text = headerLine + b.String()
	}
	return res
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
