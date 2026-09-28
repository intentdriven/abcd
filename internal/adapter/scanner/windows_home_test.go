package scanner

import (
	"strings"
	"testing"
)

// iss-2609251639261103: home_path_other and home_path_self read no Windows
// home spelling. A transcript from WSL, a pasted PowerShell session or a
// Windows CI log names homes as <drive>:\Users\<name>, and a JSON encoder
// doubles each backslash (and doubles them again for JSON quoted inside JSON).
// The paths are assembled so no committed line carries a literal one.

const winRoot = `:\Users\`

// winPath spells drive + `:\Users\` + rest with every separator written as
// depth backslashes: 1 as typed, 2 as one JSON layer writes it, 4 as two do.
func winPath(drive, rest string, depth int) string {
	p := drive + winRoot + rest
	return strings.ReplaceAll(p, `\`, strings.Repeat(`\`, depth))
}

func TestWindowsHomeIsAThirdPartyHomePath(t *testing.T) {
	other := "zqwinother"
	cases := []struct {
		name, line string
	}{
		{"as typed", "open " + winPath("C", other+`\Desktop\a.txt`, 1) + " now"},
		{"one JSON layer", `{"t":"open ` + winPath("C", other+`\Desktop`, 2) + `"}`},
		{"two JSON layers", `{"t":"{\"p\":\"` + winPath("C", other+`\Desktop`, 4) + `\"}"}`},
		{"lower-case drive and root", "open " + strings.ToLower(winPath("c", "", 1)) + other + `\x`},
		{"another drive", "open " + winPath("D", other+`\x`, 1)},
		{"at line end", "home is " + winPath("C", other, 1)},
		{"out of the Public root by traversal", "open " + winPath("C", `Public\..\`+other+`\x`, 1)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fs := ScanText(c.line, Identity{}, DefaultPatterns(), nil, "t")
			f, ok := findingOf(fs, kindHomeOther)
			if !ok {
				t.Fatalf("no %s for %q: %+v", kindHomeOther, c.line, fs)
			}
			if !strings.Contains(f.Matched, other) {
				t.Errorf("the %s span %q does not cover the name", kindHomeOther, f.Matched)
			}
			red, _ := Redact(c.line, fs)
			if strings.Contains(red, other) {
				t.Errorf("the name survived redaction:\n%s", red)
			}
		})
	}
}

// TestWindowsSystemRootsAreNotHomes is the false-positive side: the Windows
// profile directories that name no user, at any escaping depth, a relative
// path that merely contains a Users segment, and a backslash run standing
// for ONE separator rather than an empty traversal segment.
func TestWindowsSystemRootsAreNotHomes(t *testing.T) {
	lines := []string{
		"open " + winPath("C", `Public\Documents\a.txt`, 1),
		"open " + winPath("C", `Default\AppData\Local`, 1),
		`{"t":"` + winPath("C", `Public\Desktop`, 2) + `"}`,
		`{"t":"` + winPath("C", `Default\NTUSER.DAT`, 4) + `"}`,
		`see src\Users\guide.md and build` + `\Users\list.txt`,
	}
	for _, line := range lines {
		if fs := ScanText(line, Identity{}, DefaultPatterns(), nil, "t"); hasKind(fs, kindHomeOther) {
			t.Errorf("a non-home was reported as %s: %q\n%+v", kindHomeOther, line, fs)
		}
	}
}

// TestWindowsOwnHomeIsHomeSelfAtAnyDepth: the caller's own home in its
// Windows spelling is home_path_self (hard_fail) wherever it is escaped, and
// is never also reported as a third party's.
func TestWindowsOwnHomeIsHomeSelfAtAnyDepth(t *testing.T) {
	id := Identity{HomePath: "C" + winRoot + "zqwinme", HomeUser: "zqwinme"}
	for _, depth := range []int{1, 2, 4} {
		line := "saved " + winPath("C", `zqwinme\notes.txt`, depth) + " ok"
		fs := ScanText(line, id, DefaultPatterns(), nil, "t")
		if !hasKind(fs, kindHomeSelf) {
			t.Errorf("depth %d: the caller's own Windows home raised no %s: %+v", depth, kindHomeSelf, fs)
		}
		if hasKind(fs, kindHomeOther) {
			t.Errorf("depth %d: the caller's own Windows home was also reported as a third party's: %+v", depth, fs)
		}
		red, _ := Redact(line, fs)
		if strings.Contains(red, "zqwinme") || !strings.Contains(red, "notes.txt") {
			t.Errorf("depth %d: redaction left the name or took the file:\n%s", depth, red)
		}
	}
}
