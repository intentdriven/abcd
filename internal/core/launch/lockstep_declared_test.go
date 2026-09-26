package launch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// itd-2609150819432059 AC2: a non-plugin kind's lockstep check reads the primary
// from version-location.json and every declared file, refuses a declared path it
// cannot read, and reads no plugin manifest.

func declaredLockstepRepo(t *testing.T, primary, secondary string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, versionLocationRelPath, `{"outcome":"accept","blocked":false,"manifest_path":"version.json","json_pointer":"/version"}`)
	writeFile(t, root, "version.json", primary)
	writeFile(t, root, "app/meta.json", secondary)
	return root
}

func TestDeclaredLockstepDevPassesWithEveryKeyAbsentAndReadsNoPluginManifest(t *testing.T) {
	root := declaredLockstepRepo(t, `{"name":"app"}`, `{"release":{}}`)
	// No .claude-plugin/ exists at all: a check that read the marketplace
	// manifest would report it unreadable.
	res := CheckDeclaredLockstep(TreeDev, root, filepath.Join(root, versionLocationRelPath),
		[]LockstepFile{{Path: "app/meta.json", Pointer: "/release/version"}})
	if !res.OK || res.ExitCode != 0 {
		t.Fatalf("result %+v, want OK", res)
	}
	if strings.Contains(res.Detail, "marketplace") {
		t.Errorf("the declared check mentions the plugin manifest: %+v", res)
	}
}

func TestDeclaredLockstepDevReportsAPresentVersionKeyAsDrift(t *testing.T) {
	// The secondary names no pointer of its own, so it is read at the primary's.
	root := declaredLockstepRepo(t, `{"name":"app"}`, `{"version":"1.2.3"}`)
	res := CheckDeclaredLockstep(TreeDev, root, filepath.Join(root, versionLocationRelPath),
		[]LockstepFile{{Path: "app/meta.json"}})
	if res.OK || res.ExitCode != 1 || len(res.Drifts) != 1 || !strings.Contains(res.Drifts[0], "app/meta.json/version") {
		t.Fatalf("result %+v, want one dev drift at app/meta.json/version", res)
	}
}

func TestDeclaredLockstepRefusesADeclaredPathItCannotRead(t *testing.T) {
	root := declaredLockstepRepo(t, `{"name":"app"}`, `{}`)
	res := CheckDeclaredLockstep(TreeDev, root, filepath.Join(root, versionLocationRelPath),
		[]LockstepFile{{Path: "app/meta.json"}, {Path: "missing/info.json"}})
	if !res.Unreadable || res.ExitCode != 2 || !strings.Contains(res.Detail, "missing/info.json") {
		t.Fatalf("result %+v, want unreadable naming missing/info.json", res)
	}
	if strings.Contains(res.Detail, root) {
		t.Errorf("the detail carries an absolute path: %s", res.Detail)
	}
}

func TestDeclaredLockstepPublicRequiresEverySecondaryToAgree(t *testing.T) {
	root := declaredLockstepRepo(t, `{"version":"1.2.3"}`, `{"release":{"version":"1.2.0"}}`)
	vl := filepath.Join(root, versionLocationRelPath)
	files := []LockstepFile{{Path: "app/meta.json", Pointer: "/release/version"}}
	res := CheckDeclaredLockstep(TreePublic, root, vl, files)
	if res.OK || len(res.Drifts) != 1 || !strings.Contains(res.Drifts[0], `expected "1.2.3"`) {
		t.Fatalf("result %+v, want one public drift naming the primary's version", res)
	}
	writeFile(t, root, "app/meta.json", `{"release":{"version":"1.2.3"}}`)
	if res := CheckDeclaredLockstep(TreePublic, root, vl, files); !res.OK {
		t.Fatalf("agreeing secondaries: %+v", res)
	}
}

// A non-plugin kind that declares no lockstep list and has no version-location
// contract holds nothing in lockstep, and the check says so rather than refusing.
func TestDeclaredLockstepWithNothingDeclaredIsAnHonestPass(t *testing.T) {
	root := t.TempDir()
	res := CheckDeclaredLockstep(TreeDev, root, filepath.Join(root, versionLocationRelPath), nil)
	if !res.OK || !strings.Contains(res.Detail, "nothing is held in lockstep") {
		t.Fatalf("result %+v, want an OK naming that nothing is held in lockstep", res)
	}
	// A declared list with no primary to agree with is a contract nobody can check.
	res = CheckDeclaredLockstep(TreeDev, root, filepath.Join(root, versionLocationRelPath), []LockstepFile{{Path: "a.json"}})
	if !res.Unreadable || !strings.Contains(res.Detail, "version-location.json") {
		t.Fatalf("result %+v, want unreadable naming version-location.json", res)
	}
}

// A declared path is contained by what it resolves to, not by its spelling: a
// committed symlink inside the repository that points out of it is refused as
// unreadable, and the value outside is never read. A symlink that stays inside
// the repository reads through.
func TestDeclaredLockstepRefusesASymlinkThatLeavesTheRepository(t *testing.T) {
	outside := t.TempDir()
	writeFile(t, outside, "meta.json", `{"version":"9.9.9"}`)
	root := declaredLockstepRepo(t, `{"version":"1.2.3"}`, `{"version":"1.2.3"}`)
	if err := os.Symlink(filepath.Join(outside, "meta.json"), filepath.Join(root, "app", "escape.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("meta.json", filepath.Join(root, "app", "inside.json")); err != nil {
		t.Fatal(err)
	}
	vl := filepath.Join(root, versionLocationRelPath)

	res := CheckDeclaredLockstep(TreePublic, root, vl, []LockstepFile{{Path: "app/escape.json"}})
	if !res.Unreadable || res.ExitCode != 2 || !strings.Contains(res.Detail, "app/escape.json") {
		t.Fatalf("result %+v, want unreadable naming app/escape.json", res)
	}
	if strings.Contains(res.Detail, "9.9.9") || strings.Contains(res.Detail, outside) {
		t.Errorf("the refusal carries what lies outside the repository: %s", res.Detail)
	}

	res = CheckDeclaredLockstep(TreePublic, root, vl, []LockstepFile{{Path: "app/inside.json"}})
	if !res.OK {
		t.Errorf("a symlink that stays inside the repository must read through: %+v", res)
	}
}

// The primary manifest is held the same way, by both lockstep checks.
func TestLockstepRefusesAPrimaryManifestThatLeavesTheRepository(t *testing.T) {
	outside := t.TempDir()
	writeFile(t, outside, "version.json", `{"version":"1.2.3"}`)
	root := t.TempDir()
	writeFile(t, root, versionLocationRelPath, `{"outcome":"accept","blocked":false,"manifest_path":"version.json","json_pointer":"/version"}`)
	if err := os.Symlink(filepath.Join(outside, "version.json"), filepath.Join(root, "version.json")); err != nil {
		t.Fatal(err)
	}
	vl := filepath.Join(root, versionLocationRelPath)
	for name, res := range map[string]LockstepResult{
		"CheckDeclaredLockstep": CheckDeclaredLockstep(TreePublic, root, vl, nil),
		"CheckLockstep":         CheckLockstep(TreePublic, root, vl),
	} {
		if !res.Unreadable || res.ExitCode != 2 || !strings.Contains(res.Detail, "primary manifest") {
			t.Errorf("%s: result %+v, want the primary manifest unreadable", name, res)
		}
	}
}
