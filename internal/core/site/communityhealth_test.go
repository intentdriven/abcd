package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The forge reads a repository's community-health files — the contribution
// guide and the security policy among them — from `.github/` as readily as from
// the root, and a repository that keeps its root clear moves them there. The
// site follows them: the footer links the security policy where it lives, and
// the contributors page quotes a policy the manifest names in `.github/`
// (iss-2608270540523859).
func TestTheSiteReadsCommunityHealthFilesFromTheGithubDirectory(t *testing.T) {
	f := newFixture(t)
	for _, name := range []string{"SECURITY.md", "CONTRIBUTING.md"} {
		data, err := os.ReadFile(filepath.Join(f.Root(), name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(f.Root(), name)); err != nil {
			t.Fatal(err)
		}
		f.write(".github/"+name, string(data))
	}
	repointManifest(t, f, `"file": "CONTRIBUTING.md"`, `"file": ".github/CONTRIBUTING.md"`)

	out := t.TempDir()
	buildFixture(t, f, out)

	const forge = "https://example.invalid/fixture/repo/blob/HEAD/"
	index := outFile(t, out, "index.html")
	if !strings.Contains(index, `<a href="`+forge+`.github/SECURITY.md">SECURITY.md</a>`) {
		t.Error("the footer does not link the security policy in .github/")
	}
	if strings.Contains(index, forge+"SECURITY.md") {
		t.Error("the footer links a root security policy the repository does not carry")
	}
	contributors := outFile(t, out, "contributors/index.html")
	for _, want := range []string{
		`data-src=".github/CONTRIBUTING.md#attribution"`,
		`<a href="` + forge + `.github/CONTRIBUTING.md">`,
		"Human author of record.",
	} {
		if !strings.Contains(contributors, want) {
			t.Errorf("the contributors page does not carry %q", want)
		}
	}
}

// The footer names the security policy by its file name wherever it lives, and
// the provenance gate reads that name as the file the footer resolved, not as a
// root path the repository no longer carries (iss-2608270540523859).
func TestTheProvenanceGateReadsTheFooterFileItResolved(t *testing.T) {
	f := newFixture(t)
	data, err := os.ReadFile(filepath.Join(f.Root(), "SECURITY.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(f.Root(), "SECURITY.md")); err != nil {
		t.Fatal(err)
	}
	f.write(".github/SECURITY.md", string(data))

	res, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	for _, fd := range res.Findings {
		if strings.Contains(fd.Detail, `"SECURITY.md"`) {
			t.Errorf("the gate refused the footer's security link: %+v", fd)
		}
	}
}
