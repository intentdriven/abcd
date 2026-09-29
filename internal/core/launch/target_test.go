package launch

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestValidTargetReleaseAdmitsNextAndATagOnly pins the one shape the intent
// verbs write, the record lint admits and the cut reads
// (itd-2609212103572513 criterion 1): `next`, or a bare-core release tag.
func TestValidTargetReleaseAdmitsNextAndATagOnly(t *testing.T) {
	for _, ok := range []string{"next", "v0.11.0", "v1.0.0", "v10.20.30"} {
		if err := ValidTargetRelease(ok); err != nil {
			t.Errorf("ValidTargetRelease(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"", "0.11.0", "v0.11", "v0.11.0-rc.1", "v0.11.0+build", "V0.11.0", "v01.1.0", "Next", "later", " v0.11.0", "v0.11.0\n"} {
		err := ValidTargetRelease(bad)
		if err == nil {
			t.Errorf("ValidTargetRelease(%q) = nil, want a refusal", bad)
			continue
		}
		if !strings.Contains(err.Error(), "vX.Y.Z") || !strings.Contains(err.Error(), "next") {
			t.Errorf("the refusal must name both accepted shapes: %v", err)
		}
	}
}

// TestDryRunListsTargetedIntentsAndStillRefusesNothing is criterion 2 on the
// preview: given a targeted intent still planned, when `launch --dry-run`
// runs, then it is listed as targeted and unshipped in the report (and so in
// --json), in the pre-flight report both files carry, and the list adds no
// refusal.
func TestDryRunListsTargetedIntentsAndStillRefusesNothing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ArtefactRelPath, `{"kind": "plugin"}`)
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": ["commands"]}`)
	writeFile(t, root, "commands/x.md", "# doc\n")

	baseline, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	targets := []TargetedIntent{
		{ID: "itd-7", Path: ".abcd/development/intents/planned/itd-7-seven.md", Target: "v0.11.0"},
		{ID: "itd-9", Path: ".abcd/development/intents/planned/itd-9-nine.md", Target: "next"},
	}
	rep, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.0.0", Targets: targets})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Targets) != 2 || rep.Targets[0].ID != "itd-7" || rep.Targets[1].Target != "next" {
		t.Fatalf("the preview must list every targeted intent: %+v", rep.Targets)
	}
	if strings.Join(rep.WouldRefuseOn, "|") != strings.Join(baseline.WouldRefuseOn, "|") {
		t.Fatalf("a target must never add a refusal:\n  without: %v\n  with:    %v", baseline.WouldRefuseOn, rep.WouldRefuseOn)
	}

	raw, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"targets":[{"id":"itd-7"`) || !strings.Contains(string(raw), `"target_release":"v0.11.0"`) {
		t.Fatalf("--json must carry the targeted list:\n%s", raw)
	}

	pre := rep.PreflightReport(time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC))
	if len(pre.Targets) != 2 {
		t.Fatalf("the pre-flight report must carry the targeted list: %+v", pre.Targets)
	}
	md := pre.Markdown()
	for _, want := range []string{"## Targeted, not shipped", "- itd-7 targets v0.11.0 (.abcd/development/intents/planned/itd-7-seven.md)", "- itd-9 targets next"} {
		if !strings.Contains(md, want) {
			t.Errorf("the pre-flight markdown lacks %q:\n%s", want, md)
		}
	}
	if pre.Verdict != baseline.PreflightReport(time.Now()).Verdict {
		t.Errorf("a target must not change the verdict: %s", pre.Verdict)
	}
}

// TestPreflightReportWithNoTargetsSaysNothingAboutThem: an untargeted run's
// report is unchanged, so a repository that never sets a target reads no new
// section.
func TestPreflightReportWithNoTargetsSaysNothingAboutThem(t *testing.T) {
	rep := DryRunReport{Version: "1.0.0"}.PreflightReport(time.Now())
	if strings.Contains(rep.Markdown(), "Targeted") {
		t.Fatalf("no target, no section:\n%s", rep.Markdown())
	}
}
