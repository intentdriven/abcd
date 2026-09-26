package launch

import (
	"strings"
	"testing"
)

// TestDryRunAssertsTheDevPolarity is the adr-19 detector for the gate a
// maintainer actually runs.
//
// The dry-run and the ship read the SOURCE TREE, and post-adr-19 the source tree
// carries no version key by design. Asserting the public polarity there accuses
// a correct tree of drift and tells the operator to add the very key adr-19
// keeps out; the public polarity belongs over the RENDERED PAYLOAD, where
// RenderPayload already applies it.
//
// The cases are the two polarities of that one rule: a version-absent tree is
// clean, and a tree that somehow acquired a version key is drift.
func TestDryRunAssertsTheDevPolarity(t *testing.T) {
	cases := []struct {
		name string
		// version is the key written into both manifests; empty writes none.
		version   string
		wantOK    bool
		wantDrift string
	}{
		{
			name:   "an adr-19 version-absent tree is clean",
			wantOK: true,
		},
		{
			name:      "a version key in the working tree is drift",
			version:   "1.2.3",
			wantOK:    false,
			wantDrift: "DRIFT dev .claude-plugin/plugin.json/version",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, ".abcd/config/launch-payload.json",
				`{"includes": [".claude-plugin", "README.md"]}`)
			writeFile(t, root, "README.md", "clean readme\n")
			writeLockstepTree(t, root, tc.version, tc.version, tc.version)

			report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3"})
			if err != nil {
				t.Fatalf("dry-run preflight must succeed: %v", err)
			}
			if report.Lockstep.Tree != TreeDev {
				t.Errorf("the source tree must be checked under %q, got %q", TreeDev, report.Lockstep.Tree)
			}
			if report.Lockstep.OK != tc.wantOK {
				t.Fatalf("lockstep OK = %v, want %v (%+v)", report.Lockstep.OK, tc.wantOK, report.Lockstep)
			}
			if tc.wantOK {
				for _, r := range report.WouldRefuseOn {
					if strings.Contains(r, "lockstep") {
						t.Errorf("a version-absent tree must not be refused: %q", r)
					}
				}
				return
			}
			var saw bool
			for _, d := range report.Lockstep.Drifts {
				if strings.Contains(d, tc.wantDrift) {
					saw = true
				}
			}
			if !saw {
				t.Errorf("expected a drift naming %q, got %v", tc.wantDrift, report.Lockstep.Drifts)
			}
		})
	}
}

// TestDryRunVersionComesFromTheCaller pins where the previewed version comes
// from: the caller supplies it, because adr-19 leaves nothing in the tree to
// read. A dry-run that resolved it from a manifest would report the empty string
// on every correct repository and then refuse retention for it.
func TestDryRunVersionComesFromTheCaller(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": [".claude-plugin", "README.md"]}`)
	writeFile(t, root, "README.md", "clean readme\n")
	writeLockstepTree(t, root, "", "", "")

	// The tag set is injected: this test is about where the VERSION comes from,
	// and the fixture is not a repository, so a real tag listing would fail —
	// which the retention preview reports as a refusal (iss-194).
	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "2.5.0", ExistingTags: []Semver{}})
	if err != nil {
		t.Fatalf("dry-run preflight must succeed: %v", err)
	}
	if report.Version != "2.5.0" {
		t.Errorf("Version = %q, want the supplied 2.5.0", report.Version)
	}
	if report.Retention.Refused {
		t.Errorf("a supplied strict-SemVer version must not refuse retention: %s", report.Retention.RefusalReason)
	}
}
