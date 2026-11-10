package landing

// depends.go is the enqueue step's dependency check (itd-2610170634394993,
// step 2a of spc-2610230758581403): a pull request whose body carries a
// `Depends-On: #N` line (dcn-2610180720362692) is armed only once each
// pull request it names has merged. Before it arms or toggles anything, land
// reads its own pull request and, through their own Depends-On lines, the
// pull requests it depends on, builds the records the coordination pass reads
// and acts on the pass's decision for its own pull request alone: queue goes
// on as before, wait stops with no new arming made, refuse refuses. This
// account's arming already in place is withdrawn on a wait or a refusal
// (withdrawOwn), since it could merge the pull request before what it depends
// on: a forge error in the dependency read takes it off, and the next run arms
// it again. The withdrawal runs only after the check waits or refuses. It
// reads the pull request's state whole again before its first act, acts on
// what the last read shows, and reads again after each act, three acts at
// most: inside the withdrawal, this account's queue entry is taken out by its
// node id, its arming is turned off (also beside another account's queue
// entry), and the note or refusal says which was done. An arming or queue
// entry these reads show by another account, or one the forge does not
// attribute, is left and refused, saying so. One still in place after three
// tries, or a state not read whole, refuses with the step that withdraws it by
// hand: the pull request's page for a queue entry, the command for an arming,
// both where both stand; after three tries, with the last act's failure. A
// pull request whose own read shows it merged or closed is refused as such,
// nothing withdrawn.
//
// The check runs once land has read the pull request's state on the route
// that would arm it, and on the three early returns where this account's
// arming stands at the gated head: the merge gate unread, the gate not
// arming, and a conflict. The stop is read again after the gate arms and
// before that state read, so the first two run before it and the conflict's
// after. Three returns after that read make no check: a pull request already
// in the merge queue is reported done with its dependencies unread, this
// account's own queue entry included; one another account armed, or one the
// forge does not attribute, is reported as before; and this account's arming
// at a head land did not gate is refused before the check. Every refusal
// before that point (a forge read that fails, the merge rule or the default
// branch's tip unread, a listing land cannot use, any step before enqueue)
// leaves an arming as it is with its dependencies unread.
//
// Every body and every forge answer is untrusted. A body is required of the
// pull request's own read and of each dependency still open or queued, and is
// read by coord.ParseDependsOn, which fails closed and quotes no body text; a
// merged or closed dependency's body is not read. Only numbers, the forge's
// times as UTC, a state cut to maxForgeStateRunes and sanitised, a login held
// to the forge's login alphabet and bound (shownLogin), and a forge reason
// redacted, sanitised and bounded (forgeWhy, or a refusal land composed)
// reach a note or a refusal. What only the forge can know, why a
// number has no record, is refused here before the pass: a number the forge
// answers with no pull request, a pull request into another base than the
// default branch (merged there or not), a state land cannot map, a creation
// time it cannot read or that is the zero time, and a walk past
// maxDependsReads pull requests. A dependency is satisfied once the forge
// reports it MERGED into the default branch, however it merged.
//
// The wait's start is the dependant's createdAt as the forge reports it: set
// when the pull request is opened, and no agent can move it. A reopened pull
// request keeps it, as the forge reports it; a new pull request for the same branch has its own, and so
// a fresh dependsWaitLimit. A Depends-On line added to a pull request opened
// more than dependsWaitLimit ago is refused at once, as a wait past its
// deadline. A start later than the clock (Options.Now) is refused, with no
// allowance for skew.
// Rebuilding a dependant on the default branch once its dependency merges is
// not done here (the spec's step 2b).

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/coord"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// dependsQuery reads one pull request as the dependency check needs it. Every
// field is pinned in ghcontract.
const dependsQuery = `query($o:String!,$r:String!,$n:Int!){repository(owner:$o,name:$r){pullRequest(number:$n){number state isInMergeQueue baseRefName createdAt body}}}`

const (
	// dependsWaitLimit is how long a pull request may wait on its
	// dependencies, from its createdAt, before the wait is refused: the most
	// the pass allows.
	dependsWaitLimit = coord.MaxWaitLimit
	// maxDependsReads bounds the pull requests one check reads, its own
	// included. Past it the dependencies are refused as unresolvable.
	maxDependsReads = 25
	// maxForgeStateRunes bounds a state the forge gives that land cannot map,
	// in a refusal.
	maxForgeStateRunes = 32
)

// dependsArgs are the forge client's arguments for dependsQuery on pull
// request n of owner/name: the owner and name as raw strings (-f), n as a
// number (-F).
func dependsArgs(owner, name string, n int) []string {
	return []string{"api", "graphql", "-f", "query=" + dependsQuery, "-f", "o=" + owner, "-f", "r=" + name, "-F", fmt.Sprintf("n=%d", n)}
}

// dependsPR is one pull request as dependsQuery reads it.
type dependsPR struct {
	Number         int    `json:"number"`
	State          string `json:"state"`
	IsInMergeQueue *bool  `json:"isInMergeQueue"`
	BaseRefName    string `json:"baseRefName"`
	CreatedAt      string `json:"createdAt"`
	// Body is nil when the answer has no body field: an answer land cannot
	// read whole, where an empty body is no Depends-On line.
	Body *string `json:"body"`
}

type dependsAnswer struct {
	Data struct {
		Repository *struct {
			PullRequest *dependsPR `json:"pullRequest"`
		} `json:"repository"`
	} `json:"data"`
	Errors []json.RawMessage `json:"errors"`
}

// dependsCheck is one run of the check for pull request self of owner/name,
// into def.
type dependsCheck struct {
	o                Options
	dir, owner, name string
	def              string
	self             int
	reads            int
	// none is what the check's texts say became of the arming: nothing was
	// armed, or withdrewMark, which dependsHold replaces with what the
	// withdrawal did.
	none string
	// gone is set when the pull request's own read shows it merged or
	// closed: there is no arming left to withdraw.
	gone bool
}

// cannot is the fix for a forge answer about pull request n that land cannot
// read whole.
func (c *dependsCheck) cannot(n int) string {
	return fmt.Sprintf("settle what the forge reports for #%d (gh pr view %d), then run abcd land again", n, n)
}

// rewording is the fix for a dependency n that cannot be resolved: the
// author rewords the dependant's Depends-On line.
func (c *dependsCheck) rewording(n int) string {
	return fmt.Sprintf("take #%d out of #%d's Depends-On line (gh pr edit %d --body-file <file>), or name only pull requests into %s, then run abcd land again", n, c.self, c.self, c.def)
}

// readPR reads pull request n. ok is false, with a nil error, when the forge
// answers that n is no pull request: no pull request in its data, and every
// error it gives of type NOT_FOUND at the pull request (pullRequestNotFound),
// or the forge client failing with its message for that number and nothing
// else (noSuchPR; gh exits 1 on such an answer, read 2026-11-10).
func (c *dependsCheck) readPR(n int) (dependsPR, bool, error) {
	c.reads++
	raw, err := c.o.Forge(c.dir, dependsArgs(c.owner, c.name, n)...)
	if errors.Is(err, ErrNoForgeClient) {
		return dependsPR{}, false, noForgeClient()
	}
	// The forge client exits 1 on an answer carrying errors, and its stdout,
	// the NOT_FOUND answer, is not kept: what reaches here of a number with no
	// pull request is the client's message, which names the number.
	if err != nil && noSuchPR(err, n) {
		return dependsPR{}, false, nil
	}
	if err != nil {
		return dependsPR{}, false, refuse(fmt.Sprintf("the forge could not be asked for pull request #%d, so land cannot judge #%d's dependencies, and %s: %s", n, c.self, c.none, forgeWhy(err)), c.cannot(n))
	}
	var a dependsAnswer
	if json.Unmarshal([]byte(raw), &a) != nil {
		return dependsPR{}, false, refuse(fmt.Sprintf("the forge's answer for pull request #%d does not parse, so land cannot judge #%d's dependencies, and %s", n, c.self, c.none), c.cannot(n))
	}
	var pr *dependsPR
	if a.Data.Repository != nil {
		pr = a.Data.Repository.PullRequest
	}
	if pr == nil && len(a.Errors) > 0 && pullRequestNotFound(a.Errors) {
		return dependsPR{}, false, nil
	}
	if len(a.Errors) > 0 {
		return dependsPR{}, false, refuse(fmt.Sprintf("the forge answered the query for pull request #%d with errors, so land cannot judge #%d's dependencies, and %s: %s", n, c.self, c.none, forgeErrorText(a.Errors)), c.cannot(n))
	}
	if a.Data.Repository == nil {
		return dependsPR{}, false, refuse(fmt.Sprintf("the forge answered no repository %s/%s for pull request #%d, so land cannot judge #%d's dependencies, and %s", c.owner, c.name, n, c.self, c.none), c.cannot(n))
	}
	if pr == nil {
		return dependsPR{}, false, nil
	}
	if pr.Number != n {
		return dependsPR{}, false, refuse(fmt.Sprintf("the forge answered pull request #%d when asked for #%d, so land cannot judge #%d's dependencies, and %s", pr.Number, n, c.self, c.none), c.cannot(n))
	}
	return *pr, true, nil
}

// noSuchPR reports whether err is the forge client's failure whose whole
// message is its words for no pull request n. The runner (runCapped) gives
// "gh api graphql: <exit> (<stderr>)"; those words beside any other text in
// stderr are not read as it.
func noSuchPR(err error, n int) bool {
	head, msg, ok := strings.Cut(err.Error(), " (")
	return ok && strings.HasPrefix(head, "gh api graphql: ") && msg == fmt.Sprintf("gh: Could not resolve to a PullRequest with the number of %d.)", n)
}

// pullRequestNotFound reports whether every GraphQL error is of type
// NOT_FOUND at a path ending in pullRequest: a repository not found is not a
// pull request missing.
func pullRequestNotFound(errs []json.RawMessage) bool {
	for _, raw := range errs {
		var e struct {
			Type string `json:"type"`
			Path []any  `json:"path"`
		}
		if json.Unmarshal(raw, &e) != nil || e.Type != "NOT_FOUND" || len(e.Path) == 0 || e.Path[len(e.Path)-1] != "pullRequest" {
			return false
		}
	}
	return true
}

// change is pr as the pass reads it, its own Depends-On lines read only when
// it is open or queued: a merged or closed pull request ends the walk. Its
// start is left zero: only the dependant's is read (dependsHold), and only
// the dependant's decision is acted on.
func (c *dependsCheck) change(pr dependsPR) (coord.Change, error) {
	n := pr.Number
	var st coord.State
	switch {
	case pr.State == "OPEN" && pr.IsInMergeQueue == nil:
		return coord.Change{}, refuse(fmt.Sprintf("the forge's answer for pull request #%d gives no isInMergeQueue, so land cannot judge #%d's dependencies, and %s", n, c.self, c.none), c.cannot(n))
	case pr.State == "OPEN" && *pr.IsInMergeQueue:
		st = coord.StateQueued
	case pr.State == "OPEN":
		st = coord.StateOpen
	case pr.State == "MERGED":
		st = coord.StateMerged
	case pr.State == "CLOSED":
		st = coord.StateClosed
	default:
		state := []rune(strings.TrimSpace(termsafe.Sanitize(pr.State)))
		cut := ""
		if len(state) > maxForgeStateRunes {
			state, cut = state[:maxForgeStateRunes], fmt.Sprintf(", cut to its first %d runes", maxForgeStateRunes)
		}
		return coord.Change{}, refuse(fmt.Sprintf("the forge reports pull request #%d in state %q%s, which land cannot read, so it cannot judge #%d's dependencies, and %s", n, string(state), cut, c.self, c.none), c.cannot(n))
	}
	if pr.BaseRefName != c.def {
		return coord.Change{}, refuse(fmt.Sprintf("pull request #%d merges into another branch than the default branch %s, so it cannot be resolved as a dependency of #%d, and %s", n, c.def, c.self, c.none), c.rewording(n))
	}
	ch := coord.Change{Number: n, State: st}
	if st == coord.StateOpen || st == coord.StateQueued {
		if pr.Body == nil {
			return coord.Change{}, refuse(fmt.Sprintf("the forge's answer for pull request #%d gives no body, so land cannot judge #%d's dependencies, and %s", n, c.self, c.none), c.cannot(n))
		}
		deps, err := coord.ParseDependsOn(*pr.Body)
		if err != nil {
			why := strings.TrimPrefix(err.Error(), "coord: ")
			if n == c.self {
				return coord.Change{}, refuse(fmt.Sprintf("pull request #%d's body cannot be read for its dependencies, so %s: %s", n, c.none, why),
					fmt.Sprintf("edit #%d's body on the forge (gh pr edit %d --body-file <file>), then run abcd land again", n, n))
			}
			return coord.Change{}, refuse(fmt.Sprintf("pull request #%d, which #%d depends on, has a body that cannot be read for its dependencies, so %s: %s", n, c.self, c.none, why),
				fmt.Sprintf("edit #%d's body on the forge (gh pr edit %d --body-file <file>), or take #%d out of #%d's Depends-On line, then run abcd land again", n, n, n, c.self))
		}
		ch.DependsOn = deps
	}
	return ch, nil
}

// dependsHold is the dependency check for pull request self before land arms
// or toggles it. withdraw, nil when no arming is in place, is withdrawOwn: it
// runs on a wait or a refusal, and says what it did, which the note or the
// refusal then says; a withdrawal it cannot confirm is its own refusal. held
// is true when the step stopped (a wait), and a refusal is returned as an
// error; a pull request with no Depends-On line, or whose every dependency has
// merged, is neither, and the step goes on.
func dependsHold(res *Result, o Options, dir, owner, name string, self int, withdraw func() (string, error)) (held bool, err error) {
	c := &dependsCheck{o: o, dir: dir, owner: owner, name: name, def: res.Default, self: self, none: "nothing was armed"}
	if withdraw != nil {
		c.none = withdrewMark
	}
	wait, err := c.judge(res)
	if wait == "" && err == nil {
		return false, nil
	}
	if withdraw != nil && !c.gone {
		did, werr := withdraw()
		if werr != nil {
			return false, werr
		}
		// The mark comes before any forge text in every text the check
		// composes, so the first is land's own.
		wait = strings.Replace(wait, withdrewMark, did, 1)
		var r *Refusal
		if errors.As(err, &r) {
			r.Reason = strings.Replace(r.Reason, withdrewMark, did, 1)
		}
	}
	if err != nil {
		return false, err
	}
	res.add("enqueue", StatusStopped, wait, "")
	return true, nil
}

// withdrewMark is what the check's texts say of this account's arming until
// withdrawOwn has said what it did.
const withdrewMark = "land withdrew this account's earlier arming"

// judge reads the pull requests and acts on the pass's decision for c.self:
// the wait's note for a wait, a refusal as an error, and neither when the
// step goes on.
func (c *dependsCheck) judge(res *Result) (wait string, err error) {
	self, owner, name := c.self, c.owner, c.name
	own, ok, err := c.readPR(self)
	if err != nil {
		return "", err
	}
	// Merged or closed since land read its state: nothing is left to judge,
	// and no arming to withdraw.
	if ok && (own.State == "MERGED" || own.State == "CLOSED") {
		c.gone = true
		what, fix := "merged", fmt.Sprintf("nothing is left to land for #%d (gh pr view %d)", self, self)
		if own.State == "CLOSED" {
			what, fix = "closed", fmt.Sprintf("reopen it (gh pr reopen %d) if it is still to merge, then run abcd land again", self)
		}
		return "", refuse(fmt.Sprintf("the forge reports pull request #%d %s since land read its state, so land judged none of its dependencies and armed or withdrew nothing", self, what), fix)
	}
	if !ok || own.State != "OPEN" || own.IsInMergeQueue != nil && *own.IsInMergeQueue || own.BaseRefName != res.Default {
		return "", refuse(fmt.Sprintf("the forge's answer for pull request #%d no longer reads it as open into %s and outside the merge queue, so land cannot judge its dependencies, and %s", self, res.Default, c.none),
			"run abcd land again")
	}
	ownChange, err := c.change(own)
	if err != nil {
		return "", err
	}
	if len(ownChange.DependsOn) == 0 {
		return "", nil
	}
	// The zero time is no start: the pass would read it as none recorded.
	start, err := time.Parse(time.RFC3339, own.CreatedAt)
	if err != nil || start.IsZero() {
		return "", refuse(fmt.Sprintf("the forge gives pull request #%d no creation time land can read, so no deadline can be derived for its wait on its dependencies, and %s", self, c.none), c.cannot(self))
	}
	ownChange.WaitingSince = start
	changes := []coord.Change{ownChange}
	seen := map[int]bool{self: true}
	next := slices.Clone(ownChange.DependsOn)
	for len(next) > 0 {
		slices.Sort(next)
		var later []int
		for _, n := range slices.Compact(next) {
			if seen[n] {
				continue
			}
			seen[n] = true
			if c.reads == maxDependsReads {
				return "", refuse(fmt.Sprintf("pull request #%d's dependencies reach more than %d pull requests, past what land reads, so they cannot be resolved, and %s", self, maxDependsReads, c.none),
					fmt.Sprintf("name fewer pull requests in the Depends-On lines #%d reaches (gh pr edit %d --body-file <file>), then run abcd land again", self, self))
			}
			pr, ok, err := c.readPR(n)
			if err != nil {
				return "", err
			}
			if !ok {
				return "", refuse(fmt.Sprintf("the forge has no pull request #%d in %s/%s, so it cannot be resolved as a dependency of #%d, and %s", n, owner, name, self, c.none), c.rewording(n))
			}
			ch, err := c.change(pr)
			if err != nil {
				return "", err
			}
			changes = append(changes, ch)
			later = append(later, ch.DependsOn...)
		}
		next = later
	}
	now := c.o.Now()
	decisions, err := coord.Pass(coord.Records{Now: now, WaitLimit: dependsWaitLimit, Changes: changes})
	if err != nil {
		return "", refuse(fmt.Sprintf("pull request #%d's dependencies cannot be judged, so %s: %s", self, c.none, strings.TrimPrefix(err.Error(), "coord: ")),
			fmt.Sprintf("settle what it names in the pull requests #%d depends on, then run abcd land again", self))
	}
	for _, d := range decisions {
		if d.Change != self {
			continue
		}
		on := numbered(d.WaitsOn)
		switch d.Action {
		case coord.ActionQueue:
			return "", nil
		case coord.ActionWait:
			verb := fmt.Sprintf("has not merged, so %s: run abcd land again after it merges", c.none)
			if len(d.WaitsOn) > 1 {
				verb = fmt.Sprintf("have not merged, so %s: run abcd land again after they merge", c.none)
			}
			return fmt.Sprintf("pull request #%d depends on %s, which %s; the wait is refused from %s, %d hours after #%d was opened",
				self, on, verb, d.Until.UTC().Format(time.RFC3339), int(dependsWaitLimit/time.Hour), self), nil
		}
		// The fix follows the reason: the deadline's for a wait past it, the
		// clock's only for a start later than the clock, and the reword for a
		// refusal at once.
		why, fix := d.Reason, fmt.Sprintf("reword #%d's Depends-On line (gh pr edit %d --body-file <file>) so it names only open or merged pull requests into %s with no cycle among them, or remove it, then run abcd land again", self, self, res.Default)
		switch {
		case strings.HasPrefix(d.Reason, coord.RefusedAtOnce):
		case !d.Until.IsZero():
			// Past the deadline, whether land waited or the line was added
			// after it: the text says when the wait ended, not that it waited.
			has, it := "has", "it"
			if len(d.WaitsOn) > 1 {
				has, it = "have", "them"
			}
			why = fmt.Sprintf("%s %s not merged, and the %d hours #%d may wait for %s, from its opening at %s, ended at %s",
				on, has, int(dependsWaitLimit/time.Hour), self, it, start.UTC().Format(time.RFC3339), d.Until.UTC().Format(time.RFC3339))
			fix = fmt.Sprintf("merge %s first and run abcd land again, or remove the Depends-On line from #%d's body (gh pr edit %d --body-file <file>) if #%d need not wait for it", on, self, self, self)
		case start.After(now):
			fix = "check this computer's clock against the forge's, then run abcd land again"
		}
		return "", refuse(fmt.Sprintf("pull request #%d's dependencies refuse it, so %s: %s", self, c.none, why), fix)
	}
	return "", refuse(fmt.Sprintf("the coordination pass gave pull request #%d no decision, so %s", self, c.none), "run abcd land again")
}

// numbered names the numbers as #N, comma-separated.
func numbered(nums []int) string {
	parts := make([]string, len(nums))
	for i, n := range nums {
		parts[i] = fmt.Sprintf("#%d", n)
	}
	return strings.Join(parts, ", ")
}

// maxWithdrawPasses bounds withdrawOwn: three acts, each followed by a read.
const maxWithdrawPasses = 3

// withdrawn is what withdrawOwn did.
type withdrawn struct{ disarmed, dequeued bool }

// text says what was done, as a clause: the arming off, the queue entry out,
// both, or neither, the forge showing nothing of this account's left.
func (w withdrawn) text(n string) string {
	switch {
	case w.disarmed && w.dequeued:
		return withdrewMark + " and took #" + n + " out of the merge queue"
	case w.disarmed:
		return withdrewMark
	case w.dequeued:
		return "land took #" + n + " out of the merge queue"
	}
	return "land found this account's earlier arming off on reading it again"
}

// unconfirmedFix is the hand step for pull request n in state s, which land
// could not read whole or confirm: the queue page where s shows it queued,
// never the command that only turns auto-merge off; both steps where land
// cannot say which holds.
func unconfirmedFix(n string, s prState) string {
	if s.queued() {
		return queuePageFix(n)
	}
	return "check #" + n + " (gh pr view " + n + "): withdraw an arming by this account (gh pr merge " + n + " --disable-auto) and take a queue entry by this account out of the merge queue on its page, then run abcd land again, which says what holds it"
}

// sofar is what w says was done, as a clause after the lead of a refusal,
// "" where nothing was.
func (w withdrawn) sofar(n string) string {
	if !w.disarmed && !w.dequeued {
		return ""
	}
	return "; " + w.text(n)
}

// queuePageFix is the hand step for pull request n in the merge queue.
func queuePageFix(n string) string {
	return "take #" + n + " out of the merge queue on its page, then run abcd land again, which says what holds it"
}

// maxLoginRunes bounds a login the forge gives, in a refusal: the forge's
// own bound.
const maxLoginRunes = 39

// shownLogin is how a refusal names the account login: as it is where it is
// a forge login (ownerName, within maxLoginRunes), and as one the forge names
// in a form land does not show otherwise.
func shownLogin(login string) string {
	switch {
	case login == "":
		return "an account the forge does not name"
	case ownerName(login) && len(login) <= maxLoginRunes:
		return "another account (" + login + ")"
	}
	return "an account the forge names in a form land does not show"
}

// withdrawOwn takes this account's arming and queue entry of pull request n
// off and says what it did. s, the state land read before the dependency
// check, gives the hand step only if the read before the first act fails:
// every act follows a fresh whole read (view). Each pass acts only on what the
// last read shows, and the next read decides, whatever the act returned:
// queued by this account, it is taken out by its node id first (the forge may
// refuse to turn auto-merge off on a queued pull request; unconfirmed), then
// its auto-merge off if this account armed it; armed by this account, its
// auto-merge off, also beside another account's queue entry; neither, done.
// One queued by another account, or by one the forge does not name, or armed
// so and not queued, is left and refused, saying what land did. A state not
// read whole, a read that fails, or this account's arming or queue entry
// still there after maxWithdrawPasses acts refuses with the step that
// withdraws it by hand: the queue page for a queued pull request, never the
// command that only turns auto-merge off, and both where both stand; after
// the passes, with the last act's failure (forgeWhy).
func withdrawOwn(o Options, dir, n string, s prState, view func() (prState, error)) (string, error) {
	ask := func(args ...string) (string, error) { return o.Forge(dir, args...) }
	const lead = "land's dependency check does not let pull request #%s queue%s, but "
	var did withdrawn
	// unreadAgain refuses a read that fails, giving the step for state s.
	unreadAgain := func(err error, s prState) (string, error) {
		why := forgeWhy(err)
		if r := (*Refusal)(nil); errors.As(err, &r) {
			why = r.Reason
		}
		return "", refuse(fmt.Sprintf(lead, n, did.sofar(n))+"the forge could not be read again for #"+n+"'s state ("+why+"), so land cannot confirm that this account's arming and queue entry are off", unconfirmedFix(n, s))
	}
	// fail refuses, saying what land did first: where the last read is whole
	// (shown), only what it shows done.
	fail := func(shown bool, why, fix string) (string, error) {
		w := did
		if shown {
			w.disarmed = w.disarmed && !s.ownArming()
			w.dequeued = w.dequeued && !s.ownQueue()
		}
		return "", refuse(fmt.Sprintf(lead, n, w.sofar(n))+why, fix)
	}
	// The state land read came before the dependency check's reads: it is
	// read again before the first act.
	first, err := view()
	if err != nil {
		return unreadAgain(err, s)
	}
	s = first
	// lastErr is the failure of the last pass's acts, nil where they did not
	// fail.
	var lastErr error
	for pass := 0; ; pass++ {
		if why := s.unread(); why != "" {
			return fail(false, "the forge's answer for #"+n+"'s state gives no "+why+", so land cannot confirm that this account's arming and queue entry are off", unconfirmedFix(n, s))
		}
		ownQ, ownA := s.ownQueue(), s.ownArming()
		switch {
		case s.queued() && !ownQ && !ownA:
			return fail(true, "the forge reports #"+n+" in the merge queue, queued by "+shownLogin(s.enqueuer())+", while land has not found what it depends on merged; land does not take out a queue entry it did not make",
				"ask whoever queued it to take it out of the merge queue, or take it out yourself on its page, then run abcd land again, which says what holds it")
		case s.armed() && !ownA && !s.queued():
			return fail(true, "the forge now reports #"+n+" armed by another account (or one it does not attribute), which can merge #"+n+" before what it depends on and which land does not withdraw",
				"ask whoever armed it to withdraw it, or withdraw it yourself (gh pr merge "+n+" --disable-auto), then run abcd land again")
		case !ownQ && !ownA:
			return did.text(n), nil
		case pass == maxWithdrawPasses:
			still := fmt.Sprintf("after %d tries the forge still reports #%s ", maxWithdrawPasses, n)
			last := ""
			if lastErr != nil {
				last = ", the last try failing: " + forgeWhy(lastErr)
			}
			disarm := "withdraw the arming (gh pr merge " + n + " --disable-auto)"
			switch {
			case ownQ && ownA:
				return fail(true, still+"in the merge queue from this account and armed by this account, where it can merge before what it depends on"+last,
					"take #"+n+" out of the merge queue on its page and "+disarm+", then run abcd land again, which says what holds it")
			case ownQ:
				return fail(true, still+"in the merge queue from this account, where it can merge before what it depends on"+last, queuePageFix(n))
			}
			return fail(true, still+"armed by this account, which can merge it before what it depends on"+last,
				"withdraw it yourself (gh pr merge "+n+" --disable-auto), then run abcd land again, which says what holds it")
		}
		lastErr = nil
		if ownQ {
			if err := dequeue(ask, s.ID); err != nil {
				lastErr = err
			} else {
				did.dequeued = true
			}
		}
		if ownA {
			if _, err := ask("pr", "merge", n, "--disable-auto"); err != nil {
				lastErr = err
			} else {
				did.disarmed = true
			}
		}
		next, err := view()
		if err != nil {
			return unreadAgain(err, s)
		}
		s = next
	}
}
