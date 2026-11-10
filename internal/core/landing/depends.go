package landing

// depends.go is the enqueue step's dependency check (itd-2610170634394993,
// step 2a of spc-2610230758581403): a pull request whose body carries a
// `Depends-On: #N` line (dcn-2610180720362692) is armed only once each
// pull request it names has merged. Before it arms or toggles anything, land
// reads its own pull request and, through their own Depends-On lines, the
// pull requests it depends on, builds the records the coordination pass reads
// and acts on the pass's decision for its own pull request alone: queue goes
// on as before, wait stops with nothing armed, refuse refuses.
//
// Every body and every forge answer is untrusted. The bodies are read by
// coord.ParseDependsOn, which fails closed and quotes no body text; only
// numbers, the forge's times as UTC and a state cut to maxForgeStateRunes,
// sanitised, reach a note or a refusal. What only the forge can know, why a
// number has no record, is refused here before the pass: a number the forge
// answers with no pull request, a pull request into another base than the
// default branch, a state land cannot map, a creation time it cannot read,
// and a walk past maxDependsReads pull requests.
//
// The wait's start is the dependant's createdAt as the forge reports it: set
// once, and no agent can move it. Its limit is dependsWaitLimit. A start
// later than the clock (Options.Now) is refused, with no allowance for skew.
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
	"github.com/intentdriven/abcd/internal/fsutil"
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
	IsInMergeQueue bool   `json:"isInMergeQueue"`
	BaseRefName    string `json:"baseRefName"`
	CreatedAt      string `json:"createdAt"`
	Body           string `json:"body"`
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
// error it gives of type NOT_FOUND, or the forge client failing with its
// message for that number (gh exits 1 on such an answer, read 2026-11-10).
func (c *dependsCheck) readPR(n int) (dependsPR, bool, error) {
	c.reads++
	raw, err := c.o.Forge(c.dir, dependsArgs(c.owner, c.name, n)...)
	if errors.Is(err, ErrNoForgeClient) {
		return dependsPR{}, false, noForgeClient()
	}
	// The forge client exits 1 on an answer carrying errors, and its stdout,
	// the NOT_FOUND answer, is not kept: what reaches here of a number with no
	// pull request is the client's message, which names the number.
	if err != nil && strings.Contains(err.Error(), fmt.Sprintf("Could not resolve to a PullRequest with the number of %d.", n)) {
		return dependsPR{}, false, nil
	}
	if err != nil {
		return dependsPR{}, false, refuse(fmt.Sprintf("the forge could not be asked for pull request #%d, so land cannot judge #%d's dependencies, and nothing was armed: %s", n, c.self, termsafe.Sanitize(fsutil.RedactHome(err.Error()))), c.cannot(n))
	}
	var a dependsAnswer
	if json.Unmarshal([]byte(raw), &a) != nil {
		return dependsPR{}, false, refuse(fmt.Sprintf("the forge's answer for pull request #%d does not parse, so land cannot judge #%d's dependencies, and nothing was armed", n, c.self), c.cannot(n))
	}
	var pr *dependsPR
	if a.Data.Repository != nil {
		pr = a.Data.Repository.PullRequest
	}
	if pr == nil && len(a.Errors) > 0 && allNotFound(a.Errors) {
		return dependsPR{}, false, nil
	}
	if len(a.Errors) > 0 {
		return dependsPR{}, false, refuse(fmt.Sprintf("the forge answered the query for pull request #%d with errors, so land cannot judge #%d's dependencies, and nothing was armed: %s", n, c.self, forgeErrorText(a.Errors)), c.cannot(n))
	}
	if a.Data.Repository == nil {
		return dependsPR{}, false, refuse(fmt.Sprintf("the forge answered no repository %s/%s for pull request #%d, so land cannot judge #%d's dependencies, and nothing was armed", c.owner, c.name, n, c.self), c.cannot(n))
	}
	if pr == nil {
		return dependsPR{}, false, nil
	}
	if pr.Number != n {
		return dependsPR{}, false, refuse(fmt.Sprintf("the forge answered pull request #%d when asked for #%d, so land cannot judge #%d's dependencies, and nothing was armed", pr.Number, n, c.self), c.cannot(n))
	}
	return *pr, true, nil
}

// allNotFound reports whether every GraphQL error is of type NOT_FOUND.
func allNotFound(errs []json.RawMessage) bool {
	for _, raw := range errs {
		var e struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(raw, &e) != nil || e.Type != "NOT_FOUND" {
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
	case pr.State == "OPEN" && pr.IsInMergeQueue:
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
		return coord.Change{}, refuse(fmt.Sprintf("the forge reports pull request #%d in state %q%s, which land cannot read, so it cannot judge #%d's dependencies, and nothing was armed", n, string(state), cut, c.self), c.cannot(n))
	}
	if pr.BaseRefName != c.def {
		return coord.Change{}, refuse(fmt.Sprintf("pull request #%d merges into another branch than the default branch %s, so it cannot be resolved as a dependency of #%d, and nothing was armed", n, c.def, c.self), c.rewording(n))
	}
	ch := coord.Change{Number: n, State: st}
	if st == coord.StateOpen || st == coord.StateQueued {
		deps, err := coord.ParseDependsOn(pr.Body)
		if err != nil {
			why := strings.TrimPrefix(err.Error(), "coord: ")
			if n == c.self {
				return coord.Change{}, refuse(fmt.Sprintf("pull request #%d's body cannot be read for its dependencies, so nothing was armed: %s", n, why),
					fmt.Sprintf("edit #%d's body on the forge (gh pr edit %d --body-file <file>), then run abcd land again", n, n))
			}
			return coord.Change{}, refuse(fmt.Sprintf("pull request #%d, which #%d depends on, has a body that cannot be read for its dependencies, so nothing was armed: %s", n, c.self, why),
				fmt.Sprintf("edit #%d's body on the forge (gh pr edit %d --body-file <file>), or take #%d out of #%d's Depends-On line, then run abcd land again", n, n, n, c.self))
		}
		ch.DependsOn = deps
	}
	return ch, nil
}

// dependsHold is the dependency check for pull request self before land arms
// or toggles it; armed says this account's arming is already in place, which
// the check leaves as it is. held is true when the step stopped (a wait), and
// a refusal is returned as an error; a pull request with no Depends-On line,
// or whose every dependency has merged, is neither, and the step goes on.
func dependsHold(res *Result, o Options, dir, owner, name string, self int, armed bool) (held bool, err error) {
	left := ""
	if armed {
		left = fmt.Sprintf("; this account's arming already in place is left as it is: withdraw it (gh pr merge %d --disable-auto) if #%d must not merge first", self, self)
	}
	c := &dependsCheck{o: o, dir: dir, owner: owner, name: name, def: res.Default, self: self}
	own, ok, err := c.readPR(self)
	if err != nil {
		return false, err
	}
	if !ok || own.State != "OPEN" || own.IsInMergeQueue || own.BaseRefName != res.Default {
		return false, refuse(fmt.Sprintf("the forge's answer for pull request #%d no longer reads it as open into %s and outside the merge queue, so land cannot judge its dependencies, and nothing was armed", self, res.Default),
			"run abcd land again")
	}
	ownChange, err := c.change(own)
	if err != nil {
		return false, err
	}
	if len(ownChange.DependsOn) == 0 {
		return false, nil
	}
	start, err := time.Parse(time.RFC3339, own.CreatedAt)
	if err != nil {
		return false, refuse(fmt.Sprintf("the forge gives pull request #%d no creation time land can read, so no deadline can be derived for its wait on its dependencies, and nothing was armed", self), c.cannot(self))
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
				return false, refuse(fmt.Sprintf("pull request #%d's dependencies reach more than %d pull requests, past what land reads, so they cannot be resolved, and nothing was armed", self, maxDependsReads),
					fmt.Sprintf("name fewer pull requests in the Depends-On lines #%d reaches (gh pr edit %d --body-file <file>), then run abcd land again", self, self))
			}
			pr, ok, err := c.readPR(n)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, refuse(fmt.Sprintf("the forge has no pull request #%d in %s/%s, so it cannot be resolved as a dependency of #%d, and nothing was armed", n, owner, name, self), c.rewording(n))
			}
			ch, err := c.change(pr)
			if err != nil {
				return false, err
			}
			changes = append(changes, ch)
			later = append(later, ch.DependsOn...)
		}
		next = later
	}
	now := o.Now()
	decisions, err := coord.Pass(coord.Records{Now: now, WaitLimit: dependsWaitLimit, Changes: changes})
	if err != nil {
		return false, refuse(fmt.Sprintf("pull request #%d's dependencies cannot be judged, so nothing was armed: %s", self, strings.TrimPrefix(err.Error(), "coord: ")),
			fmt.Sprintf("settle what it names in the pull requests #%d depends on, then run abcd land again", self))
	}
	for _, d := range decisions {
		if d.Change != self {
			continue
		}
		on := numbered(d.WaitsOn)
		switch d.Action {
		case coord.ActionQueue:
			return false, nil
		case coord.ActionWait:
			verb := "has not merged, so nothing was armed: run abcd land again after it merges"
			if len(d.WaitsOn) > 1 {
				verb = "have not merged, so nothing was armed: run abcd land again after they merge"
			}
			res.add("enqueue", StatusStopped, fmt.Sprintf("pull request #%d depends on %s, which %s; the wait is refused from %s, %d hours after #%d was opened%s",
				self, on, verb, d.Until.UTC().Format(time.RFC3339), int(dependsWaitLimit/time.Hour), self, left), "")
			return true, nil
		}
		fix := fmt.Sprintf("reword #%d's Depends-On line (gh pr edit %d --body-file <file>) so it names only open or merged pull requests into %s with no cycle among them, or remove it, then run abcd land again", self, self, res.Default)
		switch {
		case !d.Until.IsZero():
			fix = fmt.Sprintf("merge %s first and run abcd land again, or remove the Depends-On line from #%d's body (gh pr edit %d --body-file <file>) if #%d need not wait for it", on, self, self, self)
		case ownChange.WaitingSince.After(now):
			fix = "check this computer's clock against the forge's, then run abcd land again"
		}
		return false, refuse(fmt.Sprintf("pull request #%d's dependencies refuse it, so nothing was armed: %s%s", self, d.Reason, left), fix)
	}
	return false, refuse(fmt.Sprintf("the coordination pass gave pull request #%d no decision, so nothing was armed", self), "run abcd land again")
}

// numbered names the numbers as #N, comma-separated.
func numbered(nums []int) string {
	parts := make([]string, len(nums))
	for i, n := range nums {
		parts[i] = fmt.Sprintf("#%d", n)
	}
	return strings.Join(parts, ", ")
}
