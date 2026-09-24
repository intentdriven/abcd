package launch

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// archiveFixture is renderFixture plus what the pinned archive needs beyond a
// render: a repository URL the release download address derives from, and an
// executable hook script whose mode must survive the round trip.
func archiveFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": [".claude-plugin", "README.md", "scripts"]}`)
	writeFile(t, root, "README.md", "readme\n")
	writeFile(t, root, "scripts/run.sh", "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(filepath.Join(root, "scripts/run.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeLockstepTree(t, root, "", "", "")
	writeFile(t, root, ".claude-plugin/plugin.json",
		`{"name": "abcd", "repository": "https://github.com/example/abcd"}`)
	return root
}

func renderArchiveIn(t *testing.T, root, version string) PluginArchive {
	t.Helper()
	out := t.TempDir()
	a, _, err := RenderPluginArchive(PayloadRenderRequest{
		RepoRoot: root, Dest: filepath.Join(t.TempDir(), "staging"), Version: version, Entry: sampleEntry(),
	}, out)
	if err != nil {
		t.Fatalf("RenderPluginArchive: %v", err)
	}
	return a
}

func zipEntries(t *testing.T, path string) map[string]*zip.File {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open the archive: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	out := map[string]*zip.File{}
	for _, f := range r.File {
		out[f.Name] = f
	}
	return out
}

// TestPluginArchiveIsReproducible is the property the pin rests on: the ship
// renders the archive from its tree and the release workflow renders it again
// from the tagged commit, on another machine, later. Only byte-identical output
// makes the committed digest checkable, so two renders of one tree must agree
// even when the working copy's timestamps differ between them.
func TestPluginArchiveIsReproducible(t *testing.T) {
	root := archiveFixture(t)
	first := renderArchiveIn(t, root, "1.2.3")

	later := time.Date(2031, 1, 2, 3, 4, 5, 0, time.UTC)
	for _, rel := range []string{"README.md", "scripts/run.sh", ".claude-plugin/plugin.json"} {
		if err := os.Chtimes(filepath.Join(root, rel), later, later); err != nil {
			t.Fatal(err)
		}
	}
	second := renderArchiveIn(t, root, "1.2.3")

	if first.SHA256 != second.SHA256 {
		t.Fatalf("two renders of one tree differ: %s vs %s", first.SHA256, second.SHA256)
	}
	a, _ := os.ReadFile(first.Path)
	b, _ := os.ReadFile(second.Path)
	if !bytes.Equal(a, b) {
		t.Fatal("two renders of one tree are not byte-identical")
	}
	if len(first.SHA256) != 64 || strings.ToLower(first.SHA256) != first.SHA256 {
		t.Errorf("the digest must be 64 lower-case hex characters, got %q", first.SHA256)
	}

	// A different version is a different archive: the stamp is inside it.
	if other := renderArchiveIn(t, root, "1.2.4"); other.SHA256 == first.SHA256 {
		t.Error("a different release version rendered the same digest — the version stamp is not in the archive")
	}
}

// TestPluginArchiveCarriesThePluginAndNotTheCatalog pins the archive's layout:
// the stamped plugin manifest at the top level where the harness looks for it,
// the executable bit intact, and the marketplace catalog LEFT OUT — the catalog
// is what names the archive's digest, so an archive that carried it could never
// match its own pin.
func TestPluginArchiveCarriesThePluginAndNotTheCatalog(t *testing.T) {
	root := archiveFixture(t)
	a := renderArchiveIn(t, root, "1.2.3")

	if a.Name != "abcd-plugin-v1.2.3.zip" || filepath.Base(a.Path) != a.Name {
		t.Errorf("archive name = %q at %q, want abcd-plugin-v1.2.3.zip", a.Name, a.Path)
	}
	if a.Version != "1.2.3" {
		t.Errorf("archive version = %q, want 1.2.3", a.Version)
	}
	entries := zipEntries(t, a.Path)
	if _, ok := entries[".claude-plugin/marketplace.json"]; ok {
		t.Error("the archive carries marketplace.json — the catalog that pins the archive cannot be inside it")
	}
	plugin, ok := entries[".claude-plugin/plugin.json"]
	if !ok {
		t.Fatalf("the plugin manifest is not at the archive's top level; entries: %v", keys(entries))
	}
	rc, err := plugin.Open()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(rc)
	_ = rc.Close()
	if !strings.Contains(buf.String(), `"version": "1.2.3"`) {
		t.Errorf("the archived plugin.json is not version-stamped:\n%s", buf.String())
	}
	script, ok := entries["scripts/run.sh"]
	if !ok {
		t.Fatal("the archive dropped a payload file")
	}
	if script.Mode().Perm()&0o111 == 0 {
		t.Errorf("the executable bit was lost: %v", script.Mode())
	}
	if readme := entries["README.md"]; readme == nil || readme.Mode().Perm() != 0o644 {
		t.Errorf("a plain file must archive as 0644, got %v", readme)
	}
	if a.Files != len(entries) {
		t.Errorf("reported %d files, archive holds %d", a.Files, len(entries))
	}
}

func keys(m map[string]*zip.File) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestArchiveReleaseURL derives the download address from the plugin
// manifest's own repository field, and refuses anything that is not a plain
// https GitHub repository URL — the address lands in a committed catalog every
// install follows.
func TestArchiveReleaseURL(t *testing.T) {
	root := archiveFixture(t)
	got, err := ArchiveReleaseURL(root, "1.2.3")
	if err != nil {
		t.Fatalf("ArchiveReleaseURL: %v", err)
	}
	want := "https://github.com/example/abcd/releases/download/v1.2.3/abcd-plugin-v1.2.3.zip"
	if got != want {
		t.Errorf("url = %q, want %q", got, want)
	}

	for _, repo := range []string{
		"", "http://github.com/example/abcd", "https://example.com/example/abcd",
		"https://github.com/example", "https://github.com/example/abcd/tree/main",
		"https://github.com/example/abcd?x=1", "https://github.com/../abcd",
	} {
		writeFile(t, root, ".claude-plugin/plugin.json", `{"name": "abcd", "repository": "`+repo+`"}`)
		if _, err := ArchiveReleaseURL(root, "1.2.3"); err == nil {
			t.Errorf("repository %q must be refused", repo)
		}
	}
}

// TestWriteArchivePinRewritesOnlyTheSource is the catalog half of the ship: the
// plugin's listing gains the pinned archive source and nothing else moves — in
// particular no version key, so the working tree keeps adr-19's polarity.
func TestWriteArchivePinRewritesOnlyTheSource(t *testing.T) {
	root := archiveFixture(t)
	writeFile(t, root, ".claude-plugin/marketplace.json",
		`{"name": "m", "plugins": [{"name": "abcd", "source": "./", "description": "d"}]}`)
	pin := ArchivePin{URL: "https://github.com/example/abcd/releases/download/v1.2.3/abcd-plugin-v1.2.3.zip", SHA256: strings.Repeat("ab", 32)}

	if err := WriteArchivePin(root, pin); err != nil {
		t.Fatalf("WriteArchivePin: %v", err)
	}
	got, ok, err := ReadArchivePin(root)
	if err != nil || !ok {
		t.Fatalf("ReadArchivePin after a write: ok=%v err=%v", ok, err)
	}
	if got != pin {
		t.Errorf("pin read back = %+v, want %+v", got, pin)
	}
	market, err := loadJSON(filepath.Join(root, marketplaceFile))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := resolvePointer(market, "/plugins/0/description"); v != "d" {
		t.Errorf("the listing's other fields must survive, description = %v", v)
	}
	if v, _ := resolvePointer(market, "/plugins/0/source/source"); v != "archive" {
		t.Errorf("source kind = %v, want archive", v)
	}
	if res := CheckLockstep(TreeDev, root, filepath.Join(root, versionLocationRelPath)); !res.OK {
		t.Errorf("a pinned catalog must stay version-absent, got %+v", res)
	}

	// The pin is only ever a well-formed one.
	for _, bad := range []ArchivePin{
		{URL: "http://github.com/x.zip", SHA256: pin.SHA256},
		{URL: pin.URL, SHA256: "abc"},
		{URL: pin.URL, SHA256: strings.Repeat("zz", 32)},
	} {
		if err := WriteArchivePin(root, bad); err == nil {
			t.Errorf("a malformed pin %+v must be refused", bad)
		}
	}
}

// TestVerifyArchivePin is the release gate's judgement: a re-rendered archive is
// accepted only when the committed catalog names exactly its address and its
// digest. Every other state — no pin at all, a pin for another release, a
// digest the render does not reproduce — is a refusal.
func TestVerifyArchivePin(t *testing.T) {
	root := archiveFixture(t)
	a := renderArchiveIn(t, root, "1.2.3")
	url, err := ArchiveReleaseURL(root, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}

	if err := VerifyArchivePin(root, a); !errors.Is(err, ErrArchivePinMismatch) {
		t.Errorf("an unpinned catalog (source ./) must refuse, got %v", err)
	}

	if err := WriteArchivePin(root, ArchivePin{URL: url, SHA256: strings.ToUpper(a.SHA256)}); err != nil {
		t.Fatal(err)
	}
	if err := VerifyArchivePin(root, a); err != nil {
		t.Errorf("a matching pin (digest case aside) must verify, got %v", err)
	}

	if err := WriteArchivePin(root, ArchivePin{URL: url, SHA256: strings.Repeat("0", 64)}); err != nil {
		t.Fatal(err)
	}
	err = VerifyArchivePin(root, a)
	if !errors.Is(err, ErrArchivePinMismatch) || !strings.Contains(err.Error(), a.SHA256) {
		t.Errorf("a digest mismatch must refuse and name the rendered digest, got %v", err)
	}

	other := strings.Replace(url, "v1.2.3", "v1.2.2", 2)
	if err := WriteArchivePin(root, ArchivePin{URL: other, SHA256: a.SHA256}); err != nil {
		t.Fatal(err)
	}
	if err := VerifyArchivePin(root, a); !errors.Is(err, ErrArchivePinMismatch) {
		t.Errorf("a pin naming another release must refuse, got %v", err)
	}
}

// TestPrecheckPluginArchiveRefusesBeforeAnyWrite pins the version-free refusals
// the ship makes before its release record lands: a manifest with no usable
// repository address cannot be pinned, so it is refused while nothing has been
// written yet.
func TestPrecheckPluginArchiveRefusesBeforeAnyWrite(t *testing.T) {
	root := archiveFixture(t)
	if err := PrecheckPluginArchive(root); err != nil {
		t.Fatalf("a well-formed plugin must pass the archive precheck: %v", err)
	}
	writeFile(t, root, ".claude-plugin/plugin.json", `{"name": "abcd"}`)
	if err := PrecheckPluginArchive(root); err == nil {
		t.Error("a plugin manifest with no repository must be refused")
	}
}

// TestDeclaresPluginArchive pins the positive declaration a ship pins on: only
// a version-location contract that says `"publishes_plugin_archive": true`
// publishes the archive the catalog would name. The contract alone is not the
// statement — a managed repository scaffolds workflows that upload no archive,
// and a catalog pinned there names an asset every install 404s on.
func TestDeclaresPluginArchive(t *testing.T) {
	cases := []struct {
		name    string
		body    string // "" = no version-location.json at all
		want    bool
		wantErr string
	}{
		{name: "no contract", body: "", want: false},
		{name: "the contract alone", body: `{"manifest_path": ".claude-plugin/plugin.json", "json_pointer": "/version"}`, want: false},
		{name: "declared false", body: `{"manifest_path": "p.json", "json_pointer": "/version", "publishes_plugin_archive": false}`, want: false},
		{name: "declared true", body: `{"manifest_path": "p.json", "json_pointer": "/version", "publishes_plugin_archive": true}`, want: true},
		{name: "not a boolean", body: `{"publishes_plugin_archive": "yes"}`, wantErr: "publishes_plugin_archive"},
		{name: "not an object", body: `[]`, wantErr: "not a JSON object"},
		{name: "not JSON", body: `{`, wantErr: "version-location.json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if tc.body != "" {
				writeFile(t, root, ".abcd/config/version-location.json", tc.body)
			}
			got, err := DeclaresPluginArchive(root)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want one naming %q", err, tc.wantErr)
				}
				if strings.Contains(err.Error(), root) {
					t.Errorf("the refusal leaks the absolute path: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("DeclaresPluginArchive: %v", err)
			}
			if got != tc.want {
				t.Errorf("DeclaresPluginArchive = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestCheckArchiveRepository pins the release workflow's repository binding:
// the archive address derives from plugin.json's repository, which a rename, a
// transfer or a fork leaves naming another repository, so the gate refuses
// unless the address sits under the releasing repository's download path for
// the tag. GitHub resolves owner and repository names case-insensitively.
func TestCheckArchiveRepository(t *testing.T) {
	const url = "https://github.com/Example/ABCD/releases/download/v1.2.3/abcd-plugin-v1.2.3.zip"
	cases := []struct {
		name, repository, tag string
		wantErr               error // nil = passes
		wantText              string
	}{
		{name: "this repository", repository: "Example/ABCD", tag: "v1.2.3"},
		{name: "case-insensitive", repository: "example/abcd", tag: "v1.2.3"},
		{name: "another repository", repository: "example/fork", tag: "v1.2.3", wantErr: ErrArchiveRepositoryMismatch, wantText: "example/fork"},
		{name: "a name the address only starts with", repository: "example/abc", tag: "v1.2.3", wantErr: ErrArchiveRepositoryMismatch},
		{name: "another tag", repository: "example/abcd", tag: "v1.2.4", wantErr: ErrArchiveRepositoryMismatch, wantText: "v1.2.4"},
		{name: "no slash", repository: "abcd", tag: "v1.2.3", wantText: "owner/name"},
		{name: "a path", repository: "example/abcd/extra", tag: "v1.2.3", wantText: "owner/name"},
		{name: "a traversal", repository: "example/..", tag: "v1.2.3", wantText: "owner/name"},
		{name: "empty", repository: "", tag: "v1.2.3", wantText: "owner/name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckArchiveRepository(url, tc.repository, tc.tag)
			if tc.wantErr == nil && tc.wantText == "" {
				if err != nil {
					t.Fatalf("CheckArchiveRepository: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("CheckArchiveRepository passed, want a refusal")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && errors.Is(err, ErrArchiveRepositoryMismatch) {
				t.Errorf("a malformed operand is a structural fault, not a mismatch: %v", err)
			}
			if tc.wantText != "" && !strings.Contains(err.Error(), tc.wantText) {
				t.Errorf("err = %v, want it to name %q", err, tc.wantText)
			}
		})
	}
}
