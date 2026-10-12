package recall

import (
	"sort"

	"github.com/intentdriven/abcd/internal/core/memory"
)

// pages.go — memory pages join the recall (spc-2610111627213138 step 3). A
// page is an item when its frontmatter carries a recall: list; its key is
// "page:<filename>".

// PageKeyPrefix begins a page item's key; the rest is its filename.
const PageKeyPrefix = "page:"

// pageItems turns both page scopes into items, in key order. A page of the
// same <type>_<domain>_<slug> name in both scopes is the workspace's: the
// narrower scope's page is the page of that name, and the machine's is
// suppressed, whether or not the workspace page carries a recall list. A page
// with no recall: list is never an item, so it is never a candidate.
func pageItems(workspace, machine []memory.RecallPage) []Item {
	shadowed := map[string]bool{}
	var out []Item
	for _, p := range workspace {
		shadowed[p.Filename] = true
		if len(p.Recall) > 0 {
			out = append(out, pageItem(p, LayerProject))
		}
	}
	for _, p := range machine {
		if shadowed[p.Filename] || len(p.Recall) == 0 {
			continue
		}
		out = append(out, pageItem(p, LayerMachine))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func pageItem(p memory.RecallPage, layer string) Item {
	return Item{Key: PageKeyPrefix + p.Filename, Kind: KindPage, Layer: layer, Title: p.Summary, Body: p.Body,
		Recall: append([]string(nil), p.Recall...), At: p.IngestedAt}
}

// readPages reads both page scopes through the memory store's recall read —
// the workspace's .abcd/memory/ under repoRoot, the root the rules loader
// resolved, and the machine's ~/.abcd.noindex/memory/ under home, read only —
// and returns their items. A scope or a page that cannot be read is a
// Refusal and yields nothing, never an error.
func readPages(home, repoRoot string) ([]Item, []Refusal) {
	workspace, wr := memory.ReadRecallPages(repoRoot)
	machine, mr := memory.ReadMachineRecallPages(home)
	var refused []Refusal
	for _, r := range append(wr, mr...) {
		refused = append(refused, Refusal{File: r.File, Reason: reason(r.Err), Kind: KindPage})
	}
	return pageItems(workspace, machine), refused
}

// ReadItems is every note and page in reach, in id order — notes (mem-…)
// before pages (page:…) — and every layer, scope or page that could not be
// read. It is what the prompt router and `rules --prompt` hand the recall.
func ReadItems(home, repoRoot string) ([]Item, []Refusal) {
	notes, refused := readNotes(home, repoRoot)
	pages, pr := readPages(home, repoRoot)
	return append(notes, pages...), append(refused, pr...)
}
