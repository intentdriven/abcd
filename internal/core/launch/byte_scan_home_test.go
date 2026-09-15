package launch

import "testing"

// TestBytesAndTextAgreeOnAnAlnumPrecededHome pins the byte scan to the text
// scan for the operator's home path: the same bytes, the home preceded by an
// alphanumeric byte with no separator (how a raw blob usually carries it),
// must hard-fail whether the file is deck.md or deck.png. On the merged tree
// the anchored home_path_self declined the home at its leading byte and the
// byte policy then dropped local_username, so the .png published the
// operator's absolute home path while the .md refused.
func TestBytesAndTextAgreeOnAnAlnumPrecededHome(t *testing.T) {
	// /root is the shape the text side catches only through local_username,
	// which the byte policy drops; the two-segment home is the shape the
	// leading anchor already waives.
	for _, home := range []string{"/root", "/Users/zzhomeuser42"} { // abcd-audit:allow
		t.Run(home, func(t *testing.T) {
			t.Setenv("HOME", home)
			root := t.TempDir()
			writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": ["docs"]}`)
			// Plain bytes, so the .md is text-scanned and the .png (skip-listed
			// by extension) is byte-scanned over the same content; the home is
			// preceded by the '0' of a metadata key with no separator.
			payload := "tEXtCreator0" + home + "/deck.key\n"
			writeFile(t, root, "docs/deck.md", payload)
			writeFile(t, root, "docs/deck.png", payload)

			report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.0.0"})
			if err != nil {
				t.Fatalf("dry-run must return nil error on a finding, got %v", err)
			}
			byFile := map[string]int{}
			for _, f := range report.Scan.Findings {
				byFile[f.File]++
			}
			for _, name := range []string{"docs/deck.md", "docs/deck.png"} {
				if byFile[name] == 0 {
					t.Errorf("%s: the operator's home preceded by an alphanumeric byte was not reported (findings=%v)", name, byFile)
				}
			}
			if report.Scan.HardFails < 2 {
				t.Errorf("want a hard-fail on both files, got %d: %+v", report.Scan.HardFails, report.Scan.Findings)
			}
		})
	}
}
