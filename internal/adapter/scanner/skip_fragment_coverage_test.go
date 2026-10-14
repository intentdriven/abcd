package scanner

import (
	"strings"
	"testing"
)

// publicIPv4 spells a public, non-reserved IPv4 address at runtime, so the
// source carries no literal address.
func publicIPv4() string { return strings.Join([]string{"8", "8", "8", "8"}, ".") }

// iss-2610090821506490: a skip fragment sent every path it matched to the
// byte-only branch, which drops the identity and network rules, and that file
// was never counted as unscanned. A fragment of "." matched every dotted path,
// so an address in an included markdown file shipped while an undotted
// LICENSE kept the zero-coverage sentinel quiet. A file a skip fragment sends
// to byte-only scanning is not scanned: it is Unscanned with its reason.
func TestSkipFragmentFileIsUnscanned(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/pii.json", `{"skip_path_fragments": ["."]}`)
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
		t.Fatalf("a file a skip fragment sends to byte-only scanning must be unscanned: %+v", res)
	}
	if why := res.UnscannedWhy["commands/a.md"]; !strings.Contains(why, "skip fragment") {
		t.Errorf("the reason must name the skip fragment, got %q", why)
	}
	if contains(res.ContentUnverified, "commands/a.md") || contains(res.ScannedBinary, "commands/a.md") {
		t.Errorf("the file must not also be reported byte-scanned: %+v", res)
	}
	if res.FilesScanned != 1 {
		t.Errorf("LICENSE must still be scanned in full: %+v", res)
	}
}

// The reviewed binary list still holds under a fragment: a file whose
// extension is a skip extension takes the byte branch as before, and a file
// no fragment matches is scanned in full.
func TestSkipFragmentLeavesBinaryExtensionsAndOtherPathsAlone(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/pii.json", `{"skip_path_fragments": ["assets/"]}`)
	png := writeFile(t, root, "assets/logo.png", "\x89PNG\r\n\x1a\nnot really an image")
	doc := writeFile(t, root, "commands/a.md", "clean content\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := sc.ScanBundle([]BundleFile{
		{LogicalPath: "assets/logo.png", ResolvedPath: png},
		{LogicalPath: "commands/a.md", ResolvedPath: doc},
	})
	if err != nil {
		t.Fatal(err)
	}
	if contains(res.Unscanned, "assets/logo.png") {
		t.Fatalf("a skip-extension file under a fragment must still take the byte branch: %+v", res)
	}
	if !contains(res.ContentUnverified, "assets/logo.png") && !contains(res.ContentDecoded, "assets/logo.png") {
		t.Errorf("the image must be byte-scanned: %+v", res)
	}
	if res.FilesScanned != 1 || len(res.Unscanned) != 0 {
		t.Errorf("the markdown file must be scanned in full: %+v", res)
	}
}

// An exclusion the technical facilitator declares, with its reason, is its own
// category: never counted as scanned, never a coverage gap, and the reason
// travels with the path.
func TestDeclaredExclusionIsReportedByChoice(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/pii.json",
		`{"exclude_path_fragments": [{"fragment": "testdata/vectors/", "reason": "published third-party test vectors"}]}`)
	blob := writeFile(t, root, "testdata/vectors/v1.bin", "\x00\x01opaque\x00")
	doc := writeFile(t, root, "commands/a.md", "clean content\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := sc.ScanBundle([]BundleFile{
		{LogicalPath: "testdata/vectors/v1.bin", ResolvedPath: blob},
		{LogicalPath: "commands/a.md", ResolvedPath: doc},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(res.Excluded, "testdata/vectors/v1.bin") {
		t.Fatalf("the declared exclusion must be reported excluded: %+v", res)
	}
	if got := res.ExcludedWhy["testdata/vectors/v1.bin"]; got != "published third-party test vectors" {
		t.Errorf("the exclusion's reason must travel with the path, got %q", got)
	}
	if contains(res.Unscanned, "testdata/vectors/v1.bin") || contains(res.ContentUnverified, "testdata/vectors/v1.bin") {
		t.Errorf("an excluded file is in no other category: %+v", res)
	}
	if res.FilesScanned != 1 || res.Unavailable {
		t.Errorf("only the markdown file is scanned, and the scan stands: %+v", res)
	}
}

// An exclusion that leaves no file scanned in full still trips the
// zero-coverage sentinel.
func TestExclusionOfEveryFileStillRefuses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/pii.json",
		`{"exclude_path_fragments": [{"fragment": "commands/", "reason": "generated"}]}`)
	doc := writeFile(t, root, "commands/a.md", "clean content\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := sc.ScanBundle([]BundleFile{{LogicalPath: "commands/a.md", ResolvedPath: doc}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Unavailable || !strings.Contains(res.UnavailableReason, "excluded") {
		t.Fatalf("an exclusion that leaves nothing scanned must refuse, naming the exclusion: %+v", res)
	}
}

// An exclusion without a reason, or whose fragment matches every path, is a
// config fault: the scanner fails closed.
func TestExclusionWithoutAReasonIsAConfigFault(t *testing.T) {
	for name, cfg := range map[string]string{
		"no reason":      `{"exclude_path_fragments": [{"fragment": "testdata/"}]}`,
		"blank reason":   `{"exclude_path_fragments": [{"fragment": "testdata/", "reason": "  "}]}`,
		"blank fragment": `{"exclude_path_fragments": [{"fragment": "/", "reason": "everything"}]}`,
		// A fragment of punctuation alone (".") matches nearly every path, so
		// it names no part of the tree a reviewer can weigh.
		"punctuation fragment": `{"exclude_path_fragments": [{"fragment": ".", "reason": "everything dotted"}]}`,
		"dash fragment":        `{"exclude_path_fragments": [{"fragment": "-_.", "reason": "everything"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, ".abcd/config/pii.json", cfg)
			sc, err := New(root)
			if err != nil {
				t.Fatal(err)
			}
			if bad, why := sc.Unavailable(); !bad || !strings.Contains(why, "exclude_path_fragments") {
				t.Fatalf("the config must fail closed naming the field, got %v %q", bad, why)
			}
		})
	}
}

// iss-2610111600051340: a single-character exclusion fragment is a config
// fault, like a punctuation-only one.
func TestSingleCharacterExclusionIsAConfigFault(t *testing.T) {
	for name, frag := range map[string]string{"letter": "e", "digit": "7", "letter with slash": "e/"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, ".abcd/config/pii.json",
				`{"exclude_path_fragments": [{"fragment": "`+frag+`", "reason": "vendored"}]}`)
			sc, err := New(root)
			if err != nil {
				t.Fatal(err)
			}
			if bad, why := sc.Unavailable(); !bad || !strings.Contains(why, "exclude_path_fragments") {
				t.Fatalf("a single-character fragment must fail the config closed naming the field, got %v %q", bad, why)
			}
		})
	}
}

// An exclusion names whole path segments: a fragment that is part of a
// segment's name excludes nothing, so the secret in docs/secret.md is still
// read and hard-fails. A directory fragment excludes only that directory,
// never a sibling whose name merely ends with it, and an extension-shaped
// fragment matches the end of the file name only.
func TestExclusionMatchesWholePathSegments(t *testing.T) {
	token := "ghp_" + strings.Repeat("a", 36)
	cases := []struct {
		name     string
		fragment string
		excluded []string
		scanned  []string
	}{
		{"part of a name", "se", nil, []string{"docs/secret.md"}},
		{"directory prefix", "generated/", []string{"generated/a.md", "sub/generated/b.md"}, []string{"regenerated/secret.md"}},
		{"nested directory", "testdata/vectors/", []string{"testdata/vectors/v1.md"}, []string{"testdata/vectorsx/secret.md", "testdata/secret.md"}},
		{"file path", "docs/notes.md", []string{"docs/notes.md"}, []string{"docs/notes.md.d/secret.md", "olddocs/notes.md"}},
		{"extension", ".jar", []string{"tools/wrapper.jar"}, []string{"lib.jar.d/secret.md"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, ".abcd/config/pii.json",
				`{"exclude_path_fragments": [{"fragment": "`+tc.fragment+`", "reason": "declared"}]}`)
			var files []BundleFile
			for _, p := range tc.excluded {
				files = append(files, BundleFile{LogicalPath: p, ResolvedPath: writeFile(t, root, p, "clean content\n")})
			}
			for _, p := range tc.scanned {
				files = append(files, BundleFile{LogicalPath: p, ResolvedPath: writeFile(t, root, p, "token "+token+"\n")})
			}
			sc, err := New(root)
			if err != nil {
				t.Fatal(err)
			}
			if bad, why := sc.Unavailable(); bad {
				t.Fatalf("the fragment %q must be accepted: %s", tc.fragment, why)
			}
			res, err := sc.ScanBundle(files)
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range tc.excluded {
				if !contains(res.Excluded, p) {
					t.Errorf("%q must exclude %s: %+v", tc.fragment, p, res.Excluded)
				}
			}
			for _, p := range tc.scanned {
				if contains(res.Excluded, p) {
					t.Errorf("%q must not exclude %s", tc.fragment, p)
				}
			}
			if len(res.Excluded) != len(tc.excluded) {
				t.Errorf("excluded-by-choice count = %d, want %d: %v", len(res.Excluded), len(tc.excluded), res.Excluded)
			}
			if res.HardFails != len(tc.scanned) || res.Unavailable {
				t.Errorf("every secret outside the exclusion must hard-fail: hard fails %d, want %d (unavailable %v)", res.HardFails, len(tc.scanned), res.Unavailable)
			}
		})
	}
}
