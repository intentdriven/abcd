package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// A run's lost connection (iss-2610080620372731) reads in `implement step`, in
// `implement status`, in the give-up's refusal and in `implement record`, in
// text and in --json.
func TestOutageRendersInStepStatusRefusalAndRecord(t *testing.T) {
	since := time.Date(2026, 10, 9, 9, 0, 0, 0, time.UTC)
	next := since.Add(time.Minute)
	in := &loop.OutageInfo{Since: since, Services: []string{"network"}, Down: []string{"network"}, NextProbeAt: &next, Retried: []string{"git push build/x"}}

	var b bytes.Buffer
	renderStepResult(&b, "step", loop.StepResult{RunID: "run-1", Outage: in, Next: "wait"})
	if !strings.Contains(b.String(), "outage: network down since 2026-10-09T09:00:00Z; 0 probe(s); next probe 09:01") {
		t.Fatalf("the step names the outage:\n%s", b.String())
	}

	b.Reset()
	cur := &implement.Outage{StartedAt: since, Services: []string{"network"}, Down: []string{"network"}, Kinds: []string{"tool"},
		NextProbeAt: next, Status: implement.OutageOpen}
	renderStatusOutage(&b, cur)
	if !strings.Contains(b.String(), "outage open since 2026-10-09T09:00:00Z") || !strings.Contains(b.String(), "down:     network") {
		t.Fatalf("the status names the outage:\n%s", b.String())
	}
	raw, err := json.Marshal(implementStatusRuns{Runs: []loop.State{}, Outage: cur})
	if err != nil || !strings.Contains(string(raw), `"outage":{"started_at":"2026-10-09T09:00:00Z"`) {
		t.Fatalf("the status's --json carries the outage: %s %v", raw, err)
	}

	gave := since.Add(9 * time.Hour)
	stop := &loop.Refusal{Stage: loop.StageOutage, Reason: "the run gave up; done: landed lane-1; left: lane-2 at implement", Remedy: "tell the product thinker",
		Outage: &loop.OutageInfo{Since: since, Services: []string{"network"}, Down: []string{"network"}, Probes: 11, GaveUpAt: &gave, Notify: true,
			Done: &loop.OutageDone{Landed: []string{"lane-1"}, PullRequests: []loop.OutagePR{}},
			Left: &loop.OutageLeft{Lanes: []loop.OutageLane{{Lane: "lane-2", SpecStep: 2, Stage: loop.StageImplement}}, Pending: []loop.PendingStep{}}}}
	b.Reset()
	err = loopFail(&b, true, "abcd implement step", stop)
	var ee *exitError
	if !errors.As(err, &ee) || ee.Code != 2 || !strings.Contains(ee.Msg, "landed lane-1") {
		t.Fatalf("the give-up is a refusal, exit 2, naming what was done: %v", err)
	}
	for _, want := range []string{`"stage": "outage"`, `"landed": [`, `"lane-1"`, `"stage": "implement"`, `"notify": true`, `"probes": 11`} {
		if !strings.Contains(b.String(), want) {
			t.Fatalf("the refusal's --json carries %s:\n%s", want, b.String())
		}
	}

	b.Reset()
	end := since.Add(3 * time.Minute)
	renderRunRecord(&b, loop.RunRecord{RunID: "run-1", Outages: []implement.OutageSpan{{Start: since, End: &end, Outcome: "ended", Minutes: 3,
		Services: []string{"network"}, Kinds: []string{"tool"}, Retried: []string{"git push build/x"}, Probes: 1}}})
	if !strings.Contains(b.String(), "outages: 1") || !strings.Contains(b.String(), "2026-10-09T09:00:00Z  ended after 3 minute(s): network down, 1 probe(s); retried git push build/x") {
		t.Fatalf("the record names each outage:\n%s", b.String())
	}
}
