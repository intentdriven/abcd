package guard

import (
	"strings"
	"testing"
)

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

// TestBarePushdIsNotHome — iss-2610111815024967. A `pushd` with no operand
// swaps the top two directories of the stack in bash and zsh; it does not go
// to HOME. The reader took it as `cd`'s HOME, so the scripts after it were
// read in a directory the shell never enters. It now enters the directory
// the line itself pushed, and leaves the directory unknown where the line
// did not fix the stack, as it does for `pushd +N` and `-N`.
func TestBarePushdIsNotHome(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"badhome/s.sh": hazardLine,
		"okhome/s.sh":  "echo benign",
		"a/s.sh":       hazardLine,
		"b/s.sh":       "echo benign",
		"calm/s.sh":    "echo benign",
	})
	run := func(home string, cases []scriptCase) {
		t.Helper()
		r := readingRegistry(dir, dir+"/"+home)
		for _, c := range cases {
			cmd := strings.ReplaceAll(c.cmd, "{d}", dir)
			d, err := r.Check(cmd)
			if err != nil {
				t.Errorf("Check(%q): %v", c.cmd, err)
				continue
			}
			if d.Verdict != c.want || d.EntryID != c.id {
				t.Errorf("HOME=%s: %s\n  = %s/%s, want %s/%s\n  %s", home, c.cmd, d.Verdict, d.EntryID, c.want, c.id, d.Message)
			}
		}
	}
	// No stack the line fixed: the directory is unknown, never HOME, whose
	// script would block.
	run("badhome", []scriptCase{
		{`cd {d}/b && pushd && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/b && pushd -q && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/b && pushd -- && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/b && pushd +1 && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/b && pushd -1 && bash s.sh`, VerdictAllow, ""},
		// A pushd the chain does not follow, or a stack a popd or dirs
		// changed, fixes nothing.
		{`cd {d}/calm && pushd {d}/b; pushd && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/calm && pushd {d}/b && popd && pushd && bash s.sh`, VerdictAllow, ""},
		{`cd {d}/calm && pushd {d}/b && dirs -c && pushd && bash s.sh`, VerdictAllow, ""},
		// cd still goes HOME.
		{`cd {d}/b && cd && bash s.sh`, VerdictBlock, scriptHazardEntryID},
	})
	// The line pushed the directory it left: a bare pushd swaps back to it,
	// wherever HOME is.
	run("okhome", []scriptCase{
		{`cd {d}/a && pushd {d}/b && pushd && bash s.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/a && pushd ../b && pushd && bash s.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/b && pushd {d}/a && pushd && pushd && bash s.sh`, VerdictBlock, scriptHazardEntryID},
		// The control: the directory swapped back to is benign.
		{`cd {d}/calm && pushd {d}/b && pushd && bash s.sh`, VerdictAllow, ""},
	})
}
