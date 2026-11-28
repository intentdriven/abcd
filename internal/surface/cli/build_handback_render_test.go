package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// kindHandBacks are hand-backs that carry a kind and no verdict: the lane's
// own receipt found a design decision, the loop found the lane stuck, or its
// pull request was closed (iss-2611101318284897).
var kindHandBacks = []loop.HandBack{
	{Kind: loop.HandBackDesignFinding, Reason: "the spec names two homes for the flag"},
	{Kind: loop.HandBackStuck, Reason: "the land stage refused three times"},
	{Kind: loop.HandBackClosed, Reason: "pull request #7 was closed without merging"},
}

// TestStepHandBackLineNamesTheKindWhenNoVerdict: the hand-back line of
// `implement step` and `implement receipt` names what stopped the lane when
// the hand-back carries no verdict, and never prints an empty field there.
func TestStepHandBackLineNamesTheKindWhenNoVerdict(t *testing.T) {
	for _, hb := range kindHandBacks {
		var b bytes.Buffer
		renderStepResult(&b, "step", loop.StepResult{RunID: "run-1", Lane: "lane-1", HandBack: &hb, Next: "hand back"})
		out := b.String()
		if !strings.Contains(out, "stopped as "+hb.Kind) {
			t.Errorf("the hand-back line names the kind %q:\n%s", hb.Kind, out)
		}
		if strings.Contains(out, "as  ") || strings.Contains(out, "after 0 fix round(s)") {
			t.Errorf("the hand-back line prints an empty verdict or a cap the lane never ran to:\n%s", out)
		}
	}
}

// TestStepHandBackLineKeepsTheVerdict: a hand-back the fix-round cap made is
// still named by its verdict and the cap it ran to.
func TestStepHandBackLineKeepsTheVerdict(t *testing.T) {
	hb := loop.HandBack{Verdict: loop.VerdictUnachievable, Round: 3, FixRounds: 2}
	var b bytes.Buffer
	renderStepResult(&b, "step", loop.StepResult{RunID: "run-1", Lane: "lane-1", HandBack: &hb, Next: "hand back"})
	if out := b.String(); !strings.Contains(out, "stopped as "+loop.VerdictUnachievable+" after 2 fix round(s)") {
		t.Fatalf("the hand-back line names the verdict and the cap:\n%s", out)
	}
}

// TestRunRecordHandBackNamesTheKindWhenNoVerdict: `implement record` names a
// lane's hand-back by its kind when it carries no verdict.
func TestRunRecordHandBackNamesTheKindWhenNoVerdict(t *testing.T) {
	for _, hb := range kindHandBacks {
		var b bytes.Buffer
		renderRunRecord(&b, loop.RunRecord{RunID: "run-1", Lanes: []loop.RecordLane{{ID: "lane-1", HandBack: &hb}}})
		out := b.String()
		if !strings.Contains(out, "handed back as "+hb.Kind) {
			t.Errorf("the record names the kind %q:\n%s", hb.Kind, out)
		}
		if strings.Contains(out, "as  ") || strings.Contains(out, "after 0 fix round(s)") {
			t.Errorf("the record prints an empty verdict or a cap the lane never ran to:\n%s", out)
		}
	}
}

// TestHandBackAsNamesTheKindWhenNoVerdict: `implement status` names a run's
// handed-back lane by its kind when it carries no verdict, and names
// something even when the hand-back recorded neither.
func TestHandBackAsNamesTheKindWhenNoVerdict(t *testing.T) {
	for _, hb := range kindHandBacks {
		if got := handBackAs(&hb); got != hb.Kind {
			t.Errorf("handBackAs(%+v) = %q, want the kind %q", hb, got, hb.Kind)
		}
	}
	if got := handBackAs(&loop.HandBack{Verdict: loop.VerdictUnachievable, Kind: loop.HandBackStuck}); got != loop.VerdictUnachievable {
		t.Errorf("a verdict names the hand-back before a kind, got %q", got)
	}
	if got := handBackAs(&loop.HandBack{}); strings.TrimSpace(got) == "" {
		t.Error("a hand-back with neither a verdict nor a kind still names something")
	}
}

// TestHandBackNamesNoCapForAVerdictTheCapDidNotSet: of the loop's three
// verdicts only unachievable is the fix-round cap's; a lane handed back as
// not-planned (an auto-plan lane) or cell-failed (a trial copy) never ran to
// the cap, so neither the step line nor the record names one.
func TestHandBackNamesNoCapForAVerdictTheCapDidNotSet(t *testing.T) {
	for _, v := range []string{loop.HandBackNotPlanned, loop.VerdictCellFailed} {
		hb := loop.HandBack{Verdict: v, Reason: "the plan review found the draft unready"}
		var step, rec bytes.Buffer
		renderStepResult(&step, "receipt", loop.StepResult{RunID: "run-1", Lane: "lane-1", HandBack: &hb, Next: "hand back"})
		renderRunRecord(&rec, loop.RunRecord{RunID: "run-1", Lanes: []loop.RecordLane{{ID: "lane-1", HandBack: &hb}}})
		for name, out := range map[string]string{"step": step.String(), "record": rec.String()} {
			if !strings.Contains(out, "as "+v) {
				t.Errorf("the %s names the verdict %q:\n%s", name, v, out)
			}
			if strings.Contains(out, "fix round(s)") {
				t.Errorf("the %s names a fix-round cap a %s lane never ran to:\n%s", name, v, out)
			}
		}
	}
}
