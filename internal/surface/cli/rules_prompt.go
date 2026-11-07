package cli

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/recall"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/intentdriven/abcd/internal/textwidth"
)

// rules_prompt.go — the memory recall's two front doors
// (spc-2610111627213138 steps 2 to 4): the prompt router's recall after the
// rules, and the `rules --prompt "<text>"` diagnostic that shows what one
// prompt brings back.

// injectWithRecall is the turn's one use of the budget, the hook's and the
// diagnostic's alike: the rules within what reserved leaves less the memory floor,
// then the recall within what the rules left. Whatever of the floor the notes
// leave unused goes back: the rules are rendered again within what the notes
// really took, since Inject writes nothing and rules that fit are sent in full
// (spc-2610111627213138).
func injectWithRecall(rs rules.RuleSet, prompt string, prev rules.SessionState, backstop int, bracket rules.Bracket, reserved int,
	recallWithin func(ledger map[string]string, budget int) recall.Result) (rules.InjectResult, recall.Result) {
	res := rules.InjectWithin(rs, prompt, prev, backstop, bracket, rules.RulesBudget(reserved))
	// The ledger Inject hands back, which its backstop may have cleared.
	mem := recallWithin(res.State.Recalled, max(0, rules.InjectionBudgetBytes-reserved-len(res.Text)))
	if len(mem.Text) < rules.MemoryFloorBytes {
		res = rules.InjectWithin(rs, prompt, prev, backstop, bracket, max(0, rules.InjectionBudgetBytes-reserved-len(mem.Text)))
	}
	return res, mem
}

// recallMemory is the prompt router's recall: the notes both memory layers
// hold and the pages both page scopes hold, matched against prompt and taken
// within budget, what the rules left of the loader's per-turn budget. The
// machine's are read under the home the rules user layer was
// (rules.UserHome), the project's under root, the root the loader resolved. A
// layer, scope or page that cannot be read is named on stderr and recalls
// nothing, while the rules, already rendered, still inject; so is a
// root-commit lookup git did not answer in time, which leaves the tied notes
// out this turn. The loader's kill switch silences the recall: nothing is
// read, nothing comes back, and the ledger is kept as it was. here answers
// the repository's root commit for the notes tied to one, asked only once a
// tied note matched.
//
// It runs after the rules are rendered, so a panic inside it is recovered
// here: that prompt recalls nothing, the ledger is kept as it was, stderr
// names the failure, and the rules already rendered still inject.
//
// bracket is the context bracket the session's fill put this prompt in
// (rules.BracketFor), whose row says which candidates come back in full and
// which as one line each.
func recallMemory(w io.Writer, rs rules.RuleSet, root, prompt string, ledger map[string]string, budget int, here recall.RootFunc,
	bracket rules.Bracket) (res recall.Result) {
	defer func() {
		if p := recover(); p != nil {
			diagnosticLine(w, "abcd memory: the recall failed (%s); recalling no notes or pages this turn", fsutil.RedactHome(fmt.Sprint(p)))
			res = recall.Result{Ledger: ledger}
		}
	}()
	if rs.Disabled || stageActive() {
		return recall.Result{Ledger: ledger}
	}
	items, refused := recall.ReadItems(rules.UserHome(), root)
	for _, r := range refused {
		diagnosticLine(w, "abcd memory: %s; recalling %s from it", refusalText(r), refusalLoss(r))
	}
	res = recall.Recall(recall.Request{Items: items, Prompt: prompt, Ledger: ledger, Budget: budget, Root: here, Bracket: bracket})
	if res.RootErr != nil {
		// git did not answer in time: the tied notes are left out, and,
		// unrecorded, tried again on the next turn.
		rootErrLine(w, res.RootErr)
	}
	return res
}

// rootErrLine names, on stderr, a root-commit lookup git did not answer in
// time, and that the notes tied to the repository are left out for it. The
// hook and `rules --prompt` both say it in these words.
func rootErrLine(w io.Writer, err error) {
	diagnosticLine(w, "abcd memory: this repository's root commit: %s; recalling no notes tied to it this turn",
		fsutil.RedactHome(err.Error()))
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
// matched or, in `rules --prompt`, a managed checkout's away record is read.
// The drain asked about cwd and the recall needs root, the
// repository the loader resolved from that same cwd: the two are one
// repository, so they share one root commit — and where git cannot answer
// for cwd (a refused foreign-uid root, which leaves cwd as root), it cannot
// answer for root either.
//
// The lookup it makes itself is boundedRootCommit, as the drain's is, so the
// tied notes are left out this turn on a timeout and the recall's caller
// names it on stderr.
func promptRoot(drained rootLookup, root string) recall.RootFunc {
	return sync.OnceValues(func() (string, error) {
		if drained.asked {
			return drained.sha, drained.err
		}
		return boundedRootCommit(root)
	})
}

// promptRootTimeout bounds every root-commit lookup the prompt router makes.
const promptRootTimeout = 2 * time.Second

// boundedRootCommit is the prompt router's one root-commit lookup, the live
// drain's and the recall's alike: routerRootCommit under promptRootTimeout.
// The hook runs on every prompt, so a git that never answers must cost that
// prompt a moment, not the session. A lookup past its deadline is a timeout
// wrapping gitutil.ErrGitTimedOut, so a caller tells "git did not answer"
// from "no root commit" with errors.Is.
func boundedRootCommit(root string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), promptRootTimeout)
	defer cancel()
	sha, err := routerRootCommit(ctx, root)
	if errors.Is(err, context.DeadlineExceeded) {
		return "", fmt.Errorf("%w (no answer within %s)", gitutil.ErrGitTimedOut, promptRootTimeout)
	}
	return sha, err
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

// The forms the diagnostic gives a matched domain, note or page: in full,
// shrunk to its heading and first rule (a domain, by its bracket or the
// budget), as its heading and the command alone (a domain whose first rule
// does not fit), as one line (a note or page), not fitting this turn's budget,
// or larger than the whole budget.
const (
	promptFormFull       = "full"
	promptFormShrunk     = "shrunk"
	promptFormPointer    = "pointer"
	promptFormLine       = "one-line"
	promptFormOverBudget = "over-budget"
	promptFormTooLarge   = "too-large"
)

// The sources of the diagnostic's bracket: the --fill flag, or none, which
// is an unknown fill.
const (
	promptBracketFromFlag    = "flag"
	promptBracketFromUnknown = rules.FillSourceUnknown
)

// rulesPromptBracket is the bracket the diagnostic rendered in and how it was
// chosen: from --fill, the share of the window used, or, without it, unknown,
// since the diagnostic has no session to read a fill for. Describe is the
// bracket table's own words for the row.
type rulesPromptBracket struct {
	Name           string   `json:"name"`
	Source         string   `json:"source"`
	UsedPercentage *float64 `json:"used_percentage,omitempty"`
	Describe       string   `json:"describe"`

	bracket rules.Bracket
}

// promptBracket is the diagnostic's bracket: from used, the --fill value,
// when it was given, and unknown otherwise.
func promptBracket(used *float64) rulesPromptBracket {
	b := rulesPromptBracket{Source: promptBracketFromUnknown, bracket: rules.BracketUnknown}
	if used != nil {
		u := *used
		b = rulesPromptBracket{Source: promptBracketFromFlag, UsedPercentage: &u, bracket: rules.BracketForUsed(u)}
	}
	b.Name = b.bracket.String()
	b.Describe = b.bracket.Row().Describe()
	return b
}

// bracketHelp lists the bracket table's rows in its own words, for a help
// text: one list item per row, wrapped to the help's measure, so the help
// cannot state a threshold the table does not hold.
func bracketHelp() string {
	var items []string
	for _, r := range rules.BracketTable() {
		lines := textwidth.Greedy(strings.Fields(r.Bracket.String()+" ("+r.When()+"): "+r.Sends()+"."), 74)
		items = append(items, "- "+strings.Join(lines, "\n  "))
	}
	return strings.Join(items, "\n")
}

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
	Bracket    rulesPromptBracket  `json:"bracket"`
	Domains    []rulesPromptDomain `json:"domains"`
	Notes      []rulesPromptNote   `json:"notes"`
	Pages      []rulesPromptPage   `json:"pages"`
	Unreadable []recall.Refusal    `json:"unreadable"`
}

// promptReserved is the bytes the away block takes off the turn's budget before
// the rules, read as the hook reads it. The status block the hook also reads
// first is left out: reading it takes in the status branches, which writes the
// local tier, and this diagnostic writes nothing. An unmanaged checkout holds no
// away record, and a drive's stage reads none, so the root commit is not looked
// up for either. A lookup git did not answer leaves the away record unread, as in
// the hook, which says nothing of it; the diagnostic names it on stderr, since a
// fuller form than the hook sends would otherwise be shown without a word.
func promptReserved(stderr io.Writer, root string, here recall.RootFunc) int {
	if root == "" || stageActive() || !autonomy.Managed(root) {
		return 0
	}
	sha, err := here()
	if err != nil {
		diagnosticLine(stderr, "abcd mode: this repository's root commit: %s; the away record was not read, so nothing is reserved for it",
			fsutil.RedactHome(err.Error()))
	}
	return len(readPromptAway(stderr, rootLookup{sha: sha, err: err, asked: true}, root).Block)
}

// rulesPrompt answers `rules --prompt "<text>"`: what that one prompt brings
// back to a fresh session. It runs the router's own path — the rules on an
// empty ledger within what the away block leaves, then the recall
// within what the rules left, both in the bracket used (the --fill value, or
// nil for an unknown fill) puts it in — and writes nothing: no session state
// is loaded or saved. A root-commit
// lookup git did not answer in time leaves the tied notes out of the view,
// and is named on stderr, once for the recall when a tied note matched, as the
// hook names it, and once more for the away record in a managed checkout. On a
// drive's stage, where the hook reads neither, neither is looked up.
func rulesPrompt(stderr io.Writer, rs rules.RuleSet, root, prompt string, used *float64) rulesPromptView {
	v := rulesPromptView{Disabled: rs.Disabled, Bracket: promptBracket(used), Domains: []rulesPromptDomain{}, Notes: []rulesPromptNote{},
		Pages: []rulesPromptPage{}, Unreadable: []recall.Refusal{}}
	if rs.Disabled {
		return v
	}
	bracket := v.Bracket.bracket
	// The hook's own cap, so a word past it matches here no more than there.
	prompt = rules.CapPrompt(prompt)
	// On a drive's stage the hook recalls no notes or pages (recallMemory), so
	// nothing is read for them here either: the view shows only what that agent
	// gets, and no root-commit lookup is made for a tied note.
	var items []recall.Item
	if !stageActive() {
		var refused []recall.Refusal
		items, refused = recall.ReadItems(rules.UserHome(), root)
		v.Unreadable = append(v.Unreadable, refused...)
	}
	here := promptRoot(rootLookup{}, root)
	res, mem := injectWithRecall(rs, prompt, rules.SessionState{}, 0, bracket, promptReserved(stderr, root, here), func(_ map[string]string, budget int) recall.Result {
		return recall.Recall(recall.Request{Items: items, Prompt: prompt, Budget: budget, Root: here, Bracket: bracket})
	})
	injected := map[string]bool{}
	for _, name := range res.Injected {
		injected[name] = true
	}
	shrunk := map[string]bool{}
	for _, name := range res.Shrunk {
		shrunk[name] = true
	}
	pointed := map[string]bool{}
	for _, name := range res.Pointed {
		pointed[name] = true
	}
	for _, d := range rs.Match(prompt) {
		form := promptFormOverBudget
		switch {
		case pointed[d.Name]:
			form = promptFormPointer
		case shrunk[d.Name]:
			form = promptFormShrunk
		case injected[d.Name]:
			form = promptFormFull
		}
		v.Domains = append(v.Domains, rulesPromptDomain{Name: d.Name, Label: rules.Label(d.Name, d.Source), Source: d.Source,
			Hits: rules.Hits(d, prompt), Form: form})
	}
	tooLarge := map[string]bool{}
	for _, key := range mem.TooLarge {
		tooLarge[key] = true
	}
	ranked, rankErr := recall.Rank(items, prompt, here)
	// here is memoised, so the recall and the ranking share one lookup and
	// one error: it is named once.
	if rootErr := cmp.Or(mem.RootErr, rankErr); rootErr != nil {
		rootErrLine(stderr, rootErr)
	}
	for i, c := range ranked {
		form := promptFormOverBudget
		switch {
		case mem.Form(c.Item.Key) == recall.FormFull:
			form = promptFormFull
		case mem.Form(c.Item.Key) == recall.FormLine:
			form = promptFormLine
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
	if v.Bracket.UsedPercentage != nil {
		fmt.Fprintf(w, "Bracket: %s (--fill %s: %s%% of the window used) — %s\n", v.Bracket.Name, formatPercent(*v.Bracket.UsedPercentage),
			formatPercent(*v.Bracket.UsedPercentage), v.Bracket.Describe)
	} else {
		fmt.Fprintf(w, "Bracket: %s (no --fill given, and with no session the fill is unknown), rendering the unknown-fill form: %s. --fill <percent-used> renders another bracket.\n",
			v.Bracket.Name, v.Bracket.bracket.Row().Sends())
	}
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

// formatPercent writes a share of the window as a person reads it: no
// trailing zeros.
func formatPercent(p float64) string {
	return strconv.FormatFloat(p, 'f', -1, 64)
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
	case promptFormShrunk:
		return "as its heading and first rule, with the line naming the command that renders the rest"
	case promptFormPointer:
		return "as its heading and the command that renders it, no room for a rule"
	case promptFormLine:
		return "as one line: its key and title, without its body"
	case promptFormTooLarge:
		return "too long for any turn even as one line, so it does not come back"
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
