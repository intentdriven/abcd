package launch

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// itd-2609150819432059: the preview runs against the declared artefact kind.

// AC1: no declaration is a named refusal, never a missing-file error, and the
// preview writes nothing.
func TestDryRunWithNoArtefactDeclarationRefusesNamingItsHome(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": ["commands"]}`)
	writeFile(t, root, "commands/x.md", "doc\n")
	_, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.0.0"})
	if !errors.Is(err, ErrNoArtefact) {
		t.Fatalf("err = %v, want the no-declaration refusal", err)
	}
	if !strings.Contains(err.Error(), ArtefactRelPath) || !strings.Contains(err.Error(), "plugin, binary, application") {
		t.Errorf("the refusal does not name the declaration's home and the kinds: %v", err)
	}
}

// AC9: a kind the binary does not know refuses, naming the kind and the set.
func TestDryRunRefusesAnUnknownKind(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ArtefactRelPath, `{"kind": "wheel"}`)
	_, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.0.0"})
	if err == nil || !strings.Contains(err.Error(), `"wheel"`) || !strings.Contains(err.Error(), "plugin, binary, application") {
		t.Fatalf("err = %v, want a refusal naming the kind and the accepted set", err)
	}
}

// binaryRepo is a managed Go binary: no plugin manifest, no payload include
// config, a declared lockstep file, and a record namespace that must not ship.
func binaryRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(ArtefactRelPath, `{"kind": "binary", "lockstep": ["app/meta.json"]}`)
	r.Write(versionLocationRelPath, `{"outcome":"accept","blocked":false,"manifest_path":"version.json","json_pointer":"/version"}`)
	r.Write("version.json", `{"name":"tool"}`)
	r.Write("app/meta.json", `{"bundle":"tool"}`)
	r.Write("main.go", "package main\n\nfunc main() {}\n")
	r.Write("README.md", "# tool\n")
	r.Write(".abcd/work/CONTEXT.md", "token = "+fakeSecret+"\n")
	r.Commit("a managed binary")
	return r
}

// AC2 + AC8: the declared lockstep is checked with no plugin manifest read and
// no payload include config required, and the scan runs over the archived tree
// minus the record namespace, with the report saying which tree it scanned.
func TestDryRunForABinaryScansTheArchivedTreeAndChecksTheDeclaredLockstep(t *testing.T) {
	r := binaryRepo(t)
	report, err := DryRun(DryRunRequest{RepoRoot: r.Root(), Version: "1.0.0", ExistingTags: []Semver{}})
	if err != nil {
		t.Fatalf("a declared binary with no payload config must preview: %v", err)
	}
	if report.Kind != KindBinary || report.ScannedTree != ArchiveTreeDescription {
		t.Errorf("kind %q, scanned tree %q", report.Kind, report.ScannedTree)
	}
	included := map[string]bool{}
	for _, f := range report.Bundle.Included {
		included[f.LogicalPath] = true
	}
	for _, p := range []string{"main.go", "README.md", "version.json", "app/meta.json"} {
		if !included[p] {
			t.Errorf("%s is in the archived tree but not in the bundle: %v", p, included)
		}
	}
	for p := range included {
		if strings.HasPrefix(p, ".abcd/") {
			t.Errorf("the record namespace reached the bundle: %s", p)
		}
	}
	if report.Scan.HardFails != 0 {
		t.Errorf("the secret inside .abcd/ was scanned as if it shipped: %+v", report.Scan.Findings)
	}
	if !report.Lockstep.OK {
		t.Errorf("lockstep %+v, want OK", report.Lockstep)
	}
	for _, reason := range report.WouldRefuseOn {
		if strings.Contains(reason, "marketplace") || strings.Contains(reason, "plugin.json") ||
			strings.Contains(reason, "launch-payload") || strings.Contains(reason, "installability") {
			t.Errorf("a plugin-only concern refused a binary: %s", reason)
		}
	}
	rows := map[string]GateSummary{}
	for _, g := range report.Gates {
		rows[g.Name] = g
	}
	for _, name := range []string{"installability-smoke", gateHookCompliant} {
		if rows[name].Status != "not_armed" || !strings.Contains(rows[name].Detail, "binary") {
			t.Errorf("row %s = %+v, want not_armed naming the kind", name, rows[name])
		}
	}

	// A declared path that cannot be read is refused by name.
	r.Write(ArtefactRelPath, `{"kind": "binary", "lockstep": ["app/meta.json", "gone.json"]}`)
	r.Commit("declare a file that does not exist")
	report, err = DryRun(DryRunRequest{RepoRoot: r.Root(), Version: "1.0.0", ExistingTags: []Semver{}})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, reason := range report.WouldRefuseOn {
		found = found || (strings.Contains(reason, "lockstep") && strings.Contains(reason, "gone.json"))
	}
	if !found {
		t.Errorf("an unreadable declared lockstep path was not refused by name: %v", report.WouldRefuseOn)
	}
}

// A plugin says the payload include set is what it scanned.
func TestDryRunForAPluginNamesThePayloadTree(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ArtefactRelPath, `{"kind": "plugin"}`)
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": ["commands"]}`)
	writeFile(t, root, "commands/x.md", "doc\n")
	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != KindPlugin || report.ScannedTree != PayloadTreeDescription {
		t.Errorf("kind %q, scanned tree %q", report.Kind, report.ScannedTree)
	}
}
