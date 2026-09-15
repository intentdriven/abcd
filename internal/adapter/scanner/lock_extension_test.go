package scanner

import (
	"strings"
	"testing"
)

// TestLockFileIsScanned — iss-2608270655490511. A `.lock` file is text (the
// leading-comment note on defaultSkipExtensions already flags it as the same
// class as the closed `.svg` gap): a committed lockfile can carry a secret in a
// path, a URL or a comment. While `.lock` sat in the binary skip set, such a
// file shipped UNSCANNED in a launch bundle — a fail-open coverage hole. It must
// now be scanned like any other text file and its secret caught.
func TestLockFileIsScanned(t *testing.T) {
	root := t.TempDir()
	secret := "ghp_" + strings.Repeat("z", 40)
	abs := writeFile(t, root, "deps.lock", "resolved = \"https://x/"+secret+"\"\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, _ := sc.ScanBundle([]BundleFile{{LogicalPath: "deps.lock", ResolvedPath: abs}})
	if res.FilesScanned != 1 {
		t.Fatalf("a .lock text file must be scanned, not skipped: %+v", res)
	}
	if res.HardFails == 0 {
		t.Fatalf("a secret in a .lock file must be caught: %+v", res)
	}
	for _, s := range res.ScannedBinary {
		if s == "deps.lock" {
			t.Fatalf(".lock must no longer be an extension skip: %+v", res.ScannedBinary)
		}
	}
}
