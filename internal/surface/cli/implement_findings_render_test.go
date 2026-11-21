package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// TestRenderRunRecordNamesEachReviewersLocatedFindings: the record's text
// form counts each reviewer's located findings on its verdict line, zero
// included, and names each with what the fix report said of it and who else
// in the round cites it (itd-2610240922461281, spc-2610240951109420 step 4).
func TestRenderRunRecordNamesEachReviewersLocatedFindings(t *testing.T) {
	two, zero := 2, 0
	rec := loop.RunRecord{RunID: "run-1", Lanes: []loop.RecordLane{{ID: "lane-1", Verdicts: []loop.RecordVerdict{
		{Round: 1, Role: loop.RoleRuthless, Verdict: "FIX FIRST", Reviewer: 1, Located: &two, Findings: []loop.Finding{
			{File: "a.go", Line: 3, Status: loop.FindingApplied, CitedBy: []loop.Citer{{Role: loop.RoleRuthless}, {Role: loop.RoleRuthless, Reviewer: 1}}},
			{File: "b.go", Line: 7, Status: loop.FindingUnknown},
		}},
		{Round: 1, Role: loop.RoleSecurity, Verdict: "APPROVE", Located: &zero},
		{Round: 1, Role: loop.RoleAuditor, Verdict: "MET"},
	}}}}
	var b bytes.Buffer
	renderRunRecord(&b, rec)
	out := b.String()
	for _, want := range []string{
		"FIX FIRST, 2 located finding(s)",
		"      a.go:3  applied  cited by ruthless-reviewer 0, ruthless-reviewer 1",
		"      b.go:7  unknown\n",
		"APPROVE, 0 located finding(s)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("the record's text names %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "MET, ") {
		t.Fatalf("the auditor's verdict counts no findings:\n%s", out)
	}
}

// TestRenderRunRecordNamesEachStrikeBesideItsJudge: the record's text form
// names each round's judge on a line of its own and, beneath it, each
// location it struck with its reason, which the judge keeps once the round's
// findings are dropped; a judge whose strikes were not kept says so
// (iss-2611182238519860).
func TestRenderRunRecordNamesEachStrikeBesideItsJudge(t *testing.T) {
	one := 1
	rec := loop.RunRecord{RunID: "run-1", Lanes: []loop.RecordLane{{ID: "lane-1",
		Verdicts: []loop.RecordVerdict{{Round: 1, Role: loop.RoleRuthless, Verdict: "FIX FIRST", Reviewer: 1, Located: &one}},
		Judges: []loop.RecordJudge{
			{Round: 1, RoundJudge: loop.RoundJudge{Roles: []string{loop.RoleRuthless}, Class: "top", Model: "m-1", Effort: "high", Ruled: true, Struck: 1,
				Strikes: []loop.JudgeStrike{{File: "a.go", Line: 3, Reason: "the caller checks it\x1b[2J"}}}},
			{Round: 2, RoundJudge: loop.RoundJudge{Roles: []string{loop.RoleRuthless}, Class: "top", Model: "m-1", Ruled: true, Struck: 2}},
			{Round: 3, RoundJudge: loop.RoundJudge{Roles: []string{loop.RoleSecurity}, Class: "top", Model: "m-1", Unlaunched: "the gate no longer passes"}},
		},
	}}}
	var b bytes.Buffer
	renderRunRecord(&b, rec)
	out := b.String()
	for _, want := range []string{
		"    round 1 judge, top (m-1, effort high), on ruthless-reviewer: ruled, struck 1 located finding(s)\n",
		"      a.go:3  struck: the caller checks it",
		"    round 2 judge, top (m-1), on ruthless-reviewer: ruled, struck 2 located finding(s)\n",
		"      the locations and reasons of its strikes were not kept: ",
		"    round 3 judge, top (m-1), on security-reviewer: not launched: the gate no longer passes\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("the record's text names %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\x1b") {
		t.Fatalf("a strike's reason is shown sanitized:\n%q", out)
	}
}
