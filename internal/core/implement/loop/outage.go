package loop

// outage.go is the loop's half of a lost connection (iss-2610080620372731):
// the run's one outage record lives in the shared run state
// (implement.Outages), and the loop reads it before every move. While the
// network is down a lane at a step that reaches the remote or the forge (the
// landing's push, its pull request, the arming, the merged check, a hold's
// disarm, a discard's close) waits on the one shared probe, as a landing waits
// on its merge: a contention that holds only its own lane, while the lanes
// with offline work keep moving. While the model service is down no agent is
// handed work, since the agent would meet the same outage; a network outage
// holds only the agents a runner starts, which reach the model over the
// network from this machine. A lane that meets a network failure the
// classifier recognises waits the same way, and the call records the outage
// once the run's own lock is released.
//
// The tier lock (this checkout's runs) and the shared run's lock are never
// nested: the outage is read without a lock (it is replaced whole), and the
// probe and the record are taken before and after the tier lock.
//
// The probe falls due on the shared schedule. A step that finds it due runs
// it, for the network only: the model service is proven back by the lead's
// canary alone (`abcd implement outage probe --model ok|fail`), so an outage
// with the model down is left to the lead's probe. Once the run has given up,
// every step refuses at the stage `outage`, naming what was done and what is
// left.

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// StageOutage is the refusal stage of a run that gave up on an outage, and
// DrainStoppedOutage why a drain ended on one.
const (
	StageOutage        = "outage"
	DrainStoppedOutage = "outage"
)

// OutageInfo is what an outage holds a refusal or a step to: when it began,
// the services down, the probes so far and the next one due; for a run that
// gave up, when it did, what the run had done and what is left, and whether
// the product thinker's notification is still pending.
type OutageInfo struct {
	Since       time.Time  `json:"since"`
	Services    []string   `json:"services"`
	Down        []string   `json:"down"`
	Probes      int        `json:"probes"`
	NextProbeAt *time.Time `json:"next_probe_at,omitempty"`
	// Retried is what the lanes were doing when they lost the connection.
	Retried  []string    `json:"retried"`
	GaveUpAt *time.Time  `json:"gave_up_at,omitempty"`
	Done     *OutageDone `json:"done,omitempty"`
	Left     *OutageLeft `json:"left,omitempty"`
	// Notify is true from the give-up until a session acknowledges it.
	Notify bool `json:"notify"`
}

// OutageDone is what a run that gave up had done: the lanes landed and the
// pull requests it opened that have not landed.
type OutageDone struct {
	Landed       []string   `json:"landed"`
	PullRequests []OutagePR `json:"pull_requests"`
}

// OutagePR is one pull request a lane opened.
type OutagePR struct {
	Lane string `json:"lane"`
	PR   int    `json:"pr"`
	// Merge is what the landing did with it: armed, or left open.
	Merge string `json:"merge,omitempty"`
}

// OutageLeft is what a run that gave up has left: each lane with work and its
// stage, and the spec steps no lane has opened for.
type OutageLeft struct {
	Lanes   []OutageLane  `json:"lanes"`
	Pending []PendingStep `json:"pending"`
}

// OutageLane is one lane left, at its stage.
type OutageLane struct {
	Lane     string `json:"lane"`
	SpecStep int    `json:"spec_step"`
	Stage    Stage  `json:"stage"`
}

// outageReport is a lost connection a stage met, for the call to record once
// the tier lock is released.
type outageReport struct {
	service, kind, lane, what string
}

// outageGate is what the run's outage holds one call to.
type outageGate struct {
	run     *implement.Run
	session string
	cur     *implement.Outage
	now     time.Time
	// runner reports whether role is started through a runner (Drive).
	runner func(role string) bool
	// reports are the lost connections the call's stages met.
	reports []outageReport
	// recordErr is why a report could not be recorded.
	recordErr error
}

// outageFor reads the run's outage for one call and, when its network probe is
// due, runs it first, outside every lock. A checkout with no shared run state
// has no outage. The session the outage is recorded and probed for is the
// caller's (Options.Session), or else the session whose claim names the run
// as its lane (`abcd build --session`); with neither, the outage still holds
// the lanes, and the probe and the record are left to `abcd implement outage`.
func outageFor(repoRoot, runID string, o Options) (*outageGate, error) {
	g := &outageGate{now: o.now()}
	sha := gitutil.RootCommit(repoRoot)
	if !gitutil.IsFullSHA(sha) {
		return g, nil
	}
	run, err := implement.Peek(sha)
	if err != nil {
		return nil, err
	}
	run.Now = o.Now
	g.run = run
	g.session = o.Session
	if g.session == "" {
		if claims, err := run.Claims(); err == nil {
			for _, c := range claims {
				if c.Lane == runID && c.Session != "" && !c.Unreadable {
					g.session = c.Session
					break
				}
			}
		}
	}
	if g.cur, err = run.Outage().Current(); err != nil {
		return nil, outageUnreadable(err)
	}
	if g.probeDue() {
		probe := o.NetworkProbe
		if probe == nil {
			probe = implement.RemoteProbe(repoRoot)
		}
		// A probe another session holds, or one that is refused, leaves the
		// outage as it was: the lanes keep waiting on it.
		_, _ = run.Outage().ProbeIfDue(g.session, implement.Prober{Network: probe})
		if g.cur, err = run.Outage().Current(); err != nil {
			return nil, outageUnreadable(err)
		}
	}
	return g, nil
}

// outageUnreadable refuses a call whose outage record cannot be read: the
// loop cannot tell whether the connection is down.
func outageUnreadable(err error) error {
	return refuse(StageOutage, "", "", fsutil.RedactHome(err.Error()),
		"restore or remove the outage record the reason names, then run `abcd implement step` again")
}

// probeDue reports whether this call runs the shared probe: the outage is
// open, the network is the only service down, its probe is due, and a session
// is there to run it for.
func (g *outageGate) probeDue() bool {
	return g.open() && g.session != "" && slices.Equal(g.cur.Down, []string{implement.ServiceNetwork}) &&
		!g.now.Before(g.cur.NextProbeAt)
}

func (g *outageGate) open() bool {
	return g != nil && g.cur != nil && g.cur.Status == implement.OutageOpen
}

func (g *outageGate) gaveUp() bool {
	return g != nil && g.cur != nil && g.cur.Status == implement.OutageGaveUp
}

// down reports whether the open outage has svc down.
func (g *outageGate) down(svc string) bool {
	return g.open() && slices.Contains(g.cur.Down, svc)
}

// holdsMove reports whether the network outage holds lane w's binary move.
func (g *outageGate) holdsMove(st State, w want) bool {
	return g.down(implement.ServiceNetwork) && networkMove(st, w)
}

// holdsAgent reports whether the outage holds handing role an agent: the model
// service down holds every agent; the network down holds those a runner starts.
func (g *outageGate) holdsAgent(role string) bool {
	if g.down(implement.ServiceModel) {
		return true
	}
	return g.down(implement.ServiceNetwork) && g.runner != nil && g.runner(role)
}

// networkMove reports whether lane w's binary move reaches the remote or the
// forge: the landing past its records commit (the push, the pull request, the
// arming, the merged check), and a hold that must disarm an armed pull request.
// A landing's sync and its records commit are offline.
func networkMove(st State, w want) bool {
	l := st.Lanes[w.lane]
	if l.Stage != StageLand {
		return false
	}
	ld := l.Landing
	if w.hold {
		return ld != nil && ld.Pushed != "" && ld.Merge != "" && ld.Armed
	}
	return ld != nil && ld.RecordsDone
}

// nextProbe is when the shared probe falls due: the open outage's, or, for an
// outage this call is opening, a schedule's first wait from now.
func (g *outageGate) nextProbe() time.Time {
	if g.open() {
		return g.cur.NextProbeAt
	}
	return implement.DefaultSchedule.Next(g.now, 0)
}

// info is the outage as a refusal or a result carries it; nil when none is
// open and none is being opened.
func (g *outageGate) info() *OutageInfo {
	if g.cur == nil {
		if len(g.reports) == 0 {
			return nil
		}
		next := g.nextProbe()
		down := []string{}
		retried := []string{}
		for _, r := range g.reports {
			down = addService(down, r.service)
			if !slices.Contains(retried, r.what) {
				retried = append(retried, r.what)
			}
		}
		return &OutageInfo{Since: g.now, Services: down, Down: down, NextProbeAt: &next, Retried: retried}
	}
	c := g.cur
	in := &OutageInfo{Since: c.StartedAt, Services: slices.Clone(c.Services), Down: slices.Clone(c.Down), Probes: len(c.Probes),
		Retried: c.Retried(), GaveUpAt: c.GaveUpAt, Notify: c.NotifyPending}
	if c.Status == implement.OutageOpen {
		next := c.NextProbeAt
		in.NextProbeAt = &next
	}
	return in
}

func addService(set []string, s string) []string {
	if !slices.Contains(set, s) {
		set = append(set, s)
		slices.Sort(set)
	}
	return set
}

// waitText is how a lane held by the outage waits: on the shared probe of the
// service, due next at HH:MM (UTC).
func (g *outageGate) waitText(svc string) string {
	next := g.nextProbe().UTC().Format("15:04")
	return "waits on the shared " + svc + " probe, next " + next
}

// waitRemedy is what the caller does while a lane waits on svc's probe.
func (g *outageGate) waitRemedy(svc string) string {
	if svc == implement.ServiceModel {
		return "keep to offline work; the model service is proven back only by the lead's canary: run one and report it with `abcd implement outage probe --session <id> --model ok|fail` once the next probe is due (`abcd implement outage` shows the outage)"
	}
	return "keep to offline work; `abcd implement step` at or after the next probe runs it, and the lane moves once the network is back (`abcd implement outage` shows the outage)"
}

// networkWait is the contention that holds lane's network move, what, at
// stage while the network is down.
func (g *outageGate) networkWait(stage, lane, what string) *Refusal {
	r := contend(stage, "", lane, what+" reaches the network, which is down: "+lane+" "+g.waitText(implement.ServiceNetwork),
		g.waitRemedy(implement.ServiceNetwork))
	r.Outage = g.info()
	return r
}

// agentWait is the contention that holds handing work to an agent while the
// outage holds it.
func (g *outageGate) agentWait(st State, w want) *Refusal {
	svc := implement.ServiceModel
	if !g.down(implement.ServiceModel) {
		svc = implement.ServiceNetwork
	}
	who := waitKey(st, w)
	stage := string(StageImplement)
	if w.lane >= 0 {
		stage = string(st.Lanes[w.lane].Stage)
	}
	lane := ""
	if w.open < 0 {
		lane = who
	}
	why := "the model service is down"
	if svc == implement.ServiceNetwork {
		why = "the network is down, and the " + w.role + " is started through a runner that reaches the model over it"
	}
	r := contend(stage, "", lane, fmt.Sprintf("the %s of %s is not started: %s; it %s", w.role, who, why, g.waitText(svc)), g.waitRemedy(svc))
	r.Outage = g.info()
	return r
}

// netFailure returns the lane-only contention a network failure of what makes,
// carrying the report the call records; nil when err is not a network failure
// (an authentication failure, a hook's refusal, a conflict), which stays the
// stage's own refusal.
func netFailure(stage Stage, lane, what string, err error) *Refusal {
	if err == nil || !implement.IsNetworkFailure(err.Error()) {
		return nil
	}
	r := contend(string(stage), "", lane, lane+" lost the network at "+what+": "+fsutil.RedactHome(err.Error()),
		"keep to offline work; the run waits on the shared network probe and the lane retries once it proves the network back")
	r.outage = &outageReport{service: implement.ServiceNetwork, kind: implement.OutageTool, lane: lane, what: what}
	return r
}

// take collects the report a refusal carries and rewrites it to name the
// shared probe it now waits on; it reports whether r carried one.
func (g *outageGate) take(r *Refusal) bool {
	if r == nil || r.outage == nil {
		return false
	}
	g.reports = append(g.reports, *r.outage)
	r.Reason += "; it " + g.waitText(r.outage.service)
	r.Outage = g.info()
	return true
}

// record writes each report the call collected to the shared outage, once
// the tier lock is released. A report that cannot be recorded (no session
// known, or one the run does not hold) is named, not dropped.
func (g *outageGate) record() {
	if g == nil || len(g.reports) == 0 {
		return
	}
	if g.run == nil || g.session == "" {
		g.recordErr = errors.New("no session of the shared run is known for this run (start it with `abcd build --session`)")
		return
	}
	for _, r := range g.reports {
		cur, err := g.run.Outage().Record(g.session, r.service, r.kind, r.lane, r.what)
		if err != nil {
			g.recordErr = err
			return
		}
		g.cur = &cur
	}
}

// recordNote is what the caller is told when the outage could not be recorded.
func (g *outageGate) recordNote() string {
	if g == nil || g.recordErr == nil {
		return ""
	}
	return "the outage was not recorded in the shared run (" + fsutil.RedactHome(g.recordErr.Error()) +
		"); record it with `abcd implement outage record --session <id> --service network --kind tool --lane <lane> --what <what>`"
}

// stopRefusal is every step's answer once the run gave up on the outage: what
// the run had done, what is left, and the notification.
func (g *outageGate) stopRefusal(st State) *Refusal {
	in := g.info()
	done := &OutageDone{Landed: []string{}, PullRequests: []OutagePR{}}
	left := &OutageLeft{Lanes: []OutageLane{}, Pending: slices.Clone(st.Pending)}
	if left.Pending == nil {
		left.Pending = []PendingStep{}
	}
	for _, i := range st.laneOrder() {
		l := st.Lanes[i]
		switch {
		case l.Stage == StageDone:
			done.Landed = append(done.Landed, l.ID)
			continue
		case l.PR > 0:
			pr := OutagePR{Lane: l.ID, PR: l.PR}
			if l.Landing != nil {
				pr.Merge = l.Landing.Merge
			}
			done.PullRequests = append(done.PullRequests, pr)
		}
		if l.Stage != StageDiscarded {
			left.Lanes = append(left.Lanes, OutageLane{Lane: l.ID, SpecStep: l.SpecStep, Stage: l.Stage})
		}
	}
	in.Done, in.Left = done, left

	var did, todo []string
	if len(done.Landed) > 0 {
		did = append(did, "landed "+strings.Join(done.Landed, ", "))
	}
	for _, p := range done.PullRequests {
		did = append(did, fmt.Sprintf("%s's pull request #%d is open", p.Lane, p.PR))
	}
	if len(did) == 0 {
		did = append(did, "nothing landed")
	}
	for _, l := range left.Lanes {
		todo = append(todo, l.Lane+" at "+string(l.Stage))
	}
	for _, p := range left.Pending {
		todo = append(todo, fmt.Sprintf("step %d pending", p.Number))
	}
	if len(todo) == 0 {
		todo = append(todo, "nothing")
	}
	gave := "an unrecorded time"
	if in.GaveUpAt != nil {
		gave = in.GaveUpAt.UTC().Format(time.RFC3339)
	}
	reason := fmt.Sprintf("the run gave up on the %s outage open since %s after %d probe(s), at %s, and has stopped; done: %s; left: %s",
		strings.Join(in.Services, " and "), in.Since.UTC().Format(time.RFC3339), in.Probes, gave,
		strings.Join(did, "; "), strings.Join(todo, "; "))
	remedy := "once the connection is back, close the outage with `abcd implement outage clear --session <id> --reason <why>`, then run `abcd implement step` again; the run resumes where it stopped"
	if in.Notify {
		reason += "; the product thinker is to be told (notification pending)"
		remedy = "tell the product thinker, then acknowledge it with `abcd implement outage ack --session <id>`; " + remedy
	}
	r := refuse(StageOutage, "", "", reason, remedy)
	r.Outage = in
	return r
}

// CurrentOutage is the shared run's outage in force for the checkout, nil when
// there is none or the checkout has no shared run state. It writes nothing.
func CurrentOutage(repoRoot string) (*implement.Outage, error) {
	sha := gitutil.RootCommit(repoRoot)
	if !gitutil.IsFullSHA(sha) {
		return nil, nil
	}
	run, err := implement.Peek(sha)
	if err != nil {
		return nil, err
	}
	return run.Outage().Current()
}

// runOutages are the outages the shared run's log records over a run's
// lifetime: every one that was open at any time after the run's creation and,
// for a complete run, before its last update.
func runOutages(repoRoot string, st State) ([]implement.OutageSpan, error) {
	out := []implement.OutageSpan{}
	sha := gitutil.RootCommit(repoRoot)
	if !gitutil.IsFullSHA(sha) {
		return out, nil
	}
	run, err := implement.Peek(sha)
	if err != nil {
		return nil, err
	}
	events, bad, err := run.ReadLog()
	if err != nil {
		return nil, err
	}
	complete := st.Complete()
	for _, s := range implement.Compare(events, bad).Outages {
		if (complete && s.Start.After(st.UpdatedAt)) || (s.End != nil && s.End.Before(st.CreatedAt)) {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}
