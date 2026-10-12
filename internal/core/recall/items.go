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
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The layers an item comes from, as the diagnostic names them.
const (
	LayerProject = "project"
	LayerMachine = "machine"
)

// Item is one thing the recall can bring back: a note now, a page from step
// 3. Key is its ledger key and its place in the id order — a note's id
// (mem-…), and a page's "page:<filename>" after every note. Tie is the root
// commit a personal note is tied to; nil for a note recalled everywhere and
// for every item in the project's file.
type Item struct {
	Key    string
	Layer  string
	Title  string
	Body   string
	Recall []string
	Tie    *string
}

// RootFunc answers the repository's root commit, "" where there is none. The
// recall asks it only once matching has found a tied item, and at most once
// per call, so a prompt that matches no tied item costs no git call. A caller
// that hands the same RootFunc to more than one call memoises it
// (sync.OnceValues). An error is git not answering in time
// (gitutil.RootCommitContext): the tied items are then left out.
type RootFunc func() (string, error)

// Refusal names a layer the recall could not read, and why, in words safe to
// print: the file in tilde or repository-relative form, the reason
// home-redacted and terminal-safe.
type Refusal struct {
	File   string `json:"file"`
	Reason string `json:"reason"`
}

// noteItems turns both note layers into items, in id order. A personal note
// carries the root it is tied to (Item.Tie), and is recalled only in that
// repository, or everywhere when its root is null; the tie is judged after
// matching (Matching), so reading the layers asks git nothing. Every note in
// the project's file is an item. A note in both layers (an id is minted once,
// so only a hand copy does that) is the project's: the machine's copy is
// suppressed.
func noteItems(machine, project []memnote.Note) []Item {
	seen := map[string]bool{}
	var out []Item
	for _, n := range project {
		seen[n.ID] = true
		out = append(out, noteItem(n, LayerProject, nil))
	}
	for _, n := range machine {
		if seen[n.ID] {
			continue
		}
		seen[n.ID] = true
		var tie *string
		if n.Root != nil {
			r := *n.Root
			tie = &r
		}
		out = append(out, noteItem(n, LayerMachine, tie))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func noteItem(n memnote.Note, layer string, tie *string) Item {
	return Item{Key: n.ID, Layer: layer, Title: n.Title, Body: n.Body, Recall: append([]string(nil), n.Recall...), Tie: tie}
}

// Layers is what a read of both note layers gave: the notes as they stand
// in the machine's file and in the project's.
type Layers struct {
	Machine []memnote.Note
	Project []memnote.Note
}

// ReadLayers is the one reader of both note layers, for the recall and for
// the memory verb alike: the machine's under home, the project's under
// repoRoot, the root the rules loader resolved, each through memnote's layer
// reader. A layer that cannot be read is a Refusal and yields no notes, never
// an error: the hook still injects the rules, and the verb still shows what
// the other layer holds. It asks git nothing.
func ReadLayers(home, repoRoot string) (Layers, []Refusal) {
	var l Layers
	var refused []Refusal
	if f, _, err := memnote.ReadMachine(home); err != nil {
		refused = append(refused, Refusal{File: memnote.MachineDisplayPath, Reason: Reason(err)})
	} else {
		l.Machine = f.Notes
	}
	if f, _, err := memnote.ReadRepo(repoRoot); err != nil {
		refused = append(refused, Refusal{File: memnote.RepoRelPath, Reason: Reason(err)})
	} else {
		l.Project = f.Notes
	}
	return l, refused
}

// ReadNotes is ReadLayers turned into the recall's items. A personal note's
// tie is judged after matching, against the RootFunc the recall is handed.
func ReadNotes(home, repoRoot string) ([]Item, []Refusal) {
	l, refused := ReadLayers(home, repoRoot)
	return noteItems(l.Machine, l.Project), refused
}

// Reason is a memory error's text as it is printed: home-redacted and
// terminal-safe.
func Reason(err error) string {
	return termsafe.Sanitize(fsutil.RedactHome(err.Error()))
}
