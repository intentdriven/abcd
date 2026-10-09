package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// failing is a probe that always reports the connection still down, and netUp
// one that reports it back.
func failing() (bool, string) { return false, "Could not resolve host: github.com" }
func netUp() (bool, string)   { return true, "origin answered" }

// netDown probes the network and finds it down; it gives no model verdict.
var netDown = Prober{Network: failing}

// recordOutage records an outage report or fails the test.
func recordOutage(t *testing.T, r *Run, session, service, kind, lane, what string) Outage {
	t.Helper()
	o, err := r.Outage().Record(session, service, kind, lane, what)
	if err != nil {
		t.Fatalf("Record(%s, %s, %s, %s, %s): %v", session, service, kind, lane, what, err)
	}
	return o
}

// takeLeaseForTest takes the probe lease for session and never releases it: a
// holder that died mid-probe.
func (o *Outages) takeLeaseForTest(session string) error {
	return o.run.withLock(session, func() error {
		cur, err := o.read()
		if err != nil || cur == nil {
			return fmt.Errorf("no outage to lease: %v", err)
		}
		now := o.run.now()
		cur.Lease = &ProbeLeaseHolder{Holder: session, Until: now.Add(ProbeLease)}
		return o.write(cur)
	})
}

// TestOutageScheduleOneFiveTenThenHourlyEightHours: the schedule as data, and the
// same schedule driven through the shared probe. Each wait is counted from the
// probe that actually ran; the hourly stage begins at the third failed probe;
// the first failed probe at or after eight hours into that stage gives up.
func TestOutageScheduleOneFiveTenThenHourlyEightHours(t *testing.T) {
	s := DefaultSchedule
	t0 := time.Date(2026, 10, 9, 23, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		last     time.Duration
		failures int
		want     time.Duration
	}{
		{0, 0, time.Minute},                      // the outage opens: first probe a minute on
		{time.Minute, 1, 6 * time.Minute},        // one failure: five minutes after it
		{6 * time.Minute, 2, 16 * time.Minute},   // two: ten minutes after
		{16 * time.Minute, 3, 76 * time.Minute},  // three: hourly from here
		{80 * time.Minute, 4, 140 * time.Minute}, // a late probe: counted from when it ran
		{7 * time.Hour, 10, 8 * time.Hour},       // still hourly
		{3 * time.Minute, 1, 8 * time.Minute},    // a late first probe moves the second
		{2 * time.Minute, 2, 12 * time.Minute},   // and the third
		{100 * time.Hour, 50, 101 * time.Hour},   // any count past the waits is hourly
		{20 * time.Second, 0, 80 * time.Second},  // the opening wait is from the start
		{16 * time.Minute, 3, 16*time.Minute + s.Hourly},
	} {
		if got := s.Next(t0.Add(c.last), c.failures); !got.Equal(t0.Add(c.want)) {
			t.Errorf("Next(+%s, %d failures) = +%s, want +%s", c.last, c.failures, got.Sub(t0), c.want)
		}
	}
	h := t0.Add(16 * time.Minute)
	if s.GivesUp(h, h.Add(8*time.Hour-time.Second)) {
		t.Error("gave up a second before eight hours of the hourly stage")
	}
	if !s.GivesUp(h, h.Add(8*time.Hour)) || !s.GivesUp(h, h.Add(9*time.Hour)) {
		t.Error("did not give up at or after eight hours of the hourly stage")
	}

	// Driven through the shared probe, every probe run the moment it is due.
	r, c := newRun(t)
	c.t = t0
	join(t, r, "alpha", RoleFirst)
	recordOutage(t, r, "alpha", ServiceNetwork, "tool", "lane-1", "git push")
	var at []time.Duration
	var outcome ProbeOutcome
	for i := 0; i < 20; i++ {
		cur, err := r.Outage().Current()
		if err != nil || cur == nil {
			t.Fatalf("Current = %v, %v", cur, err)
		}
		if cur.Status == OutageGaveUp {
			break
		}
		// One second early is not due, and says when it will be.
		c.t = cur.NextProbeAt.Add(-time.Second)
		var wait *OutageWaitError
		if _, err := r.Outage().ProbeIfDue("alpha", netDown); !errors.As(err, &wait) || !errors.Is(err, ErrContention) ||
			!wait.NextProbeAt.Equal(cur.NextProbeAt) {
			t.Fatalf("a probe a second early = %v; want contention carrying next_probe_at %s", err, cur.NextProbeAt)
		}
		c.t = cur.NextProbeAt
		if outcome, err = r.Outage().ProbeIfDue("alpha", netDown); err != nil || !outcome.Probed {
			t.Fatalf("probe %d = %+v, %v", i+1, outcome, err)
		}
		at = append(at, c.t.Sub(t0))
	}
	want := []time.Duration{time.Minute, 6 * time.Minute, 16 * time.Minute}
	for h := 1; h <= 8; h++ {
		want = append(want, 16*time.Minute+time.Duration(h)*time.Hour)
	}
	if !reflect.DeepEqual(at, want) {
		t.Fatalf("probes ran at %v\nwant %v", at, want)
	}
	if !outcome.GaveUp {
		t.Fatalf("the last probe = %+v; want the give-up", outcome)
	}
	cur, _ := r.Outage().Current()
	if cur.HourlySince == nil || !cur.HourlySince.Equal(t0.Add(16*time.Minute)) {
		t.Fatalf("hourly_since = %v, want the third failed probe at +16m", cur.HourlySince)
	}
	if cur.Status != OutageGaveUp || !cur.NotifyPending || len(cur.Probes) != 11 {
		t.Fatalf("after the give-up: status %s, notify %v, %d probes", cur.Status, cur.NotifyPending, len(cur.Probes))
	}
}

// TestOutageProbeHasOneHolder: two Run handles on one run (two sessions in two
// worktrees). While the first session's probe is running, the second is told
// who holds it and when the next probe falls due, and its prober never runs.
func TestOutageProbeHasOneHolder(t *testing.T) {
	r1, c := newRun(t)
	join(t, r1, "alpha", RoleFirst)
	r2, err := Open(testSHA)
	if err != nil {
		t.Fatal(err)
	}
	r2.Now = c.now
	join(t, r2, "beta", RoleSecond)
	recordOutage(t, r1, "alpha", ServiceNetwork, "tool", "lane-1", "git push")
	recordOutage(t, r2, "beta", ServiceNetwork, "tool", "lane-2", "gh pr view")
	c.advance(time.Minute)

	var runs atomic.Int32
	started, finish := make(chan struct{}), make(chan struct{})
	slow := func() (bool, string) {
		runs.Add(1)
		close(started)
		<-finish
		return false, "dial tcp: i/o timeout"
	}
	var wg sync.WaitGroup
	var first ProbeOutcome
	var firstErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		first, firstErr = r1.Outage().ProbeIfDue("alpha", Prober{Network: slow})
	}()
	<-started
	second := func() (bool, string) { runs.Add(1); return true, "" }
	_, err = r2.Outage().ProbeIfDue("beta", Prober{Network: second})
	close(finish)
	wg.Wait()

	var wait *OutageWaitError
	if !errors.As(err, &wait) || !errors.Is(err, ErrContention) {
		t.Fatalf("the second caller = %v; want contention", err)
	}
	if wait.Holder != "alpha" || wait.NextProbeAt.IsZero() {
		t.Fatalf("contention = %+v; want the holder named and next_probe_at", wait)
	}
	if !strings.Contains(err.Error(), "alpha") || !strings.Contains(err.Error(), wait.NextProbeAt.Format(time.RFC3339)) {
		t.Fatalf("contention message %q names neither the holder nor next_probe_at", err)
	}
	if firstErr != nil || !first.Probed || first.Probe == nil || first.Probe.OK {
		t.Fatalf("the holder's probe = %+v, %v", first, firstErr)
	}
	if n := runs.Load(); n != 1 {
		t.Fatalf("the prober ran %d times; want once", n)
	}
	cur, err := r2.Outage().Current()
	if err != nil || cur == nil || cur.Lease != nil || len(cur.Probes) != 1 {
		t.Fatalf("after the probe = %+v, %v; want one probe recorded and the lease released", cur, err)
	}
	if !cur.NextProbeAt.Equal(c.now().Add(5 * time.Minute)) {
		t.Fatalf("next_probe_at = %s; want five minutes after the failed probe", cur.NextProbeAt)
	}
	if probe := lastEvent(t, r1, EventOutageProbe); probe.Session != "alpha" || probe.String("ok") != "false" {
		t.Fatalf("outage_probe line = %+v", probe.Fields)
	}

	// A lease whose holder died lapses: the next caller probes.
	c.advance(5 * time.Minute)
	crashed := r1.Outage()
	if err := crashed.takeLeaseForTest("alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := r2.Outage().ProbeIfDue("beta", Prober{Network: second}); !errors.Is(err, ErrContention) {
		t.Fatalf("a live lease = %v; want contention", err)
	}
	c.advance(ProbeLease)
	if out, err := r2.Outage().ProbeIfDue("beta", Prober{Network: second}); err != nil || !out.Ended {
		t.Fatalf("after the lease lapsed = %+v, %v; want the probe run and the outage ended", out, err)
	}
}

// TestOutageEndLogsMinutesAndRetried: the outage opens on the first report and
// logs outage_start; a report of another kind extends it; a successful probe
// ends it with the minutes it lasted, the kinds and what was retried, and the
// record is gone.
func TestOutageEndLogsMinutesAndRetried(t *testing.T) {
	r, c := newRun(t)
	join(t, r, "alpha", RoleFirst)
	join(t, r, "beta", RoleSecond)
	if cur, err := r.Outage().Current(); err != nil || cur != nil {
		t.Fatalf("Current before any report = %+v, %v", cur, err)
	}
	if out, err := r.Outage().ProbeIfDue("alpha", netDown); err != nil || out.Probed || out.Outage != nil {
		t.Fatalf("a probe with no outage = %+v, %v; want nothing to wait on", out, err)
	}
	t0 := c.now()
	o := recordOutage(t, r, "alpha", ServiceNetwork, "tool", "lane-1", "git push")
	if !o.StartedAt.Equal(t0) || o.Status != OutageOpen || !o.NextProbeAt.Equal(t0.Add(time.Minute)) {
		t.Fatalf("opened outage = %+v", o)
	}
	start := lastEvent(t, r, EventOutageStart)
	if start.String("service") != ServiceNetwork || start.String("kind") != "tool" || start.String("lane") != "lane-1" || start.String("what") != "git push" {
		t.Fatalf("outage_start = %+v", start.Fields)
	}
	c.advance(30 * time.Second)
	o = recordOutage(t, r, "beta", ServiceNetwork, "host", "lane-2", "model call")
	recordOutage(t, r, "beta", ServiceNetwork, "host", "lane-2", "model call") // restated: retried once
	if !reflect.DeepEqual(o.Kinds, []string{"host", "tool"}) || len(o.Reports) != 2 || !o.StartedAt.Equal(t0) {
		t.Fatalf("extended outage = %+v", o)
	}
	if n := countEvents(t, r, EventOutageStart); n != 1 {
		t.Fatalf("%d outage_start lines; an extension opens no second outage", n)
	}
	for _, bad := range [][3]string{{"wifi", "lane-1", "x"}, {"tool", "../x", "x"}, {"tool", "lane-1", " "}} {
		if _, err := r.Outage().Record("alpha", ServiceNetwork, bad[0], bad[1], bad[2]); !errors.Is(err, ErrRefused) {
			t.Errorf("Record(%q) = %v; want a refusal", bad, err)
		}
	}
	if _, err := r.Outage().Record("ghost", ServiceNetwork, "tool", "lane-1", "x"); !errors.Is(err, ErrRefused) {
		t.Errorf("a session that has not joined = %v; want a refusal", err)
	}

	c.t = t0.Add(time.Minute)
	if _, err := r.Outage().ProbeIfDue("alpha", netDown); err != nil {
		t.Fatal(err)
	}
	c.t = t0.Add(6 * time.Minute)
	out, err := r.Outage().ProbeIfDue("beta", Prober{Network: netUp})
	if err != nil || !out.Ended || out.Minutes != 6 || out.Outage != nil {
		t.Fatalf("the successful probe = %+v, %v", out, err)
	}
	end := lastEvent(t, r, EventOutageEnd)
	if m, ok := end.Number("minutes"); !ok || m != 6 {
		t.Fatalf("outage_end minutes = %v (%v); want 6", m, ok)
	}
	var kinds, retried []string
	_ = json.Unmarshal(end.Fields["kinds"], &kinds)
	_ = json.Unmarshal(end.Fields["retried"], &retried)
	if !reflect.DeepEqual(kinds, []string{"host", "tool"}) || !reflect.DeepEqual(retried, []string{"git push", "model call"}) {
		t.Fatalf("outage_end kinds %v retried %v", kinds, retried)
	}
	var svc []string
	_ = json.Unmarshal(end.Fields["services"], &svc)
	if !reflect.DeepEqual(svc, []string{ServiceNetwork}) {
		t.Fatalf("outage_end services = %v", svc)
	}
	if end.String("how") != "probe" || end.String("started_at") != t0.Format(time.RFC3339) {
		t.Fatalf("outage_end = %+v", end.Fields)
	}
	if cur, err := r.Outage().Current(); err != nil || cur != nil {
		t.Fatalf("Current after the end = %+v, %v; want none", cur, err)
	}

	// A hand clear closes the next one and logs the person's intervention.
	recordOutage(t, r, "alpha", ServiceNetwork, "agent", "lane-3", "sub-agent died")
	c.advance(2 * time.Minute)
	if _, err := r.Outage().Clear("alpha", " "); !errors.Is(err, ErrRefused) {
		t.Fatalf("a clear without a reason = %v; want a refusal", err)
	}
	cl, err := r.Outage().Clear("alpha", "the router was restarted")
	if err != nil || cl.Minutes != 2 {
		t.Fatalf("Clear = %+v, %v", cl, err)
	}
	if e := lastEvent(t, r, EventOutageEnd); e.String("how") != "cleared" || e.String("reason") != "the router was restarted" {
		t.Fatalf("cleared outage_end = %+v", e.Fields)
	}
	if e := lastEvent(t, r, EventIntervention); e.String("kind") != "other" || e.String("why") != "the router was restarted" {
		t.Fatalf("clear's intervention = %+v", e.Fields)
	}
	if _, err := r.Outage().Clear("alpha", "again"); !errors.Is(err, ErrRefused) {
		t.Fatalf("clearing no outage = %v; want a refusal", err)
	}
}

// countEvents counts the lines of one event.
func countEvents(t *testing.T, r *Run, name string) int {
	t.Helper()
	n := 0
	for _, e := range eventNames(t, r) {
		if e == name {
			n++
		}
	}
	return n
}

// TestOutageGiveUpNotifiesUntilAck: the give-up stops probing, logs
// outage_give_up once, and holds notify_pending until a session acknowledges
// it; the acknowledgement is not raised again.
func TestOutageGiveUpNotifiesUntilAck(t *testing.T) {
	r, c := newRun(t)
	join(t, r, "alpha", RoleFirst)
	t0 := c.now()
	recordOutage(t, r, "alpha", ServiceNetwork, "tool", "lane-1", "git push")
	if _, err := r.Outage().Ack("alpha"); !errors.Is(err, ErrRefused) {
		t.Fatalf("an ack with nothing to acknowledge = %v; want a refusal", err)
	}
	for {
		cur, err := r.Outage().Current()
		if err != nil {
			t.Fatal(err)
		}
		if cur.Status == OutageGaveUp {
			break
		}
		c.t = cur.NextProbeAt
		if _, err := r.Outage().ProbeIfDue("alpha", netDown); err != nil {
			t.Fatal(err)
		}
	}
	if n := countEvents(t, r, EventOutageGiveUp); n != 1 {
		t.Fatalf("%d outage_give_up lines; want one", n)
	}
	g := lastEvent(t, r, EventOutageGiveUp)
	if m, _ := g.Number("minutes"); m != (8*time.Hour + 16*time.Minute).Minutes() {
		t.Fatalf("outage_give_up minutes = %v", m)
	}
	if g.String("hourly_since") != t0.Add(16*time.Minute).Format(time.RFC3339) {
		t.Fatalf("outage_give_up = %+v", g.Fields)
	}
	if countEvents(t, r, EventOutageEnd) != 0 {
		t.Fatal("a give-up also logged an outage_end")
	}

	c.advance(time.Hour)
	if _, err := r.Outage().ProbeIfDue("alpha", netDown); !errors.Is(err, ErrRefused) {
		t.Fatalf("a probe after the give-up = %v; want a refusal (the run has stopped)", err)
	}
	if countEvents(t, r, EventOutageProbe) != 11 {
		t.Fatal("a probe ran after the give-up")
	}
	cur, _ := r.Outage().Current()
	if !cur.NotifyPending {
		t.Fatal("notify_pending is not raised at the give-up")
	}
	if cur, err := r.Outage().Ack("alpha"); err != nil || cur.NotifyPending || cur.Status != OutageGaveUp {
		t.Fatalf("Ack = %+v, %v", cur, err)
	}
	if cur, _ := r.Outage().Current(); cur.NotifyPending {
		t.Fatal("notify_pending raised again after the ack")
	}
	// A report after the give-up joins the stopped outage; it opens no second
	// notification.
	recordOutage(t, r, "alpha", ServiceNetwork, "tool", "lane-1", "git fetch")
	if cur, _ := r.Outage().Current(); cur.NotifyPending || cur.Status != OutageGaveUp {
		t.Fatalf("after a late report = %+v", cur)
	}
	if _, err := r.Outage().Ack("alpha"); !errors.Is(err, ErrRefused) {
		t.Fatalf("a second ack = %v; want a refusal (nothing pending)", err)
	}
	if _, err := r.Outage().Clear("alpha", "network restored by hand"); err != nil {
		t.Fatalf("clearing a stopped outage: %v", err)
	}
	if cur, _ := r.Outage().Current(); cur != nil {
		t.Fatalf("after the clear = %+v", cur)
	}
}

// TestOutageModelServiceNeedsTheCanary: the model service being down is the
// same outage — one record, one schedule, one lease — and only a canary's
// verdict proves it back. A network probe that succeeds leaves it down; a probe
// with no model verdict leaves it unproven; a mixed outage ends only when both
// services are proven back.
func TestOutageModelServiceNeedsTheCanary(t *testing.T) {
	r, c := newRun(t)
	join(t, r, "alpha", RoleFirst)
	t0 := c.now()
	o := recordOutage(t, r, "alpha", ServiceModel, "agent", "lane-1", "API Error: 529 overloaded")
	if !reflect.DeepEqual(o.Down, []string{ServiceModel}) || !reflect.DeepEqual(o.Services, []string{ServiceModel}) {
		t.Fatalf("model outage = %+v", o)
	}
	if e := lastEvent(t, r, EventOutageStart); e.String("service") != ServiceModel || e.String("kind") != "agent" {
		t.Fatalf("outage_start = %+v", e.Fields)
	}
	if _, err := r.Outage().Record("alpha", "power", "agent", "lane-1", "x"); !errors.Is(err, ErrRefused) {
		t.Fatalf("an unknown service = %v; want a refusal", err)
	}

	// The network is fine and no canary ran: the model side stays down, and the
	// probe counts as a failed one on the shared schedule.
	var netRan int
	c.t = t0.Add(time.Minute)
	out, err := r.Outage().ProbeIfDue("alpha", Prober{Network: func() (bool, string) { netRan++; return true, "" }})
	if err != nil || out.Ended || !out.Probed || out.Probe.OK {
		t.Fatalf("a network-ok probe of a model outage = %+v, %v; want it still down", out, err)
	}
	if netRan != 0 {
		t.Fatal("the network was probed although only the model service is down")
	}
	if len(out.Probe.Results) != 1 || out.Probe.Results[0].Service != ServiceModel || out.Probe.Results[0].OK ||
		!strings.Contains(out.Probe.Results[0].Detail, "unproven") {
		t.Fatalf("probe results = %+v; want the model side unproven", out.Probe.Results)
	}
	if !out.Outage.NextProbeAt.Equal(c.now().Add(5 * time.Minute)) {
		t.Fatalf("next_probe_at = %s; want the schedule's second wait", out.Outage.NextProbeAt)
	}

	// The canary says it is still overloaded: still down.
	c.t = out.Outage.NextProbeAt
	out, err = r.Outage().ProbeIfDue("alpha", Prober{Model: func() (bool, string) { return false, "529 overloaded" }})
	if err != nil || out.Ended {
		t.Fatalf("a failed canary = %+v, %v", out, err)
	}
	// The canary answers: the outage ends.
	c.t = out.Outage.NextProbeAt
	out, err = r.Outage().ProbeIfDue("alpha", Prober{Model: func() (bool, string) { return true, "canary answered" }})
	if err != nil || !out.Ended {
		t.Fatalf("a canary that answered = %+v, %v; want the outage ended", out, err)
	}
	var services []string
	_ = json.Unmarshal(lastEvent(t, r, EventOutageEnd).Fields["services"], &services)
	if !reflect.DeepEqual(services, []string{ServiceModel}) {
		t.Fatalf("outage_end services = %v", services)
	}

	// Network and model both down: the network back alone does not end it, and
	// the model's canary then does, without the network re-proven.
	recordOutage(t, r, "alpha", ServiceNetwork, "tool", "lane-1", "git push")
	recordOutage(t, r, "alpha", ServiceModel, "host", "lane-2", "Request timed out")
	cur, _ := r.Outage().Current()
	c.t = cur.NextProbeAt
	out, err = r.Outage().ProbeIfDue("alpha", Prober{Network: netUp})
	if err != nil || out.Ended || !reflect.DeepEqual(out.Outage.Down, []string{ServiceModel}) {
		t.Fatalf("network back, model unproven = %+v, %v; want only the model still down", out, err)
	}
	c.t = out.Outage.NextProbeAt
	out, err = r.Outage().ProbeIfDue("alpha", Prober{Network: failing, Model: netUp})
	if err != nil || !out.Ended {
		t.Fatalf("the canary back with the network already proven = %+v, %v; want the outage ended", out, err)
	}
	_ = json.Unmarshal(lastEvent(t, r, EventOutageEnd).Fields["services"], &services)
	if !reflect.DeepEqual(services, []string{ServiceModel, ServiceNetwork}) {
		t.Fatalf("mixed outage_end services = %v", services)
	}

	// Mixed again: the canary back alone does not end it either.
	recordOutage(t, r, "alpha", ServiceNetwork, "tool", "lane-1", "git push")
	recordOutage(t, r, "alpha", ServiceModel, "agent", "lane-2", "API Error: 500")
	cur, _ = r.Outage().Current()
	c.t = cur.NextProbeAt
	out, err = r.Outage().ProbeIfDue("alpha", Prober{Network: failing, Model: netUp})
	if err != nil || out.Ended || !reflect.DeepEqual(out.Outage.Down, []string{ServiceNetwork}) {
		t.Fatalf("model back, network down = %+v, %v; want only the network still down", out, err)
	}
}

// TestIsModelFailure: the model service's own failures count; a usage or rate
// limit is not an outage.
func TestIsModelFailure(t *testing.T) {
	for _, c := range []struct {
		out  string
		want bool
	}{
		{`API Error: 529 {"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`, true},
		{"API Error: 500 Internal Server Error", true},
		{"Overloaded", true},
		{"API Error: Request timed out.", true},
		{"API Error: Connection error.", true},
		{"503 Service Unavailable", true},
		{"status 529", true},

		{"", false},
		{"Claude usage limit reached. Your limit will reset at 5pm.", false},
		{"API Error: 429 rate limit exceeded", false},
		{`API Error: 429 {"type":"error","error":{"type":"rate_limit_error"}}`, false},
		{"the agent finished: 15290 lines read, all tests pass", false},
		{"go test: FAIL TestServer500Path", false},
	} {
		if got := IsModelFailure(c.out); got != c.want {
			t.Errorf("IsModelFailure(%q) = %v, want %v", c.out, got, c.want)
		}
	}
}

// TestIsNetworkFailure: the network errors git and gh actually print count; an
// authentication failure, a hook's refusal and a merge conflict do not, even
// when their output also carries a network-sounding line.
func TestIsNetworkFailure(t *testing.T) {
	for _, c := range []struct {
		stderr string
		want   bool
	}{
		{"fatal: unable to access 'https://github.com/o/r.git/': Could not resolve host: github.com", true},
		{"ssh: Could not resolve hostname github.com: nodename nor servname provided, or not known", true},
		{"fatal: unable to access 'https://github.com/o/r.git/': Failed to connect to github.com port 443 after 75004 ms: Couldn't connect to server", true},
		{"ssh: connect to host github.com port 22: Connection refused", true},
		{"ssh: connect to host github.com port 22: Connection timed out", true},
		{"ssh: connect to host github.com port 22: Operation timed out", true},
		{"ssh: connect to host github.com port 22: Network is unreachable", true},
		{"error connecting to api.github.com\ncheck your internet connection or https://githubstatus.com", true},
		{`Post "https://api.github.com/graphql": dial tcp: lookup api.github.com: no such host`, true},
		{`Get "https://api.github.com/repos/o/r": net/http: TLS handshake timeout`, true},
		{"error: RPC failed; curl 92 HTTP/2 stream 0 was not closed cleanly\nfatal: unexpected disconnect while reading sideband packet", true},
		{"fatal: unable to access 'https://github.com/o/r.git/': Operation timed out after 300000 milliseconds", true},
		{"read tcp 192.0.2.2:51234->203.0.113.6:443: read: connection reset by peer", true},
		{"fatal: unable to access 'https://github.com/': Temporary failure in name resolution", true},
		{"ssh: connect to host github.com port 22: No route to host", true},

		{"", false},
		{"remote: Invalid username or token.\nfatal: Authentication failed for 'https://github.com/o/r.git/'", false},
		{"git@github.com: Permission denied (publickey).\nfatal: Could not read from remote repository.", false},
		{"fatal: could not read Username for 'https://github.com': terminal prompts disabled", false},
		{"HTTP 401: Bad credentials (https://api.github.com/graphql)", false},
		{"HTTP 403: Resource not accessible by integration", false},
		{"remote: error: GH006: Protected branch update failed\n ! [remote rejected] main -> main (protected branch hook declined)", false},
		{" ! [remote rejected] x -> x (pre-receive hook declined)\nerror: failed to push some refs", false},
		{"pre-push: no preflight receipt for 1234abc; run make preflight\nerror: failed to push some refs to 'origin'", false},
		{" ! [rejected]        main -> main (fetch first)\nerror: failed to push some refs", false},
		{"CONFLICT (content): Merge conflict in go.mod\nAutomatic merge failed; fix conflicts and then commit the result.", false},
		{"error: could not apply 1234abc... fix\nhint: Resolve all conflicts manually", false},
		// A hook may print anything: its refusal wins over a word it quoted.
		{"pre-push hook: Connection refused to the receipt store\nerror: failed to push some refs (pre-receive hook declined)", false},
	} {
		if got := IsNetworkFailure(c.stderr); got != c.want {
			t.Errorf("IsNetworkFailure(%q) = %v, want %v", c.stderr, got, c.want)
		}
	}
}

// TestLogRefusesOutageEventsByHand: the four outage events are written by the
// outage verbs alone, and the verbs' own lines are held to their fields.
func TestLogRefusesOutageEventsByHand(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	for _, ev := range []string{EventOutageStart, EventOutageProbe, EventOutageEnd, EventOutageGiveUp} {
		_, err := r.Log("alpha", ev, map[string]string{"minutes": "3", "kinds": "tool", "retried": "git push",
			"started_at": "2026-10-09T00:00:00Z", "service": "network", "services": "network", "kind": "tool", "lane": "l", "what": "x", "ok": "true", "failures": "0",
			"hourly_since": "2026-10-09T00:00:00Z"})
		if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "written by the implement verbs") {
			t.Errorf("hand-logged %s = %v; want the verb-owned refusal", ev, err)
		}
	}
	if n := len(eventNames(t, r)); n != 1 {
		t.Fatalf("the refused lines reached the log (%d lines)", n)
	}
	for _, c := range []struct {
		event  string
		fields map[string]any
	}{
		{EventOutageEnd, map[string]any{"minutes": -1.0, "services": []string{"network"}, "kinds": []string{"tool"}, "retried": []string{"x"}, "started_at": "2026-10-09T00:00:00Z"}},
		{EventOutageEnd, map[string]any{"minutes": 1.0, "services": []string{"network"}, "kinds": []string{}, "retried": []string{"x"}, "started_at": "2026-10-09T00:00:00Z"}},
		{EventOutageGiveUp, map[string]any{"minutes": 1.0, "services": []string{"network"}, "kinds": []string{"tool"}, "retried": []string{"x"}, "started_at": "2026-10-09T00:00:00Z"}},
		{EventOutageStart, map[string]any{"service": "network", "kind": "wifi", "lane": "l", "what": "x"}},
		{EventOutageStart, map[string]any{"service": "power", "kind": "tool", "lane": "l", "what": "x"}},
		{EventOutageEnd, map[string]any{"minutes": 1.0, "services": []string{}, "kinds": []string{"tool"}, "retried": []string{"x"}, "started_at": "2026-10-09T00:00:00Z"}},
		{EventOutageProbe, map[string]any{"ok": "maybe", "failures": 1}},
	} {
		if err := checkOutageFields(c.event, c.fields); !errors.Is(err, ErrRefused) {
			t.Errorf("checkOutageFields(%s, %v) = %v; want a refusal", c.event, c.fields, err)
		}
	}
}

// TestCompareReportsOutages: the report reads each outage from the log alone —
// its start, its end or give-up, the minutes, the kinds and what was retried —
// and an outage still open at the log's end says so.
func TestCompareReportsOutages(t *testing.T) {
	log := strings.Join([]string{
		`{"ts":"2026-10-09T10:00:00Z","session":"A","event":"session_open","role":"first"}`,
		`{"ts":"2026-10-09T10:05:00Z","session":"A","event":"outage_start","service":"network","kind":"tool","lane":"l1","what":"git push"}`,
		`{"ts":"2026-10-09T10:06:00Z","session":"A","event":"outage_probe","ok":false,"failures":1}`,
		`{"ts":"2026-10-09T10:11:00Z","session":"B","event":"outage_probe","ok":true,"failures":1}`,
		`{"ts":"2026-10-09T10:11:00Z","session":"B","event":"outage_end","how":"probe","minutes":6,"services":["model","network"],"kinds":["host","tool"],"retried":["git push","model call"],"started_at":"2026-10-09T10:05:00Z","probes":2}`,
		`{"ts":"2026-10-09T12:00:00Z","session":"A","event":"outage_start","service":"model","kind":"agent","lane":"l2","what":"sub-agent"}`,
		`{"ts":"2026-10-09T20:16:00Z","session":"A","event":"outage_give_up","minutes":496,"services":["model"],"kinds":["agent"],"retried":["sub-agent"],"started_at":"2026-10-09T12:00:00Z","hourly_since":"2026-10-09T12:16:00Z","probes":11}`,
		`{"ts":"2026-10-09T21:00:00Z","session":"A","event":"outage_start","service":"model","kind":"host","lane":"l3","what":"model call"}`,
		`{"ts":"2026-10-09T21:30:00Z","session":"A","event":"lane_open","lane":"l4"}`,
	}, "\n")
	events, bad := ParseLog("outages.jsonl", []byte(log))
	if len(bad) != 0 {
		t.Fatalf("fixture lines unparsed: %+v", bad)
	}
	rep := Compare(events, bad)
	if len(rep.Outages) != 3 {
		t.Fatalf("outages = %+v; want 3", rep.Outages)
	}
	ended, gaveUp, open := rep.Outages[0], rep.Outages[1], rep.Outages[2]
	if ended.Outcome != "ended" || ended.Minutes != 6 || ended.Probes != 2 || ended.End == nil ||
		!ended.Start.Equal(time.Date(2026, 10, 9, 10, 5, 0, 0, time.UTC)) ||
		!reflect.DeepEqual(ended.Kinds, []string{"host", "tool"}) || !reflect.DeepEqual(ended.Services, []string{"model", "network"}) || !reflect.DeepEqual(ended.Retried, []string{"git push", "model call"}) {
		t.Errorf("ended outage = %+v", ended)
	}
	if gaveUp.Outcome != "gave_up" || gaveUp.Minutes != 496 || gaveUp.Probes != 11 || !reflect.DeepEqual(gaveUp.Kinds, []string{"agent"}) {
		t.Errorf("gave-up outage = %+v", gaveUp)
	}
	if open.Outcome != "open" || open.End != nil || open.Minutes != 30 || !reflect.DeepEqual(open.Kinds, []string{"host"}) || !reflect.DeepEqual(open.Services, []string{"model"}) ||
		!reflect.DeepEqual(open.Retried, []string{"model call"}) {
		t.Errorf("open outage = %+v", open)
	}
	if empty := Compare(nil, nil); empty.Outages == nil || len(empty.Outages) != 0 {
		t.Errorf("an empty log's outages = %#v; want an empty list", empty.Outages)
	}
}

// TestRemoteProbeAnswersFromALocalOrigin: the production network probe, with
// no network — an origin on disk answers, and a checkout whose origin is
// reached but holds no HEAD is reached all the same.
func TestRemoteProbeAnswersFromALocalOrigin(t *testing.T) {
	origin := gittest.NewRepo(t)
	origin.Write("a.txt", "a\n")
	origin.Commit("init")
	repo := gittest.NewRepo(t)
	repo.Git("remote", "add", "origin", origin.Root())
	if ok, detail := RemoteProbe(repo.Root())(); !ok {
		t.Fatalf("a local origin = down (%s); want it answered", detail)
	}
	bare := filepath.Join(t.TempDir(), "bare.git")
	origin.Git("init", "--bare", "-q", bare)
	headless := gittest.NewRepo(t)
	headless.Git("remote", "add", "origin", bare)
	if ok, detail := RemoteProbe(headless.Root())(); !ok || !strings.Contains(detail, "no HEAD") {
		t.Fatalf("an origin with no HEAD = down (%s); want it answered", detail)
	}
	lone := gittest.NewRepo(t)
	if ok, detail := RemoteProbe(lone.Root())(); !ok || !strings.Contains(detail, "refused") {
		t.Fatalf("a checkout with no origin = %v %q; want reached-but-refused (not a network failure)", ok, detail)
	}
}
