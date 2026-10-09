package implement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// A lost connection is waited out by the run as a whole, not by each lane on
// its own (iss-2610080620372731). A lane that meets one records it here; the
// run keeps one outage record beside its claims and sessions, outage.json, and
// every lane that reaches a step needing the connection waits on one shared
// probe instead of retrying alone. The probe runs on a schedule — a minute,
// five, ten, then hourly for eight hours — and the first probe that proves
// every service back ends the outage; the probe that fails at or after the
// eighth hour gives up, stops the run and raises one notification for the
// product thinker, held until a session acknowledges it.
//
// Two services can be down, and they are one outage: the network (git, gh, a
// download) and the model service the host's agents call. They share the
// record, the schedule and the probe's lease, and differ only in what proves
// them back: the network by a call to the repository's remote, the model by a
// canary agent the lead runs and reports, since abcd calls no model itself.
// What noticed the outage (the host's own turn, a sub-agent, a tool) is the
// report's kind.
//
// The record lives in the machine-scoped run, so two sessions in two worktrees
// share it, and every change to it is made under the run's lock and written
// whole (fsutil.WriteFileAtomicInRoot), so a reader never sees half of one.
// The probe itself runs outside the lock: the lock is held for file operations
// only, and a probe takes seconds. A lease in the record marks who is probing,
// so a second caller is told to wait rather than probing too, and a holder that
// dies mid-probe frees the probe when its lease lapses.

// The services an outage can take down.
const (
	ServiceNetwork = "network"
	ServiceModel   = "model"
)

// OutageServices returns the closed service vocabulary.
func OutageServices() []string { return []string{ServiceModel, ServiceNetwork} }

// The kinds of report: what noticed the lost connection.
const (
	// OutageHost is the host's own model calls stalling.
	OutageHost = "host"
	// OutageAgent is a sub-agent coming back failed mid-task.
	OutageAgent = "agent"
	// OutageTool is a tool's network call failing: git push, gh, a download.
	OutageTool = "tool"
)

// OutageKinds returns the closed kind vocabulary.
func OutageKinds() []string { return []string{OutageAgent, OutageHost, OutageTool} }

// An outage's status. An outage that ends is removed, so the record only ever
// holds one of these two.
const (
	OutageOpen   = "open"
	OutageGaveUp = "gave_up"
)

// The outage events, written by the outage verbs alone.
const (
	EventOutageStart  = "outage_start"
	EventOutageProbe  = "outage_probe"
	EventOutageEnd    = "outage_end"
	EventOutageGiveUp = "outage_give_up"
)

// ProbeLease is how long a probe's holder keeps it: longer than the network
// probe's own timeout, so a live probe never lapses, and short enough that a
// holder that died mid-probe holds the other lanes for minutes, not hours.
const ProbeLease = 2 * time.Minute

// NetworkProbeTimeout bounds the network probe's call to the remote.
const NetworkProbeTimeout = 20 * time.Second

// outageFileName is the record's name inside the run directory.
const outageFileName = "outage.json"

// Bounds on the record, so a lane reporting in a loop cannot grow it past what
// a reader holds.
const (
	maxOutageReports = 256
	maxOutageBytes   = 1 << 20
)

// Schedule is when the shared probe runs, as data: the waits after the outage
// opens and after each of the first failed probes, then a probe every Hourly,
// for HourlyFor after the hourly stage begins.
type Schedule struct {
	Waits     []time.Duration `json:"waits"`
	Hourly    time.Duration   `json:"hourly"`
	HourlyFor time.Duration   `json:"hourly_for"`
}

// DefaultSchedule is the product thinker's (interview of 2026-10-09): one
// minute, five, ten, then hourly for up to eight hours.
var DefaultSchedule = Schedule{
	Waits:     []time.Duration{time.Minute, 5 * time.Minute, 10 * time.Minute},
	Hourly:    time.Hour,
	HourlyFor: 8 * time.Hour,
}

// Next is when the next probe falls due: the wait for failures failed probes
// so far, counted from last — the outage's start when nothing has been probed,
// and otherwise the probe that actually ran, so a late probe moves every one
// after it rather than bunching them up.
func (s Schedule) Next(last time.Time, failures int) time.Time {
	if failures < 0 {
		failures = 0
	}
	if failures < len(s.Waits) {
		return last.Add(s.Waits[failures])
	}
	return last.Add(s.Hourly)
}

// GivesUp reports whether a probe failing at at gives up, for an hourly stage
// that began at hourlySince: at or after HourlyFor into it.
func (s Schedule) GivesUp(hourlySince, at time.Time) bool {
	return !at.Before(hourlySince.Add(s.HourlyFor))
}

// OutageReport is one lane's report of a lost connection.
type OutageReport struct {
	At      time.Time `json:"at"`
	Session string    `json:"session"`
	Service string    `json:"service"`
	Kind    string    `json:"kind"`
	Lane    string    `json:"lane"`
	What    string    `json:"what"`
}

// ServiceResult is one service's answer to a probe. A service the probe had no
// way to prove (no canary verdict for the model) is not OK, and says it is
// unproven.
type ServiceResult struct {
	Service string `json:"service"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail,omitempty"`
}

// OutageProbe is one run of the shared probe. OK is true when it left no
// service down.
type OutageProbe struct {
	At      time.Time       `json:"at"`
	Session string          `json:"session"`
	OK      bool            `json:"ok"`
	Results []ServiceResult `json:"results"`
}

// ProbeLeaseHolder is the session running the probe, and until when it holds it.
type ProbeLeaseHolder struct {
	Holder string    `json:"holder"`
	Until  time.Time `json:"until"`
}

// Outage is the run's outage record.
type Outage struct {
	StartedAt time.Time `json:"started_at"`
	// Services are every service the outage has taken down; Down are those not
	// yet proven back. The outage ends when Down is empty.
	Services []string `json:"services"`
	Down     []string `json:"down"`
	// Kinds are what noticed it, over every report.
	Kinds   []string       `json:"kinds"`
	Reports []OutageReport `json:"reports"`
	// ReportsDropped counts the reports past maxOutageReports, kept as a count.
	ReportsDropped int           `json:"reports_dropped,omitempty"`
	Probes         []OutageProbe `json:"probes"`
	NextProbeAt    time.Time     `json:"next_probe_at"`
	// HourlySince is the third failed probe, where the hourly stage begins.
	HourlySince *time.Time        `json:"hourly_since,omitempty"`
	Lease       *ProbeLeaseHolder `json:"lease,omitempty"`
	Status      string            `json:"status"`
	GaveUpAt    *time.Time        `json:"gave_up_at,omitempty"`
	// NotifyPending is raised once, at the give-up, and lowered by Ack.
	NotifyPending bool `json:"notify_pending"`
}

// Failures counts the probes that left a service down.
func (o Outage) Failures() int {
	n := 0
	for _, p := range o.Probes {
		if !p.OK {
			n++
		}
	}
	return n
}

// Retried is what the lanes were trying to do when they lost the connection,
// each once, in the order first reported.
func (o Outage) Retried() []string {
	out := []string{}
	for _, r := range o.Reports {
		if !slices.Contains(out, r.What) {
			out = append(out, r.What)
		}
	}
	return out
}

// Prober is how each service is proven back. A nil function leaves its service
// unproven, and still down when it was down. Only the services down are
// probed. Model is the lead's canary verdict: abcd calls no model, and the
// lead's own turn running is not proof the service is back.
type Prober struct {
	Network func() (ok bool, detail string)
	Model   func() (ok bool, detail string)
}

// ProbeOutcome is what ProbeIfDue did.
type ProbeOutcome struct {
	// Probed is false when there was no outage to probe.
	Probed bool         `json:"probed"`
	Probe  *OutageProbe `json:"probe,omitempty"`
	// Ended is true when the probe proved every service back; Minutes is then
	// how long the outage lasted.
	Ended   bool    `json:"ended"`
	Minutes float64 `json:"minutes,omitempty"`
	// GaveUp is true when this probe gave up.
	GaveUp bool `json:"gave_up"`
	// Outage is the record after the probe, nil once it has ended.
	Outage *Outage `json:"outage"`
}

// OutageEnd is how an outage ended.
type OutageEnd struct {
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	How       string    `json:"how"`
	Minutes   float64   `json:"minutes"`
	Services  []string  `json:"services"`
	Kinds     []string  `json:"kinds"`
	Retried   []string  `json:"retried"`
	Probes    int       `json:"probes"`
}

// OutageWaitError is the contention a caller meets when the probe is not due
// or another session holds it: it names when the next probe falls due and, for
// a held probe, the holder and its lease.
type OutageWaitError struct {
	NextProbeAt time.Time `json:"next_probe_at"`
	Down        []string  `json:"down"`
	Holder      string    `json:"holder,omitempty"`
	LeaseUntil  time.Time `json:"lease_until,omitzero"`
}

func (e *OutageWaitError) Error() string {
	down := strings.Join(e.Down, " and ")
	if e.Holder != "" {
		return fmt.Sprintf("%s: session %s is probing the %s outage (its lease holds until %s; next_probe_at %s); wait on its result and keep to offline work",
			ErrContention, e.Holder, down, e.LeaseUntil.Format(time.RFC3339), e.NextProbeAt.Format(time.RFC3339))
	}
	return fmt.Sprintf("%s: the %s outage's next probe is not due until %s (next_probe_at); keep to offline work until then",
		ErrContention, down, e.NextProbeAt.Format(time.RFC3339))
}

// Is makes an OutageWaitError an ErrContention.
func (e *OutageWaitError) Is(target error) bool { return target == ErrContention }

// Outages is the run's outage handle.
type Outages struct {
	run *Run
	// Schedule is the probe schedule; the zero value is DefaultSchedule.
	Schedule Schedule
}

// Outage returns the run's outage handle, on DefaultSchedule.
func (r *Run) Outage() *Outages { return &Outages{run: r, Schedule: DefaultSchedule} }

func (o *Outages) schedule() Schedule {
	if len(o.Schedule.Waits) == 0 && o.Schedule.Hourly == 0 {
		return DefaultSchedule
	}
	return o.Schedule
}

// read returns the record, nil when there is none.
func (o *Outages) read() (*Outage, error) {
	if !o.run.exists {
		return nil, nil
	}
	root, err := o.run.root()
	if err != nil {
		return nil, err
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, outageFileName, maxOutageBytes)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read the outage record: %w", err)
	}
	var out Outage
	if err := json.Unmarshal(data, &out); err != nil || (out.Status != OutageOpen && out.Status != OutageGaveUp) {
		return nil, fmt.Errorf("the outage record %s is unreadable; remove it by hand once the connection is back", outageFileName)
	}
	return &out, nil
}

// write replaces the record whole. The caller holds the lock.
func (o *Outages) write(cur *Outage) error {
	data, err := json.MarshalIndent(cur, "", "  ")
	if err != nil {
		return err
	}
	root, err := o.run.root()
	if err != nil {
		return err
	}
	defer root.Close()
	if err := fsutil.WriteFileAtomicInRoot(root, outageFileName, append(data, '\n'), fileMode); err != nil {
		return fmt.Errorf("cannot write the outage record: %w", err)
	}
	return nil
}

// remove deletes the record. The caller holds the lock.
func (o *Outages) remove() error {
	root, err := o.run.root()
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.Remove(outageFileName); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("cannot remove the outage record: %w", err)
	}
	return nil
}

// Current returns the outage in force, nil when there is none. It writes
// nothing and takes no lock: the record is only ever replaced whole.
func (o *Outages) Current() (*Outage, error) { return o.read() }

// Record opens an outage, or extends the one open, with a lane's report: the
// service it lost, what noticed it, the lane and what it was doing. The first
// report logs outage_start and sets the first probe a schedule's wait away; a
// later one adds its service to those down (a service proven back and lost
// again is down again) and its kind to the kinds. A report to an outage that
// gave up joins it and raises no second notification.
func (o *Outages) Record(session, service, kind, lane, what string) (Outage, error) {
	if !slices.Contains(OutageServices(), service) {
		return Outage{}, refusal("unknown service %q (one of: %s)", service, strings.Join(OutageServices(), ", "))
	}
	if !slices.Contains(OutageKinds(), kind) {
		return Outage{}, refusal("unknown kind %q (one of: %s)", kind, strings.Join(OutageKinds(), ", "))
	}
	if err := validName("lane", lane); err != nil {
		return Outage{}, err
	}
	what = strings.TrimSpace(what)
	if what == "" {
		return Outage{}, refusal("an outage report names what the lane was doing (--what)")
	}
	if len(what) > maxValueBytes {
		return Outage{}, refusal("--what is %d bytes, over %d", len(what), maxValueBytes)
	}
	var out Outage
	err := o.run.withLock(session, func() error {
		if _, err := o.run.requireSession(session); err != nil {
			return err
		}
		cur, err := o.read()
		if err != nil {
			return err
		}
		now := o.run.now()
		opening := cur == nil
		if opening {
			cur = &Outage{StartedAt: now, Status: OutageOpen, NextProbeAt: o.schedule().Next(now, 0),
				Services: []string{}, Down: []string{}, Kinds: []string{}, Reports: []OutageReport{}, Probes: []OutageProbe{}}
		}
		if len(cur.Reports) < maxOutageReports {
			cur.Reports = append(cur.Reports, OutageReport{At: now, Session: session, Service: service, Kind: kind, Lane: lane, What: what})
		} else {
			cur.ReportsDropped++
		}
		cur.Services = addSorted(cur.Services, service)
		cur.Down = addSorted(cur.Down, service)
		cur.Kinds = addSorted(cur.Kinds, kind)
		if opening {
			fields := map[string]any{"service": service, "kind": kind, "lane": lane, "what": what,
				"next_probe_at": cur.NextProbeAt.Format(time.RFC3339)}
			if err := checkOutageFields(EventOutageStart, fields); err != nil {
				return err
			}
			if err := o.write(cur); err != nil {
				return err
			}
			if _, err := o.run.append(session, EventOutageStart, fields); err != nil {
				// The log never misses an outage the record holds.
				_ = o.remove()
				return err
			}
		} else if err := o.write(cur); err != nil {
			return err
		}
		out = *cur
		return nil
	})
	return out, err
}

// ProbeIfDue runs the shared probe if it is due and nobody else is running it.
// Under the run's lock it reads the outage: none is nothing to wait on; one
// that gave up is refused, because the run has stopped; one whose next probe
// is not due, or whose probe another live lease holds, is an *OutageWaitError
// (contention) carrying next_probe_at. Otherwise it takes the lease, releases
// the lock, probes each service down (prober's function for it; none leaves it
// unproven), then re-locks and writes the result: every service back ends the
// outage and logs outage_end; otherwise the failure is counted, the next probe
// set by the schedule, and a failure at or past the hourly stage's limit gives
// up, raising notify_pending and logging outage_give_up. A result whose lease
// lapsed and was taken by another session meanwhile is discarded.
func (o *Outages) ProbeIfDue(session string, p Prober) (ProbeOutcome, error) {
	var down []string
	var lease ProbeLeaseHolder
	err := o.run.withLock(session, func() error {
		if _, err := o.run.requireSession(session); err != nil {
			return err
		}
		cur, err := o.read()
		if err != nil || cur == nil {
			return err
		}
		if cur.Status == OutageGaveUp {
			return refusal("the run gave up on this outage at %s after %d probes and has stopped; once the connection is back, `abcd implement outage clear --reason` closes it",
				fmtTimePtr(cur.GaveUpAt), len(cur.Probes))
		}
		now := o.run.now()
		if now.Before(cur.NextProbeAt) {
			return &OutageWaitError{NextProbeAt: cur.NextProbeAt, Down: slices.Clone(cur.Down)}
		}
		if cur.Lease != nil && now.Before(cur.Lease.Until) {
			return &OutageWaitError{NextProbeAt: cur.NextProbeAt, Down: slices.Clone(cur.Down),
				Holder: cur.Lease.Holder, LeaseUntil: cur.Lease.Until}
		}
		lease = ProbeLeaseHolder{Holder: session, Until: now.Add(ProbeLease)}
		cur.Lease = &lease
		down = slices.Clone(cur.Down)
		return o.write(cur)
	})
	if err != nil || down == nil {
		return ProbeOutcome{}, err
	}

	results := make([]ServiceResult, 0, len(down))
	for _, svc := range down {
		results = append(results, probeService(svc, p))
	}

	var out ProbeOutcome
	err = o.run.withLock(session, func() error {
		cur, err := o.read()
		if err != nil {
			return err
		}
		now := o.run.now()
		probe := OutageProbe{At: now, Session: session, Results: results}
		out = ProbeOutcome{Probed: true, Probe: &probe}
		if cur == nil {
			// Cleared by hand while the probe ran: nothing left to record on.
			return nil
		}
		if cur.Lease == nil || cur.Lease.Holder != session || !cur.Lease.Until.Equal(lease.Until) {
			e := &OutageWaitError{NextProbeAt: cur.NextProbeAt, Down: slices.Clone(cur.Down)}
			if cur.Lease != nil {
				e.Holder, e.LeaseUntil = cur.Lease.Holder, cur.Lease.Until
			}
			out = ProbeOutcome{}
			return e
		}
		for _, r := range results {
			if r.OK {
				cur.Down = slices.DeleteFunc(cur.Down, func(s string) bool { return s == r.Service })
			}
		}
		probe.OK = len(cur.Down) == 0
		cur.Probes = append(cur.Probes, probe)
		cur.Lease = nil
		failures := cur.Failures()
		probeFields := map[string]any{"ok": probe.OK, "failures": failures, "results": results}
		if !probe.OK {
			probeFields["down"] = cur.Down
		}
		if probe.OK {
			end := o.endOf(cur, now, "probe")
			if err := checkOutageFields(EventOutageProbe, probeFields); err != nil {
				return err
			}
			endFields := end.fields()
			if err := checkOutageFields(EventOutageEnd, endFields); err != nil {
				return err
			}
			if err := o.remove(); err != nil {
				return err
			}
			if _, err := o.run.append(session, EventOutageProbe, probeFields); err != nil {
				return err
			}
			out.Ended, out.Minutes = true, end.Minutes
			_, err := o.run.append(session, EventOutageEnd, endFields)
			return err
		}
		s := o.schedule()
		if cur.HourlySince == nil && failures >= len(s.Waits) {
			at := now
			cur.HourlySince = &at
		}
		var giveUp map[string]any
		if cur.HourlySince != nil && s.GivesUp(*cur.HourlySince, now) {
			at := now
			cur.Status, cur.GaveUpAt, cur.NotifyPending = OutageGaveUp, &at, true
			giveUp = o.endOf(cur, now, "gave_up").fields()
			delete(giveUp, "how")
			giveUp["hourly_since"] = cur.HourlySince.Format(time.RFC3339)
			if err := checkOutageFields(EventOutageGiveUp, giveUp); err != nil {
				return err
			}
			out.GaveUp = true
		} else {
			cur.NextProbeAt = s.Next(now, failures)
			probeFields["next_probe_at"] = cur.NextProbeAt.Format(time.RFC3339)
		}
		if err := checkOutageFields(EventOutageProbe, probeFields); err != nil {
			return err
		}
		if err := o.write(cur); err != nil {
			return err
		}
		snapshot := *cur
		out.Outage = &snapshot
		if _, err := o.run.append(session, EventOutageProbe, probeFields); err != nil {
			return err
		}
		if giveUp != nil {
			_, err := o.run.append(session, EventOutageGiveUp, giveUp)
			return err
		}
		return nil
	})
	return out, err
}

// probeService proves one service back, or says why it could not.
func probeService(service string, p Prober) ServiceResult {
	var fn func() (bool, string)
	unproven := ""
	switch service {
	case ServiceNetwork:
		fn, unproven = p.Network, "unproven: no network probe was given"
	case ServiceModel:
		fn, unproven = p.Model, "unproven: no canary verdict was given (the lead's own turn running is not proof the model service is back)"
	}
	if fn == nil {
		return ServiceResult{Service: service, Detail: unproven}
	}
	ok, detail := fn()
	if len(detail) > maxValueBytes {
		detail = detail[:maxValueBytes]
	}
	return ServiceResult{Service: service, OK: ok, Detail: detail}
}

// endOf is the end of cur at now.
func (o *Outages) endOf(cur *Outage, now time.Time, how string) OutageEnd {
	return OutageEnd{StartedAt: cur.StartedAt, EndedAt: now, How: how,
		Minutes:  round2(now.Sub(cur.StartedAt).Minutes()),
		Services: slices.Clone(cur.Services), Kinds: slices.Clone(cur.Kinds), Retried: cur.Retried(), Probes: len(cur.Probes)}
}

// fields are the outage_end (and outage_give_up) line's fields.
func (e OutageEnd) fields() map[string]any {
	return map[string]any{"how": e.How, "minutes": e.Minutes, "services": e.Services, "kinds": e.Kinds,
		"retried": e.Retried, "started_at": e.StartedAt.Format(time.RFC3339), "probes": e.Probes}
}

// Ack lowers the give-up's notification: the product thinker has been told.
// It is refused when nothing is pending, so a notification is never raised
// again by acknowledging it twice.
func (o *Outages) Ack(session string) (Outage, error) {
	var out Outage
	err := o.run.withLock(session, func() error {
		if _, err := o.run.requireSession(session); err != nil {
			return err
		}
		cur, err := o.read()
		if err != nil {
			return err
		}
		if cur == nil || !cur.NotifyPending {
			return refusal("no outage notification is pending; nothing to acknowledge")
		}
		cur.NotifyPending = false
		if err := o.write(cur); err != nil {
			return err
		}
		out = *cur
		return nil
	})
	return out, err
}

// Clear closes the outage by hand, open or given up, with the reason. It logs
// outage_end (how "cleared") and, because a person or a session stepped in
// where the shared probe should have, an intervention naming the gap.
func (o *Outages) Clear(session, reason string) (OutageEnd, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return OutageEnd{}, refusal("a clear names its reason (--reason)")
	}
	if len(reason) > maxValueBytes {
		return OutageEnd{}, refusal("--reason is %d bytes, over %d", len(reason), maxValueBytes)
	}
	var out OutageEnd
	err := o.run.withLock(session, func() error {
		if _, err := o.run.requireSession(session); err != nil {
			return err
		}
		cur, err := o.read()
		if err != nil {
			return err
		}
		if cur == nil {
			return refusal("there is no outage to clear")
		}
		now := o.run.now()
		out = o.endOf(cur, now, "cleared")
		fields := out.fields()
		fields["reason"] = reason
		if cur.Status == OutageGaveUp {
			fields["after_give_up"] = true
		}
		if err := checkOutageFields(EventOutageEnd, fields); err != nil {
			return err
		}
		if err := o.remove(); err != nil {
			return err
		}
		if _, err := o.run.append(session, EventOutageEnd, fields); err != nil {
			return err
		}
		_, err = o.run.append(session, EventIntervention, map[string]any{
			"kind": "other", "by": session, "what": "cleared the run's outage by hand", "why": reason,
			"autonomy_gap": "the shared probe did not prove the connection back on its own",
		})
		return err
	})
	return out, err
}

// addSorted adds v to a sorted set.
func addSorted(set []string, v string) []string {
	if slices.Contains(set, v) {
		return set
	}
	set = append(set, v)
	slices.Sort(set)
	return set
}

func fmtTimePtr(t *time.Time) string {
	if t == nil {
		return "an unrecorded time"
	}
	return t.Format(time.RFC3339)
}

// checkOutageFields holds an outage line the verbs write to the fields the
// report reads, the way backoffFields holds a hand-logged backoff: each value
// is read as the text a line carries (a list as its items joined), checked
// against the event's rules in eventFields, and the closed vocabularies
// checked by name.
func checkOutageFields(event string, fields map[string]any) error {
	text := make(map[string]string, len(fields))
	for k, v := range fields {
		text[k] = outageFieldText(v)
	}
	if err := checkFields(event, text); err != nil {
		return err
	}
	if v, ok := fields["service"]; ok && !slices.Contains(OutageServices(), text["service"]) {
		return refusal("%s field service is %v (one of: %s)", event, v, strings.Join(OutageServices(), ", "))
	}
	if v, ok := fields["kind"]; ok && !slices.Contains(OutageKinds(), text["kind"]) {
		return refusal("%s field kind is %v (one of: %s)", event, v, strings.Join(OutageKinds(), ", "))
	}
	if ss, ok := fields["services"].([]string); ok {
		for _, s := range ss {
			if !slices.Contains(OutageServices(), s) {
				return refusal("%s field services holds %q (one of: %s)", event, s, strings.Join(OutageServices(), ", "))
			}
		}
	}
	if _, ok := fields["ok"]; ok {
		if _, isBool := fields["ok"].(bool); !isBool {
			return refusal("%s field ok is %v, not true or false", event, fields["ok"])
		}
	}
	return nil
}

// outageFieldText is a field's value as checkFields reads it.
func outageFieldText(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case int:
		return strconv.Itoa(x)
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return "NaN"
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case []string:
		return strings.Join(x, ",")
	case time.Time:
		return x.Format(time.RFC3339)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// IsNetworkFailure reports whether a git or gh command's stderr says the
// network failed it: a name that did not resolve, a connection refused, timed
// out, reset or cut off. A refusal by the other end — an authentication
// failure, a hook's refusal, a rejected push — or a merge conflict is not one,
// even when its text also quotes a network-sounding line, because waiting does
// not fix it.
func IsNetworkFailure(stderr string) bool {
	s := strings.ToLower(stderr)
	for _, m := range notNetworkMarkers {
		if strings.Contains(s, m) {
			return false
		}
	}
	for _, m := range networkMarkers {
		if strings.Contains(s, m) {
			return true
		}
	}
	return false
}

// networkMarkers are the lower-cased phrases git, curl, ssh and gh print when
// the network fails them.
var networkMarkers = []string{
	"could not resolve host", // curl's and ssh's (hostname)
	"temporary failure in name resolution",
	"name or service not known",
	"nodename nor servname",
	"no such host",
	"connection refused",
	"connection timed out",
	"operation timed out",
	"connection reset by peer",
	"network is unreachable",
	"no route to host",
	"dial tcp",
	"i/o timeout",
	"tls handshake timeout",
	"unexpected disconnect",
	"failed to connect",
	"couldn't connect to server",
	"error connecting to", // gh: error connecting to api.github.com
}

// notNetworkMarkers are the other end's refusals and the local conflicts.
var notNetworkMarkers = []string{
	"authentication failed",
	"permission denied",
	"could not read username",
	"could not read password",
	"invalid username or token",
	"bad credentials",
	"http 401",
	"http 403",
	"resource not accessible",
	"hook declined",
	"pre-receive",
	"pre-push",
	"[rejected]",
	"[remote rejected]",
	"protected branch",
	"conflict",
}

// IsModelFailure reports whether an agent's output says the model service
// failed it: an API error, the service overloaded (529), a server error, a
// request that timed out or lost its connection. A usage or rate limit (429)
// is not an outage — the service answered, and waiting on a probe is not how
// a limit is met — so it reads false.
func IsModelFailure(output string) bool {
	s := strings.ToLower(output)
	for _, m := range []string{"usage limit", "rate limit", "rate_limit"} {
		if strings.Contains(s, m) {
			return false
		}
	}
	if status429Re.MatchString(s) {
		return false
	}
	for _, m := range modelMarkers {
		if strings.Contains(s, m) {
			return true
		}
	}
	return status529Re.MatchString(s)
}

// modelMarkers are the lower-cased phrases a failed model call prints.
var modelMarkers = []string{
	"api error",
	"overloaded",
	"internal server error",
	"bad gateway",
	"service unavailable",
	"gateway timeout",
	"request timed out",
	"connection error",
}

var (
	status429Re = regexp.MustCompile(`\b429\b`)
	status529Re = regexp.MustCompile(`\b529\b`)
)

// RemoteProbe is the production network probe: `git ls-remote --exit-code
// origin HEAD` in repoRoot through the isolated git helper, bounded by
// NetworkProbeTimeout. The connection is back when the remote answers — with
// HEAD (exit 0), without it (exit 2), or with a refusal that is not a network
// failure, since a remote that refuses has been reached. It is down when git
// prints a network failure or gives no answer in time. Credentials are never
// asked for at a terminal: the probe's askpass fails at once.
func RemoteProbe(repoRoot string) func() (bool, string) {
	return func() (bool, string) {
		ctx, cancel := context.WithTimeout(context.Background(), NetworkProbeTimeout)
		defer cancel()
		_, err := gitutil.RunLimitedContext(ctx, repoRoot, 4096,
			"-c", "core.askPass=false", "ls-remote", "--exit-code", "origin", "HEAD")
		if err == nil {
			return true, "origin answered"
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return false, fmt.Sprintf("origin gave no answer within %s", NetworkProbeTimeout)
		}
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 2 {
			return true, "origin answered (no HEAD)"
		}
		msg := err.Error()
		if IsNetworkFailure(msg) {
			return false, msg
		}
		return true, "origin was reached but refused: " + msg
	}
}
