// Package recall is the one recall for memory notes and pages
// (itd-2610091918433290, itd-39; spc-2610111627213138). The prompt router
// calls it once per prompt, after rules.Inject, with the items the memory
// layers yield, and it returns one block headed "## MEMORY" for the hook to
// append after the rules block.
//
// Matching is the loader's own (rules.TermMatcher), so a note's recall word
// and a rule domain's recall word hit on exactly the same prompts. The
// per-session ledger is SessionState.Recalled, keyed on each item's content,
// cleared by the loader's resets. The budget is what the rules leave of the
// loader's one per-turn budget (rules.InjectionBudgetBytes). Until the
// relevance ranking lands (step 4), the candidates are taken in id order.
//
// Step 2 recalls notes; pages join the same path in step 3. Nothing here
// writes to stdout or knows a transport.
package recall

import (
	"sort"

	"github.com/intentdriven/abcd/internal/core/memnote"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The layers an item comes from, as the diagnostic names them.
const (
	LayerProject = "project"
	LayerMachine = "machine"
)

// Item is one thing the recall can bring back: a note now, a page from step
// 3. Key is its ledger key and its place in the id order — a note's id
// (mem-…), and a page's "page:<filename>" after every note.
type Item struct {
	Key    string
	Layer  string
	Title  string
	Body   string
	Recall []string
}

// Refusal names a layer the recall could not read, and why, in words safe to
// print: the file in tilde or repository-relative form, the reason
// home-redacted and terminal-safe.
type Refusal struct {
	File   string `json:"file"`
	Reason string `json:"reason"`
}

// noteItems turns both note layers into items, in id order. A personal note
// is an item only in the repository it is tied to — root, the repository's
// root commit, "" where there is none — or everywhere when its root is null.
// Every note in the project's file is an item. A note in both layers (an id
// is minted once, so only a hand copy does that) is the project's: the
// machine's copy is suppressed.
func noteItems(machine, project []memnote.Note, root string) []Item {
	seen := map[string]bool{}
	var out []Item
	for _, n := range project {
		seen[n.ID] = true
		out = append(out, noteItem(n, LayerProject))
	}
	for _, n := range machine {
		if seen[n.ID] {
			continue
		}
		if n.Root != nil && (root == "" || *n.Root != root) {
			continue
		}
		seen[n.ID] = true
		out = append(out, noteItem(n, LayerMachine))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func noteItem(n memnote.Note, layer string) Item {
	return Item{Key: n.ID, Layer: layer, Title: n.Title, Body: n.Body, Recall: append([]string(nil), n.Recall...)}
}

// rootCommit is the repository's root commit; a seam for the tests.
var rootCommit = gitutil.RootCommit

// ReadNotes reads both note layers through memnote's layer readers — the
// machine's under home, the project's under repoRoot, the root the rules
// loader resolved — and returns their items. A layer that cannot be read is
// a Refusal and yields no items, never an error: the hook still injects the
// rules, and recalls what the other layer holds. The repository's root
// commit is asked of git only when a personal note is tied to one.
func ReadNotes(home, repoRoot string) ([]Item, []Refusal) {
	var refused []Refusal
	var machine, project []memnote.Note
	if f, _, err := memnote.ReadMachine(home); err != nil {
		refused = append(refused, Refusal{File: memnote.MachineDisplayPath, Reason: reason(err)})
	} else {
		machine = f.Notes
	}
	if f, _, err := memnote.ReadRepo(repoRoot); err != nil {
		refused = append(refused, Refusal{File: memnote.RepoRelPath, Reason: reason(err)})
	} else {
		project = f.Notes
	}
	root := ""
	for _, n := range machine {
		if n.Root != nil {
			root = rootCommit(repoRoot)
			break
		}
	}
	return noteItems(machine, project, root), refused
}

// reason is a refusal's text, home-redacted and terminal-safe.
func reason(err error) string {
	return termsafe.Sanitize(fsutil.RedactHome(err.Error()))
}
