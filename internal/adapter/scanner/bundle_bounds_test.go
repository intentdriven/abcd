package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bundle_bounds_test.go — iss-2608291849371769: ScanBundle's text branch read
// a file with an uncapped os.ReadFile while the skip-listed branch was capped;
// .gitignore sat on the default skip filenames, so a text file took the weaker
// byte rules; and findings accumulated uncapped across the whole bundle.

func TestTextBundleFileOverTheCapIsAnUnscannedGap(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "big.md")
	if err := os.WriteFile(abs, []byte(strings.Repeat("plain prose line\n", maxTextScanBytes/17+2)), 0o644); err != nil {
		t.Fatal(err)
	}
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "big.md", abs)
	if !contains(res.Unscanned, "big.md") {
		t.Fatalf("a text file over the scan cap must be a loud Unscanned gap: %+v", res.Unscanned)
	}
	if why := res.UnscannedWhy["big.md"]; !strings.Contains(why, "cap") {
		t.Errorf("the gap does not name the cap: %q", why)
	}
}

func TestGitignoreIsScannedAsText(t *testing.T) {
	root := t.TempDir()
	abs := writeFile(t, root, ".gitignore", "# old checkout at /home/colleague/work\n") // abcd-audit:allow
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, ".gitignore", abs)
	if res.FilesScanned != 1 || contains(res.ScannedBinary, ".gitignore") {
		t.Errorf(".gitignore is text and takes the full rule set: %+v", res)
	}
	if !hasKind(res.Findings, kindHomeOther) {
		t.Errorf("a prose identity rule did not run over .gitignore: %+v", res.Findings)
	}
}

func TestBundleFindingsAreCappedAndHardFailsStillCounted(t *testing.T) {
	root := t.TempDir()
	var files []BundleFile
	perFile := 1000
	nFiles := maxBundleFindings/perFile + 2
	body := strings.Repeat("token="+fakeToken()+"\n", perFile)
	for i := 0; i < nFiles; i++ {
		name := fmt.Sprintf("f%d.md", i)
		files = append(files, BundleFile{LogicalPath: name, ResolvedPath: writeFile(t, root, name, body)})
	}
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := sc.ScanBundle(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) > maxBundleFindings {
		t.Errorf("findings are not capped: %d > %d", len(res.Findings), maxBundleFindings)
	}
	if want := nFiles * perFile; res.HardFails != want || len(res.Findings)+res.FindingsOmitted != want {
		t.Errorf("hard fails %d, kept %d + omitted %d; want every one of %d counted", res.HardFails, len(res.Findings), res.FindingsOmitted, want)
	}
}
