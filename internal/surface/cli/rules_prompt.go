package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/intentdriven/abcd/internal/core/recall"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// rules_prompt.go — the memory recall's two front doors
// (spc-2610111627213138 steps 2 to 4): the prompt router's recall after the
// rules, and the `rules --prompt "<text>"` diagnostic that shows what one
// prompt brings back.

// recallMemory is the prompt router's recall: the notes both memory layers
// hold and the pages both page scopes hold, matched against prompt and taken
// within budget, what the rules left of the loader's per-turn budget. The
// machine's are read under the home the rules user layer was
// (rules.UserHome), the project's under root, the root the loader resolved. A
// layer, scope or page that cannot be read is named on stderr and recalls
// nothing, while the rules, already rendered, still inject. The loader's kill
// switch silences the recall: nothing is read, nothing comes back, and the
// ledger is kept as it was.
func recallMemory(w io.Writer, rs rules.RuleSet, root, prompt string, ledger map[string]string, budget int) recall.Result {
	if rs.Disabled {
		return recall.Result{Ledger: ledger}
	}
	items, refused := recall.ReadItems(rules.UserHome(), root)
	for _, r := range refused {
		diagnosticLine(w, "abcd memory: %s; recalling %s from it", refusalText(r), refusalLoss(r))
	}
	return recall.Recall(recall.Request{Items: items, Prompt: prompt, Ledger: ledger, Budget: budget})
}

// refusalText words a refused layer once: the file, then why, without the
// "memory: <file>:" prefix the reader's own error already carries.
func refusalText(r recall.Refusal) string {
	why := strings.TrimPrefix(r.Reason, "memory: ")
	why = strings.TrimPrefix(why, r.File+": ")
	return r.File + ": " + why
}

// refusalLoss is what a refused file costs the recall: a memory.json's notes,
// or whatever a page store or a page would have brought back.
func refusalLoss(r recall.Refusal) string {
	if r.Kind == recall.KindPage {
		return "nothing"
	}
	return "no notes"
}

// The forms the diagnostic gives a matched domain, note or page.
const (
	promptFormFull       = "full"
	promptFormOverBudget = "over-budget"
	promptFormTooLarge   = "too-large"
)

// rulesPromptDomain is one matched domain in the diagnostic.
type rulesPromptDomain struct {
	Name   string   `json:"name"`
	Label  string   `json:"label"`
	Source string   `json:"source"`
	Hits   []string `json:"hits"`
	Form   string   `json:"form"`
}

// rulesPromptNote is one matched note in the diagnostic. Rank is its place in
// the one ranked order notes and pages share (1 first), Score its BM25F
// score, and Terms each matched term with the fields it was found in and its
// contribution (recall.Rank).
type rulesPromptNote struct {
	ID    string             `json:"id"`
	Title string             `json:"title"`
	Layer string             `json:"layer"`
	Hits  []string           `json:"hits"`
	Rank  int                `json:"rank"`
	Score float64            `json:"score"`
	Terms []recall.TermScore `json:"terms"`
	Form  string             `json:"form"`
}

// rulesPromptPage is one matched page in the diagnostic: its key is the one
// the router's envelope names it by (page:<filename>); Rank, Score and Terms
// are a note's.
type rulesPromptPage struct {
	Key      string             `json:"key"`
	Filename string             `json:"filename"`
	Title    string             `json:"title"`
	Layer    string             `json:"layer"`
	Hits     []string           `json:"hits"`
	Rank     int                `json:"rank"`
	Score    float64            `json:"score"`
	Terms    []recall.TermScore `json:"terms"`
	Form     string             `json:"form"`
}

// rulesPromptView is `rules --prompt`'s --json envelope.
type rulesPromptView struct {
	Disabled   bool                `json:"disabled"`
	Domains    []rulesPromptDomain `json:"domains"`
	Notes      []rulesPromptNote   `json:"notes"`
	Pages      []rulesPromptPage   `json:"pages"`
	Unreadable []recall.Refusal    `json:"unreadable"`
}

// rulesPrompt answers `rules --prompt "<text>"`: what that one prompt brings
// back to a fresh session. It runs the router's own path — rules.Inject on an
// empty ledger, then the recall within what the rules left — and writes
// nothing: no session state is loaded or saved.
func rulesPrompt(rs rules.RuleSet, root, prompt string) rulesPromptView {
	v := rulesPromptView{Disabled: rs.Disabled, Domains: []rulesPromptDomain{}, Notes: []rulesPromptNote{}, Pages: []rulesPromptPage{},
		Unreadable: []recall.Refusal{}}
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
	items, refused := recall.ReadItems(rules.UserHome(), root)
	v.Unreadable = append(v.Unreadable, refused...)
	mem := recall.Recall(recall.Request{Items: items, Prompt: prompt, Budget: rules.InjectionBudgetBytes - len(res.Text)})
	recalled := map[string]bool{}
	for _, key := range mem.Recalled {
		recalled[key] = true
	}
	tooLarge := map[string]bool{}
	for _, key := range mem.TooLarge {
		tooLarge[key] = true
	}
	for i, c := range recall.Rank(items, prompt) {
		form := promptFormOverBudget
		switch {
		case recalled[c.Item.Key]:
			form = promptFormFull
		case tooLarge[c.Item.Key]:
			form = promptFormTooLarge
		}
		terms := c.Terms
		if terms == nil {
			terms = []recall.TermScore{}
		}
		if c.Item.Kind == recall.KindPage {
			v.Pages = append(v.Pages, rulesPromptPage{Key: c.Item.Key, Filename: strings.TrimPrefix(c.Item.Key, recall.PageKeyPrefix),
				Title: c.Item.Title, Layer: c.Item.Layer, Hits: c.Hits, Rank: i + 1, Score: c.Score, Terms: terms, Form: form})
			continue
		}
		v.Notes = append(v.Notes, rulesPromptNote{ID: c.Item.Key, Title: c.Item.Title, Layer: c.Item.Layer, Hits: c.Hits,
			Rank: i + 1, Score: c.Score, Terms: terms, Form: form})
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
	if len(v.Notes)+len(v.Pages) > 0 {
		fmt.Fprintln(w, "Notes and pages share one ranked order (#1 first), by relevance score (BM25F); under each, every matched term, the fields it was found in and what it added.")
	}
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
			fmt.Fprintf(w, "  #%d %s  %s (%s layer) — hit by %s — score %.4f — %s\n", n.Rank, n.ID, termsafe.Sanitize(n.Title), n.Layer,
				termsafe.Sanitize(strings.Join(n.Hits, ", ")), n.Score, promptFormWords(n.Form))
			renderTermScores(w, n.Terms)
		}
	}
	if len(v.Pages) == 0 {
		fmt.Fprintln(w, "Pages: none matched")
	} else {
		fmt.Fprintln(w, "Pages:")
		for _, p := range v.Pages {
			fmt.Fprintf(w, "  #%d %s  %s (%s layer) — hit by %s — score %.4f — %s\n", p.Rank, p.Key, termsafe.Sanitize(p.Title), p.Layer,
				termsafe.Sanitize(strings.Join(p.Hits, ", ")), p.Score, promptFormWords(p.Form))
			renderTermScores(w, p.Terms)
		}
	}
	for _, u := range v.Unreadable {
		fmt.Fprintf(w, "unreadable: %s (%s recalled from it)\n", termsafe.Sanitize(refusalText(u)), refusalLoss(u))
	}
}

// renderTermScores writes, under a ranked note or page, each matched term
// with the fields it was found in and what it contributed to the score.
func renderTermScores(w io.Writer, terms []recall.TermScore) {
	for _, ts := range terms {
		var found []string
		for _, f := range []struct {
			name string
			n    int
		}{{recall.FieldRecall, ts.Recall}, {recall.FieldTitle, ts.Title}, {recall.FieldBody, ts.Body}} {
			if f.n > 0 {
				found = append(found, fmt.Sprintf("%s %d", f.name, f.n))
			}
		}
		fmt.Fprintf(w, "      %s — %s — %.4f\n", ts.Term, strings.Join(found, ", "), ts.Contribution)
	}
}

// promptFormWords is a form as the plain diagnostic says it.
func promptFormWords(form string) string {
	switch form {
	case promptFormFull:
		return "in full"
	case promptFormTooLarge:
		return "larger than the whole per-turn budget, so it does not come back in full"
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
