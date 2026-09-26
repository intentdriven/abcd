package guard

import "testing"

// TestPushdChainsLikeCD — review-guard finding 6. `pushd` and `popd` change
// directory exactly as `cd` does, and fail the same way, so a delete chained
// after one runs wherever the shell already was when the change fails. The
// after_cd constraint knew only `cd`.
func TestPushdChainsLikeCD(t *testing.T) {
	cases := []struct {
		cmd   string
		want  Verdict
		entry string
	}{
		{`pushd s && rm -rf *`, VerdictBlock, "rm-rf-after-cd-chain"},
		{`pushd s; rm -rf .`, VerdictBlock, "rm-rf-after-cd-chain"},
		{`popd && rm -rf *`, VerdictBlock, "rm-rf-after-cd-chain"},

		{`pushd s && ls -la`, VerdictAllow, ""},
		{"pushd s\nrm -rf ./build", VerdictAllow, ""},
	}
	for _, tc := range cases {
		t.Run(tc.cmd, func(t *testing.T) {
			d := verdictOf(t, tc.cmd)
			if d.Verdict != tc.want || d.EntryID != tc.entry {
				t.Errorf("verdict = %q via %q, want %q via %q", d.Verdict, d.EntryID, tc.want, tc.entry)
			}
		})
	}
}
