package sweep

// auto.go is the sweep abcd runs on its own (iss-2611150649557869,
// dcn-2611161331159806): in a repository abcd manages, once a landing sees its
// pull request merged and at the end of a drain, the provably spent part of an
// applied sweep, with exactly the proofs `--apply` uses, judged against a
// fresh fetch: the worktrees, the local branches and the merged branches on
// origin. Unattended, it deletes only the branches abcd made
// (dcn-2611161459048806, narrowed by dcn-2611161654034047): a branch with
// exactly the shape the build loop gives a lane. A worktree store worktree is
// removed when spent, but its branch, unless it has a lane's shape, is kept
// like any other branch judged deletable and listed for the applying sweep a
// person runs, who saw a dry run first: `ahoy worktree add` can check a
// person's existing branch out into the store. For the same reason a spent
// store worktree holding any ignored file outside its local tier, which a
// removal deletes unarchived (a .env, a local build config), is kept and
// listed for that sweep, and so is one whose ignored files cannot be listed
// (iss-2611161722120909). It dispatches no reservation
// workflow and files no inbox report; both stay with the sweep a person runs.
// Its answer is never an error: a refusal or a fault is carried in the
// answer, for the act it follows to report, and that act's own outcome
// stands.

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	store "github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// KeyAutomatic is the one setting that turns the automatic sweep off, in
// .abcd/config.json or ~/.abcd.noindex/config.json: a bool, on by default in
// a repository abcd manages. It only restricts (adr-2611011332427073): false
// from any layer holds, and no layer turns the sweep on outside a managed
// repository.
const KeyAutomatic = "sweep.automatic"

// autoRows is how many rows an automatic sweep's report shows before it names
// how many more there are and where the full list is.
const autoRows = 5

// autoRowCap bounds each row the report shows, in bytes.
const autoRowCap = 240

// autoFaultCap bounds the refusal or fault the report carries, in bytes.
const autoFaultCap = 480

// Setting is sweep.automatic as resolved for one repository.
type Setting struct {
	On bool
	// Why is why the sweep is off, "" when it is on.
	Why string
	// Said is whether the act's output names the off: a layer that turned it
	// off, or a setting that could not be read. Outside a managed repository
	// the sweep is off and nothing is said.
	Said bool
}

// readAutomatic resolves sweep.automatic for a repository through the layered
// configuration reader at r, which claims the sweep namespace, so a misspelt
// key is refused naming its file. managed says the repository is one abcd
// manages: outside one the sweep is off whatever a layer says. Every layer
// holding the key is decoded; a false in any holds. A file that cannot be
// read, a misspelt key or a value that is not a bool reads as off, and so does
// a home the reader refused, whose machine-wide off cannot be read; each says
// why.
func readAutomatic(r layered.Roots, managed bool) Setting {
	if !managed {
		return Setting{Why: "not a repository abcd manages"}
	}
	unread := func(err error) Setting {
		return Setting{Why: clean(KeyAutomatic + " could not be read (" + err.Error() + "), so it reads as off"), Said: true}
	}
	s, err := layered.Load(layered.Config, r)
	if err != nil {
		return unread(err)
	}
	if err := s.Claim("sweep", "automatic"); err != nil {
		return unread(err)
	}
	found, err := s.Lookup(KeyAutomatic)
	if err != nil {
		return unread(err)
	}
	off := ""
	for _, f := range found {
		v, err := layered.Decode[bool](f.Raw)
		if err != nil {
			return unread(fmt.Errorf("%s (%s layer): %v; it takes true or false", f.Origin, f.Layer, err))
		}
		if !v && off == "" {
			off = fmt.Sprintf("%s is off (%s layer, %s)", KeyAutomatic, f.Layer, f.Origin)
		}
	}
	if off != "" {
		return Setting{Why: clean(off), Said: true}
	}
	if r.HomeRefused {
		return Setting{Why: "the home directory was refused, so a machine-wide " + KeyAutomatic + " cannot be read, and it reads as off", Said: true}
	}
	return Setting{On: true}
}

// AutoOptions is one automatic sweep: the options a sweep runs with, and where
// its setting is read.
type AutoOptions struct {
	Options
	// Roots is where sweep.automatic is read; nil is layered.RootsFor at the
	// repository.
	Roots *layered.Roots
	// Managed says whether the repository is one abcd manages; nil asks
	// ahoy.ManagedContext. An error reads as not managed.
	Managed func(ctx context.Context, repoRoot string) (bool, error)
}

// Auto is what an automatic sweep came to.
type Auto struct {
	Setting Setting
	// Ran says the sweep ran: the setting was on.
	Ran    bool
	Result Result
	// Fault is the run's refusal or fault, "" when it completed.
	Fault string
	// unjudged says the run was refused before any half judged anything:
	// its fetch, or the reads before it, failed.
	unjudged bool
}

// Automatic runs the automatic sweep of the repository at repoRoot: its
// setting read first, then, when it is on, Run's fetch and an applied sweep of
// the worktrees, the local branches and the branches on origin, the branches
// it deletes held to the ones abcd made (abcdMade) and the worktrees it
// removes to ones holding no ignored file outside the local tier
// (keptForIgnored). It never returns an error
// and never panics: a refusal, a fault, or a panic in any half, is the
// answer's Fault. ctx bounds the whole of it, the fetch and each delete's
// push included; once ctx ends, it stops between removals.
func Automatic(ctx context.Context, repoRoot string, a AutoOptions) (out Auto) {
	defer func() {
		if p := recover(); p != nil {
			out.Fault = clean(fmt.Sprintf("the sweep stopped on a fault: %v", p))
		}
	}()
	managed := a.Managed
	if managed == nil {
		managed = ahoy.ManagedContext
	}
	m, err := managed(ctx, repoRoot)
	if err != nil {
		m = false
	}
	roots := a.Roots
	if roots == nil {
		r, _ := layered.RootsFor(repoRoot)
		roots = &r
	}
	out.Setting = readAutomatic(*roots, m)
	if !out.Setting.On {
		return out
	}
	out.Ran = true
	opts := a.Options
	opts.ours = abcdMade
	if opts.ignored == nil {
		opts.ignored = store.IgnoredForRemoval
	}
	o, err := prepare(ctx, repoRoot, opts)
	if err != nil {
		out.Result, _ = refused(err)
		out.Fault = out.Result.Refusal
		out.unjudged = true
		return out
	}
	res, err := runHalves(ctx, repoRoot, o, true, true)
	out.Result = res
	if err != nil {
		out.Fault = firstNonEmpty(res.Refusal, clean(err.Error()))
	}
	return out
}

// Text is the automatic sweep's report for the act it followed, one line: what
// it removed and what it left, counted, then the first rows in the sweep's own
// row wording, acted-on rows first, then how many more and where the full
// list is, then any refusal or fault, which leaves the act's outcome as it
// was. A sweep that did not run says why only when the setting is to be
// named (Setting.Said); otherwise Text is "".
func (a Auto) Text() string {
	if !a.Ran {
		if a.Setting.Said {
			return "abcd did not sweep on its own: " + a.Setting.Why
		}
		return ""
	}
	if a.unjudged {
		return "abcd's own sweep was refused before it judged anything: " + clip(a.Fault, autoFaultCap) + "; nothing was removed, and the act it followed stands"
	}
	var acted, left []Row
	removed, saved := 0, 0
	for _, r := range a.Result.Rows {
		switch r.Outcome {
		case OutcomeRemoved, OutcomeDeleted:
			removed++
			acted = append(acted, r)
		case OutcomeSaved:
			saved++
			acted = append(acted, r)
		default:
			left = append(left, r)
		}
	}
	parts := []string{fmt.Sprintf("abcd swept on its own: %d removed or deleted, %d saved as a patch and kept, %d left as they were", removed, saved, len(left))}
	shown := append(acted, left...)
	more := 0
	if len(shown) > autoRows {
		more = len(shown) - autoRows
		shown = shown[:autoRows]
	}
	for _, r := range shown {
		parts = append(parts, "["+clip(RowLine(r, true), autoRowCap)+"]")
	}
	tail := "`abcd sweep` lists every row"
	if more > 0 {
		tail = strconv.Itoa(more) + " more; " + tail
	}
	parts = append(parts, tail)
	if a.Fault != "" {
		parts = append(parts, "refused: "+clip(a.Fault, autoFaultCap)+"; the act it followed stands")
	}
	return strings.Join(parts, " ")
}

// RowLine is one row in the sweep's wording: what apply would do (applied,
// what it did), the kind, the name, the path where there is one, the state
// read, and the reason. Every field is display-ready already (Row).
func RowLine(r Row, applied bool) string {
	verb := string(r.Action)
	if applied && r.Outcome != "" {
		verb = string(r.Outcome)
	}
	fields := []string{verb, string(r.Kind), r.Name}
	if r.Path != "" {
		fields = append(fields, r.Path)
	}
	if r.State != "" {
		fields = append(fields, r.State)
	}
	line := strings.Join(fields, "  ")
	if r.Reason != "" {
		line += "; " + r.Reason
	}
	return line
}

// clip is s cut to at most n bytes at a rune boundary, marked when cut.
func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && (s[cut]&0xC0) == 0x80 {
		cut--
	}
	return s[:cut] + "…"
}

// leftForAPerson is row, a branch's, as the sweep abcd runs on its own judges
// it when ours (Options.ours) is set: a branch judged deletable that ours does
// not name is kept, saying it is left for the applying sweep a person runs,
// with the reason it was judged deletable after (dcn-2611161459048806). Any
// other row, and every row with ours nil, is row as it was.
func leftForAPerson(row Row, ours func(string) bool) Row {
	if ours == nil || row.Action != ActionDelete || ours(row.Name) {
		return row
	}
	why := "abcd deletes on its own only the branches it made, a build lane's, named build/<run-id>-<lane-id>"
	row.Action = ActionKeep
	row.Reason = clean("left for `abcd sweep --apply`: " + why + "; " + row.Reason)
	row.Evidence = append(row.Evidence, Evidence{Kind: KindProtected, Strength: Proven, Detail: why})
	return row
}

// abcdMade is the test of a branch abcd made, as the sweep abcd runs on its
// own reads it (dcn-2611161654034047): exactly the shape the build loop gives
// a lane (loop.LaneBranch). A worktree store worktree's branch is no proof:
// `ahoy worktree add` checks a person's existing branch out into the store.
func abcdMade(name string) bool { return loop.LaneBranch(name) }
