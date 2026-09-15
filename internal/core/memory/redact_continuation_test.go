package memory

import (
	"strings"
	"testing"
)

// TestStoreRedactorSweepsHomeWithANameSuffix pins the memory store's redactor
// to the three continuation shapes: the caller's home followed by ".zip",
// "-old" and "_snapshot", behind a URL host and as a plain path, must be
// written with the name gone — neither committed verbatim nor refused.
func TestStoreRedactorSweepsHomeWithANameSuffix(t *testing.T) {
	home := "/Users/zzhomeuser42" // abcd-audit:allow
	t.Setenv("HOME", home)
	r, err := newStoreRedactor(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		"the archive is at https://ci.example.com" + home + ".zip for review\n",
		"old copy under " + home + "-old/x here\n",
		"snapshot under /Volumes/T7" + home + "_snapshot/x here\n",
	} {
		out, _, err := r.redactText(text, "page")
		if err != nil {
			t.Errorf("%q: the write was refused: %v", text, err)
			continue
		}
		if strings.Contains(out, "zzhomeuser42") {
			t.Errorf("%q: the caller's name reached the page:\n%s", text, out)
		}
	}
}
