package loop

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// receiptLeak is a receipt value no refusal may carry back: a marker and a
// third party's absolute home path.
const receiptLeak = "zzleak-7f3a /Users/zzotherperson/notes" // abcd-lint:allow — a planted home path the refusal must not echo

// TestReceiptRefusalsDoNotEchoThePayload — iss-2609290300462829. The receipt is
// written by the implementer agent, a host payload, and its refusals quoted the
// strict decoder's message raw, a foreign run, lane and branch, a malformed
// commit name and a path outside the lane's directory with %q or %v. A token
// or a home path in any of them reached the terminal and the transcript. A
// value is described now, a path inside the lane's directory and an undeclared
// key are named redacted, and each refusal still names what is missing.
func TestReceiptRefusalsDoNotEchoThePayload(t *testing.T) {
	// The harness pins HOME per test (awaitingLane), so the caller's home is
	// read when the receipt is written, and must not come back either.
	cases := []struct {
		name  string
		edit  func(t *testing.T, repo *gittest.Repo, l Lane, dir string, rc *LaneReceipt) any
		names string
		leaks []string
	}{
		{"undeclared key", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			b, _ := json.Marshal(rc)
			return strings.Replace(string(b), `"schema_version":1`,
				`"schema_version":1,"reviewer_notes /Users/zzotherperson/notes `+os.Getenv("HOME")+`/x":1`, 1) // abcd-lint:allow — a planted home path in a KEY
		}, "reviewer_notes", []string{"zzotherperson", "$HOME"}},
		{"run and lane", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			rc.RunID, rc.Lane = receiptLeak, receiptLeak
			return rc
		}, "the run and lane", []string{"zzleak-7f3a", "zzotherperson"}},
		{"branch", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			rc.Branch = receiptLeak
			return rc
		}, "the lane's branch", []string{"zzleak-7f3a", "zzotherperson"}},
		{"commit name", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			rc.Commits = append(rc.Commits, receiptLeak)
			return rc
		}, "not a full object name", []string{"zzleak-7f3a", "zzotherperson"}},
		{"report outside the lane's directory", func(t *testing.T, _ *gittest.Repo, _ Lane, _ string, rc *LaneReceipt) any {
			rc.Report = "/Users/zzotherperson/notes/report.md" // abcd-lint:allow — a planted home path the refusal must not echo
			return rc
		}, "is not a path inside the lane's directory", []string{"zzotherperson"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, runID, l, dir := awaitingLane(t)
			c1 := laneCommit(t, repo, l, "one.txt")
			rc := goodReceipt(t, runID, l, dir, c1)
			path := writeReceipt(t, dir, tc.edit(t, repo, l, dir, &rc))

			_, err := Receipt(repo.Root(), runID, path, DefaultStages(), Options{})
			r := mustRefusal(t, err)
			for _, leak := range tc.leaks {
				if leak == "$HOME" {
					leak = os.Getenv("HOME")
				}
				if strings.Contains(err.Error(), leak) || strings.Contains(r.Reason, leak) {
					t.Errorf("the refusal echoes the receipt (%q): %q", leak, r.Reason)
				}
			}
			if !strings.Contains(r.Reason, tc.names) {
				t.Errorf("the refusal no longer names %q: %q", tc.names, r.Reason)
			}
		})
	}
}
