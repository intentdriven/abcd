package loop

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestReceiptRefusalsSealAGluedToken — iss-2609290541525428. A receipt's
// undeclared key and a missing report path inside the lane's directory are
// named through scanner.RedactRefusal, whose patterns anchor on a leading \b,
// so a token glued behind an underscore or a letter came back raw. Each is
// still named, with the token sealed.
func TestReceiptRefusalsSealAGluedToken(t *testing.T) {
	pat := "gh" + "p_" + strings.Repeat("D", 36)
	akia := "AK" + "IA" + strings.Repeat("Q", 16)
	patBody, akiaBody := strings.Repeat("D", 6), strings.Repeat("Q", 6)
	cases := []struct {
		name        string
		edit        func(rc *LaneReceipt) any
		token, body string
		names       string
	}{
		{"report path, pat behind an underscore", func(rc *LaneReceipt) any {
			rc.Report = "notes_" + pat + ".md"
			return rc
		}, pat, patBody, "notes_"},
		{"report path, access key between two letters", func(rc *LaneReceipt) any {
			rc.Report = "x" + akia + "y.md"
			return rc
		}, akia, akiaBody, "does not exist"},
		{"undeclared key, pat behind an underscore", func(rc *LaneReceipt) any {
			b, _ := json.Marshal(rc)
			return strings.Replace(string(b), `"schema_version":1`, `"schema_version":1,"notes_`+pat+`":1`, 1)
		}, pat, patBody, "notes_"},
		{"undeclared key, access key between two letters", func(rc *LaneReceipt) any {
			b, _ := json.Marshal(rc)
			return strings.Replace(string(b), `"schema_version":1`, `"schema_version":1,"x`+akia+`y":1`, 1)
		}, akia, akiaBody, "does not parse as a receipt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, runID, l, dir := awaitingLane(t)
			c1 := laneCommit(t, repo, l, "one.txt")
			rc := goodReceipt(t, runID, l, dir, c1)
			path := writeReceipt(t, dir, tc.edit(&rc))

			_, err := Receipt(repo.Root(), runID, path, DefaultStages(), Options{})
			r := mustRefusal(t, err)
			for _, s := range []string{err.Error(), r.Reason} {
				if strings.Contains(s, tc.token) || strings.Contains(s, tc.body) {
					t.Errorf("the refusal echoes the glued token: %q", s)
				}
			}
			if !strings.Contains(r.Reason, tc.names) {
				t.Errorf("the refusal no longer names %q: %q", tc.names, r.Reason)
			}
		})
	}
}
