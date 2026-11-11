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
