package launch

import (
	"path/filepath"
	"strings"
	"testing"
)

// writeLockstepTree writes a version-location contract, a primary plugin.json
// and a marketplace.json with the given version values (empty string == omit the
// key entirely, to exercise the absent-vs-null distinction).
func writeLockstepTree(t *testing.T, root, primaryVer, marketVer, changelogVer string) {
	t.Helper()
	writeFile(t, root, ".abcd/config/version-location.json",
		`{"manifest_path": ".claude-plugin/plugin.json", "json_pointer": "/version"}`)
	if primaryVer == "" {
		writeFile(t, root, ".claude-plugin/plugin.json", `{"name": "abcd"}`)
	} else {
		writeFile(t, root, ".claude-plugin/plugin.json", `{"name": "abcd", "version": "`+primaryVer+`"}`)
	}
	// source is not part of the lockstep contract, but a listing without one is
	// not a well-formed marketplace entry — the installability smoke reads the
	// same manifest, so the fixture carries the canonical adr-28 "./".
	mk := `{"plugins": [{"name": "abcd", "source": "./"`
	if marketVer != "" {
		mk += `, "version": "` + marketVer + `"`
	}
	if changelogVer != "" {
		mk += `, "changelog": {"version": "` + changelogVer + `"}`
	}
	mk += `}]}`
	writeFile(t, root, ".claude-plugin/marketplace.json", mk)
}

func TestLockstepAgreement(t *testing.T) {
	root := t.TempDir()
	writeLockstepTree(t, root, "1.2.3", "1.2.3", "1.2.3")
	res := CheckLockstep(TreePublic, root, root)
	if !res.OK || res.ExitCode != 0 {
		t.Errorf("expected agreement OK/0, got %+v", res)
	}
}

func TestLockstepDrift(t *testing.T) {
	root := t.TempDir()
	writeLockstepTree(t, root, "1.2.3", "1.2.4", "1.2.3")
	res := CheckLockstep(TreePublic, root, root)
	if res.OK || res.ExitCode != 1 || len(res.Drifts) == 0 {
		t.Errorf("expected drift/1, got %+v", res)
	}
}

func TestLockstepNonSemverIsDrift(t *testing.T) {
	root := t.TempDir()
	writeLockstepTree(t, root, "1.2", "1.2", "1.2")
	res := CheckLockstep(TreePublic, root, root)
	if res.OK || res.ExitCode != 1 {
		t.Errorf("expected non-semver primary to drift, got %+v", res)
	}
}

func TestLockstepBlockedContractUnreadable(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/version-location.json", `{"blocked": true}`)
	writeFile(t, root, ".claude-plugin/plugin.json", `{}`)
	writeFile(t, root, ".claude-plugin/marketplace.json", `{"plugins":[{}]}`)
	res := CheckLockstep(TreePublic, root, root)
	if !res.Unreadable || res.ExitCode != 2 {
		t.Errorf("expected blocked contract → unreadable/2, got %+v", res)
	}
}

func TestLockstepDevKeysAbsent(t *testing.T) {
	root := t.TempDir()
	// Dev tree: no version keys anywhere.
	writeLockstepTree(t, root, "", "", "")
	res := CheckLockstep(TreeDev, root, root)
	if !res.OK || res.ExitCode != 0 {
		t.Errorf("dev tree with absent keys must be OK, got %+v", res)
	}
	// A present version key in the dev tree is drift.
	writeLockstepTree(t, root, "1.2.3", "", "")
	res2 := CheckLockstep(TreeDev, root, root)
	if res2.OK || res2.ExitCode != 1 {
		t.Errorf("dev tree with a present version key must drift, got %+v", res2)
	}
}

// TestResolvePointerOverflowIndex guards B20: a long all-digit pointer token used
// to overflow atoiIndex to a negative int, which passed the `idx >= len(c)` bound
// and panicked on c[idx]. It must now resolve as an absent key, not panic.
func TestResolvePointerOverflowIndex(t *testing.T) {
	doc := []any{"a"}
	for _, tok := range []string{
		"/9223372036854775808",           // 2^63: overflows int64 to negative
		"/99999999999999999999999999999", // far past any int
	} {
		v, present := resolvePointer(doc, tok)
		if present {
			t.Errorf("resolvePointer(%q) resolved present (v=%v); want absent", tok, v)
		}
	}
	// A valid in-range index still resolves.
	if v, present := resolvePointer(doc, "/0"); !present || v != "a" {
		t.Errorf("resolvePointer(/0)=(%v,%v); want (a,true)", v, present)
	}
}

// TestValidateVersionLocationContainsManifestPath proves manifest_path is held to
// the same lexical containment launch include patterns are (gh-488): a ".."
// traversal or an absolute path is refused, so it can never be joined onto a
// trusted root and walked outside it. A well-formed repo-relative path still
// passes.
func TestValidateVersionLocationContainsManifestPath(t *testing.T) {
	uncontained := []string{
		"../outside.json",
		"../../etc/passwd.json",
		".claude-plugin/../../escape.json",
		"/etc/evil.json",
		"a/../../b.json",
	}
	for _, mp := range uncontained {
		decision := map[string]any{"manifest_path": mp, "json_pointer": "/version"}
		_, _, verr := validateVersionLocation(decision)
		if verr == "" {
			t.Errorf("validateVersionLocation accepted an uncontained manifest_path %q; want refusal", mp)
		}
	}
	// A normal in-tree manifest_path is still accepted.
	ok := map[string]any{"manifest_path": ".claude-plugin/plugin.json", "json_pointer": "/version"}
	if gotMP, gotPtr, verr := validateVersionLocation(ok); verr != "" || gotMP != ".claude-plugin/plugin.json" || gotPtr != "/version" {
		t.Errorf("validateVersionLocation rejected a valid contract: mp=%q ptr=%q verr=%q", gotMP, gotPtr, verr)
	}
}

// TestLockstepRefusesUncontainedManifestPath proves the containment holds through
// the CheckLockstep consumer, which joins manifest_path onto repoRoot and reads
// it. A valid manifest sits OUTSIDE the tree (a sibling of root): without the
// guard CheckLockstep would happily read and version-check that arbitrary file;
// with it, the path is refused lexically as unreadable/2 (gh-488).
func TestLockstepRefusesUncontainedManifestPath(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "repo")
	// A perfectly readable, well-formed manifest outside the repo tree.
	writeFile(t, parent, "outside.json", `{"version": "1.2.3"}`)
	writeFile(t, root, ".abcd/config/version-location.json",
		`{"manifest_path": "../outside.json", "json_pointer": "/version"}`)
	writeFile(t, root, ".claude-plugin/marketplace.json",
		`{"plugins":[{"name":"abcd","source":"./","version":"1.2.3","changelog":{"version":"1.2.3"}}]}`)
	res := CheckLockstep(TreePublic, root, root)
	if !res.Unreadable || res.ExitCode != 2 {
		t.Fatalf("expected an uncontained manifest_path to be refused (unreadable/2), got %+v", res)
	}
}

// TestLockstepUnreadableDetailHasNoAbsolutePath proves an unreadable pinned input
// yields a path-free detail: the dry-run report carrying it is a success envelope
// that never passes through cli.scrubPaths, so an embedded os.PathError absolute
// path would leak the developer-identity root into machine output (iss-76/iss-29).
func TestLockstepUnreadableDetailHasNoAbsolutePath(t *testing.T) {
	root := t.TempDir()
	// version-location.json is absent, so loadJSON returns an os.ReadFile error.
	res := CheckLockstep(TreePublic, root, root)
	if !res.Unreadable || res.ExitCode != 2 {
		t.Fatalf("expected unreadable/2, got %+v", res)
	}
	if strings.Contains(res.Detail, root) || strings.Contains(res.Detail, string(filepath.Separator)+".abcd") {
		t.Errorf("detail leaks an absolute path: %q", res.Detail)
	}
	// The cause is still named, just without the path.
	if !strings.Contains(res.Detail, "version-location.json not readable") {
		t.Errorf("detail dropped the cause: %q", res.Detail)
	}
}
