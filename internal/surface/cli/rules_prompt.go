package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/intentdriven/abcd/internal/core/recall"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// rules_prompt.go — the memory recall's two front doors
// (spc-2610111627213138 step 2): the prompt router's recall after the rules,
// and the `rules --prompt "<text>"` diagnostic that shows what one prompt
// brings back.

// recallMemory is the prompt router's recall: the notes both memory layers
// hold, matched against prompt and taken within budget, what the rules left
// of the loader's per-turn budget. The machine layer is read under the home
// the rules user layer was (rules.UserHome), the project's under root, the
// root the loader resolved. A layer that cannot be read is named on stderr
// and recalls nothing, while the rules, already rendered, still inject; so
// is a root-commit lookup git did not answer in time, which leaves the tied
// notes out this turn. The
// loader's kill switch silences the recall: nothing is read, nothing comes
// back, and the ledger is kept as it was. here answers the repository's root
// commit for the notes tied to one, asked only once a tied note matched.
func recallMemory(w io.Writer, rs rules.RuleSet, root, prompt string, ledger map[string]string, budget int, here recall.RootFunc) recall.Result {
	if rs.Disabled {
		return recall.Result{Ledger: ledger}
	}
	items, refused := recall.ReadNotes(rules.UserHome(), root)
	for _, r := range refused {
		diagnosticLine(w, "abcd memory: %s; recalling no notes from it", refusalText(r))
	}
	res := recall.Recall(recall.Request{Items: items, Prompt: prompt, Ledger: ledger, Budget: budget, Root: here})
	if res.RootErr != nil {
		// git did not answer in time: the tied notes are left out, and,
		// unrecorded, tried again on the next turn.
		diagnosticLine(w, "abcd memory: this repository's root commit: %s; recalling no notes tied to it this turn",
			fsutil.RedactHome(res.RootErr.Error()))
	}
	return res
}

// routerRootCommit is the repository's root commit as the prompt router asks
// git for it; a seam for the tests.
var routerRootCommit = gitutil.RootCommitContext

// rootLookup is one root-commit lookup the live drain made: its answer, its
// error, and whether it was made at all (a hook with no cwd makes none).
type rootLookup struct {
	sha   string
	err   error
	asked bool
}

// promptRoot is the recall's root-commit answer for one prompt, memoised. It
// reuses the lookup the live drain already made for this prompt, so the
// router spawns that full-history rev-list once; only when the drain made
// none does it ask git itself, about root, and then only when a tied note
// matched. The drain asked about cwd and the recall needs root, the
// repository the loader resolved from that same cwd: the two are one
// repository, so they share one root commit — and where git cannot answer
// for cwd (a refused foreign-uid root, which leaves cwd as root), it cannot
// answer for root either.
func promptRoot(drained rootLookup, root string) recall.RootFunc {
	return sync.OnceValues(func() (string, error) {
		if drained.asked {
			return drained.sha, drained.err
		}
		return routerRootCommit(context.Background(), root)
	})
}

// refusalText words a refused layer once: the file, then why, without the
// "memory: <file>:" prefix the reader's own error already carries.
func refusalText(r recall.Refusal) string {
	why := strings.TrimPrefix(r.Reason, "memory: ")
	why = strings.TrimPrefix(why, r.File+": ")
	return r.File + ": " + why
}

// The forms the diagnostic gives a matched domain or note.
const (
	promptFormFull       = "full"
	promptFormOverBudget = "over-budget"
)

// rulesPromptDomain is one matched domain in the diagnostic.
type rulesPromptDomain struct {
	Name   string   `json:"name"`
	Label  string   `json:"label"`
	Source string   `json:"source"`
	Hits   []string `json:"hits"`
	Form   string   `json:"form"`
}

// rulesPromptNote is one matched note in the diagnostic.
type rulesPromptNote struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Layer string   `json:"layer"`
	Hits  []string `json:"hits"`
	Form  string   `json:"form"`
}

// rulesPromptView is `rules --prompt`'s --json envelope.
type rulesPromptView struct {
	Disabled   bool                `json:"disabled"`
	Domains    []rulesPromptDomain `json:"domains"`
	Notes      []rulesPromptNote   `json:"notes"`
	Unreadable []recall.Refusal    `json:"unreadable"`
}

// rulesPrompt answers `rules --prompt "<text>"`: what that one prompt brings
// back to a fresh session. It runs the router's own path — rules.Inject on an
// empty ledger, then the recall within what the rules left — and writes
// nothing: no session state is loaded or saved.
func rulesPrompt(rs rules.RuleSet, root, prompt string) rulesPromptView {
	v := rulesPromptView{Disabled: rs.Disabled, Domains: []rulesPromptDomain{}, Notes: []rulesPromptNote{}, Unreadable: []recall.Refusal{}}
	if rs.Disabled {
		return v
	}
	res := rules.Inject(rs, prompt, rules.SessionState{}, 0)
	injected := map[string]bool{}
	for _, name := range res.Injected {
		injected[name] = true
	}
	for _, d := range rs.Match(prompt) {
		form := promptFormOverBudget
		if injected[d.Name] {
			form = promptFormFull
		}
		v.Domains = append(v.Domains, rulesPromptDomain{Name: d.Name, Label: rules.Label(d.Name, d.Source), Source: d.Source,
			Hits: rules.Hits(d, prompt), Form: form})
	}
	items, refused := recall.ReadNotes(rules.UserHome(), root)
	v.Unreadable = append(v.Unreadable, refused...)
	here := promptRoot(rootLookup{}, root)
	mem := recall.Recall(recall.Request{Items: items, Prompt: prompt, Budget: rules.InjectionBudgetBytes - len(res.Text), Root: here})
	recalled := map[string]bool{}
	for _, key := range mem.Recalled {
		recalled[key] = true
	}
	matched, _ := recall.Matching(items, prompt, here)
	for _, c := range matched {
		form := promptFormOverBudget
		if recalled[c.Item.Key] {
			form = promptFormFull
		}
		v.Notes = append(v.Notes, rulesPromptNote{ID: c.Item.Key, Title: c.Item.Title, Layer: c.Item.Layer, Hits: c.Hits, Form: form})
	}
	return v
}

// renderRulesPrompt writes the diagnostic for a person.
func renderRulesPrompt(w io.Writer, rs rules.RuleSet, v rulesPromptView) {
	if v.Disabled {
		fmt.Fprintf(w, "abcd rules — disabled (kill switch set in %s); this prompt brings back nothing\n", strings.Join(killSwitchFiles(rs), " and "))
		return
	}
	fmt.Fprintln(w, "abcd rules --prompt — what this prompt brings back to a fresh session (nothing written)")
	if len(v.Domains) == 0 {
		fmt.Fprintln(w, "Domains: none matched")
	} else {
		fmt.Fprintln(w, "Domains:")
		for _, d := range v.Domains {
			fmt.Fprintf(w, "  %s — hit by %s — %s\n", d.Label, termsafe.Sanitize(strings.Join(d.Hits, ", ")), promptFormWords(d.Form))
		}
	}
	if len(v.Notes) == 0 {
		fmt.Fprintln(w, "Notes: none matched")
	} else {
		fmt.Fprintln(w, "Notes:")
		for _, n := range v.Notes {
			fmt.Fprintf(w, "  %s  %s (%s layer) — hit by %s — %s\n", n.ID, termsafe.Sanitize(n.Title), n.Layer,
				termsafe.Sanitize(strings.Join(n.Hits, ", ")), promptFormWords(n.Form))
		}
	}
	for _, u := range v.Unreadable {
		fmt.Fprintf(w, "unreadable: %s (no notes recalled from it)\n", refusalText(u))
	}
}

// promptFormWords is a form as the plain diagnostic says it.
func promptFormWords(form string) string {
	if form == promptFormFull {
		return "in full"
	}
	return "does not fit this turn's budget"
}

// killSwitchFiles names every file that set the kill switch: the switch is
// sticky across layers, so each has to clear it.
func killSwitchFiles(rs rules.RuleSet) []string {
	var files []string
	for _, src := range rs.KillSwitchSources() {
		files = append(files, rules.LayerPath(src))
	}
	if len(files) == 0 {
		files = []string{rules.RepoRelPath}
	}
	return files
}
