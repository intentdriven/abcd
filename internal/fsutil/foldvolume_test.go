package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFoldPathFoldsWhatThisVolumeFolds: every spelling this volume resolves to a
// real scratch file shares that file's comparison key. A variant the volume keeps
// distinct says nothing, so the test holds on any file system.
func TestFoldPathFoldsWhatThisVolumeFolds(t *testing.T) {
	for _, c := range []struct {
		name     string
		variants []string
	}{
		{"stop.json", []string{"STOP.JSON", "Stop.json", "\u017ftop.json", "\ufb06op.json", "\ufb05op.json", "s\u200dtop.json", "\uff53top.json", "stop.jso\u0274"}},
		{"shared-folder.json", []string{"\u017fhared-folder.json", "SHARED-FOLDER.JSON", "shared-\ufb00older.json"}},
		{"caf\u00e9.json", []string{"cafe\u0301.json", "CAF\u00c9.JSON", "CAFE\u0301.JSON"}},
		{"k.json", []string{"\u212a.json", "K.JSON"}},
		{"strasse.json", []string{"stra\u00dfe.json", "STRASSE.JSON"}},
		{"\u00c5.json", []string{"\u212b.json", "A\u030a.json", "\u00e5.json"}},
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, c.name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		key := FoldPath(c.name, true)
		for _, v := range c.variants {
			if _, err := os.Lstat(filepath.Join(dir, v)); err != nil {
				t.Logf("%q is not %q on this volume", v, c.name)
				continue
			}
			t.Logf("%q is %q on this volume", v, c.name)
			if got := FoldPath(v, true); got != key {
				t.Errorf("this volume resolves %q to %q, but their keys are %q and %q", v, c.name, got, key)
			}
		}
	}
}

// TestFoldPathFoldsByUnicodeRule: a case fold, then a normal form, on any host.
func TestFoldPathFoldsByUnicodeRule(t *testing.T) {
	for _, c := range [][2]string{
		{"\u017ftop.json", "stop.json"},
		{"\ufb06op.json", "stop.json"},
		{"\u212a", "k"},
		{"STOP.JSON", "stop.json"},
		{"cafe\u0301", "caf\u00e9"},
		{"CAF\u00c9", "caf\u00e9"},
	} {
		if a, b := FoldPath(c[0], true), FoldPath(c[1], true); a != b {
			t.Errorf("FoldPath(%q) = %q, FoldPath(%q) = %q", c[0], a, c[1], b)
		}
	}
	if got := FoldPath("St\u00d6P", false); got != "St\u00d6P" {
		t.Errorf("FoldPath with fold off changed it: %q", got)
	}
}
