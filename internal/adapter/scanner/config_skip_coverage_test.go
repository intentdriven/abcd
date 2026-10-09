package scanner

import (
	"strings"
	"testing"
)

// Only abcd's bundled binary lists (defaultSkipExtensions,
// defaultSkipFilenames) count as reviewed. An extension or filename a repo
// adds through skip_extensions or skip_filenames sent every file it matched to
// the byte-only branch, which drops the identity and network rules, and the
// file was never counted as a gap: a repo-added ".md" let an address in a
// markdown file ship. Such a file is Unscanned with its reason, the same as a
// file a skip fragment alone matches (iss-2610090821506490).
func TestRepoAddedSkipExtensionIsUnscanned(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/pii.json", `{"skip_extensions": [".md"]}`)
	doc := writeFile(t, root, "commands/a.md", "dns "+publicIPv4()+"\n")
	license := writeFile(t, root, "LICENSE", "clean licence text\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := sc.ScanBundle([]BundleFile{
		{LogicalPath: "commands/a.md", ResolvedPath: doc},
		{LogicalPath: "LICENSE", ResolvedPath: license},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(res.Unscanned, "commands/a.md") {
		t.Fatalf("a file a repo-added skip extension matches must be unscanned: %+v", res)
	}
	if why := res.UnscannedWhy["commands/a.md"]; !strings.Contains(why, "skip_extensions") || !strings.Contains(why, "exclude_path_fragments") {
		t.Errorf("the reason must name skip_extensions and the way to exclude, got %q", why)
	}
	if contains(res.ContentUnverified, "commands/a.md") || contains(res.ScannedBinary, "commands/a.md") || contains(res.ContentDecoded, "commands/a.md") {
		t.Errorf("the file must not also be reported byte-scanned: %+v", res)
	}
	if res.FilesScanned != 1 {
		t.Errorf("LICENSE must still be scanned in full: %+v", res)
	}
}

// A filename a repo adds through skip_filenames is held to the same rule.
func TestRepoAddedSkipFilenameIsUnscanned(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/pii.json", `{"skip_filenames": ["NOTICE"]}`)
	notice := writeFile(t, root, "NOTICE", "clean notice\n")
	doc := writeFile(t, root, "commands/a.md", "clean content\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := sc.ScanBundle([]BundleFile{
		{LogicalPath: "NOTICE", ResolvedPath: notice},
		{LogicalPath: "commands/a.md", ResolvedPath: doc},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(res.Unscanned, "NOTICE") {
		t.Fatalf("a file a repo-added skip filename matches must be unscanned: %+v", res)
	}
	if why := res.UnscannedWhy["NOTICE"]; !strings.Contains(why, "skip_filenames") {
		t.Errorf("the reason must name skip_filenames, got %q", why)
	}
}

// The bundled list still holds: a .png takes the byte branch whatever the
// repo adds, and a repo restating a bundled extension changes nothing.
func TestBundledSkipExtensionStillByteScans(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/pii.json", `{"skip_extensions": [".md", ".PNG"]}`)
	png := writeFile(t, root, "assets/logo.png", "\x89PNG\r\n\x1a\nnot really an image")
	doc := writeFile(t, root, "LICENSE", "clean content\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := sc.ScanBundle([]BundleFile{
		{LogicalPath: "assets/logo.png", ResolvedPath: png},
		{LogicalPath: "LICENSE", ResolvedPath: doc},
	})
	if err != nil {
		t.Fatal(err)
	}
	if contains(res.Unscanned, "assets/logo.png") {
		t.Fatalf("a bundled skip extension must still take the byte branch: %+v", res)
	}
	if !contains(res.ContentUnverified, "assets/logo.png") && !contains(res.ContentDecoded, "assets/logo.png") {
		t.Errorf("the image must be byte-scanned: %+v", res)
	}
}

// A file the technical facilitator excludes by name, with a reason, is
// excluded by choice even when a repo-added skip extension also matches it:
// the exclusion is the declared way to leave it out, and an extension-shaped
// fragment is enough to declare it.
func TestExclusionOverridesRepoAddedSkipExtension(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/pii.json",
		`{"skip_extensions": [".jar"], "exclude_path_fragments": [{"fragment": ".jar", "reason": "vendored build tool, checked upstream"}]}`)
	jar := writeFile(t, root, "tools/wrapper.jar", "PK\x03\x04opaque")
	doc := writeFile(t, root, "commands/a.md", "clean content\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := sc.ScanBundle([]BundleFile{
		{LogicalPath: "tools/wrapper.jar", ResolvedPath: jar},
		{LogicalPath: "commands/a.md", ResolvedPath: doc},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(res.Excluded, "tools/wrapper.jar") || res.ExcludedWhy["tools/wrapper.jar"] != "vendored build tool, checked upstream" {
		t.Fatalf("the declared exclusion must be reported with its reason: %+v", res)
	}
	if contains(res.Unscanned, "tools/wrapper.jar") || contains(res.ContentUnverified, "tools/wrapper.jar") {
		t.Errorf("an excluded file is in no other category: %+v", res)
	}
	if res.FilesScanned != 1 || res.Unavailable {
		t.Errorf("the markdown file is scanned and the scan stands: %+v", res)
	}
}
