package afterupdate

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// privateFullSHAPattern finds a hex class sized to a full object name (forty
// digits for SHA-1, sixty-four for SHA-256) written into a source file.
var privateFullSHAPattern = regexp.MustCompile(`\[0-9a-f\]\{(?:40|64)\}`)

// TestNoPrivateFullSHAPattern keeps the store key's check on the one
// canonical primitive: a root commit is keyed only when gitutil.IsFullSHA
// admits it, so no file of this package declares a full-SHA pattern of its
// own that could drift from it (iss-2610231057586567).
func TestNoPrivateFullSHAPattern(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	scanned := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		if loc := privateFullSHAPattern.FindIndex(src); loc != nil {
			line := 1 + strings.Count(string(src[:loc[0]]), "\n")
			t.Errorf("%s:%d declares its own full-SHA pattern %q; call gitutil.IsFullSHA instead", f, line, src[loc[0]:loc[1]])
		}
	}
	if scanned == 0 {
		t.Fatal("no source file scanned: the detector would pass on nothing")
	}
}
