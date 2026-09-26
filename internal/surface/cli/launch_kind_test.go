package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// itd-2609150819432059: every launch verb runs against the declared artefact
// kind, and the release-cut gate reaches a kind that is not a plugin.

// binaryShipFixture is shipFixture declared as a managed Go binary: no plugin
// manifest, no payload include config, released at v0.4.0.
func binaryShipFixture(t *testing.T) *gittest.Repo {
	t.Helper()
	r := shipFixture(t)
	r.Remove(".claude-plugin/plugin.json")
	r.Remove(".claude-plugin/marketplace.json")
	r.Write(".abcd/config/artefact.json", `{"kind": "binary"}`+"\n")
	r.Write("main.go", "package main\n\nfunc main() {}\n")
	data, err := GenerateSurface(r.Root())
	if err != nil {
		t.Fatalf("GenerateSurface: %v", err)
	}
	r.Write(SurfaceSnapshotPath, string(data))
	r.Commit("the released binary")
	r.Git("tag", "-f", "v0.4.0")
	return r
}

// AC9: a kind the binary does not know refuses every launch verb, naming the
// kind and the accepted set, and writes nothing.
func TestEveryLaunchVerbRefusesAnUnknownKind(t *testing.T) {
	verbs := map[string][]string{
		"dry-run":  {"launch", "--dry-run"},
		"ship":     {"launch", "ship"},
		"scaffold": {"launch", "scaffold"},
		"receipts": {"launch", "receipts"},
		"archive":  {"launch", "archive", "--out", "OUT"},
	}
	for name, args := range verbs {
		t.Run(name, func(t *testing.T) {
			r := binaryShipFixture(t)
			r.Write(".abcd/config/artefact.json", `{"kind": "container-image"}`+"\n")
			r.Commit("declare a kind abcd does not know")
			out := t.TempDir()
			for i, a := range args {
				if a == "OUT" {
					args[i] = out
				}
			}
			stdout, err := shipIn(t, r, args...)
			if err == nil {
				t.Fatalf("%v accepted an unknown kind:\n%s", args, stdout)
			}
			for _, want := range []string{`"container-image"`, "plugin, binary, application"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("%v: the refusal does not name %q: %v", args, want, err)
				}
			}
			if status := r.Git("status", "--porcelain", "--ignored"); status != "" {
				t.Errorf("%v wrote into the repository:\n%s", args, status)
			}
			if entries, _ := os.ReadDir(out); len(entries) != 0 {
				t.Errorf("%v wrote into --out: %v", args, entries)
			}
		})
	}
}

// AC6: a declared binary's cut carrying an open major captured since the
// anchor tag, with no deferral, refuses naming the record, as a plugin's does.
func TestLaunchShipForADeclaredBinaryRefusesAnOpenMajor(t *testing.T) {
	r := binaryShipFixture(t)
	r.Write(".abcd/development/intents/shipped/itd-73-x.md", "---\nid: itd-73\nimpact: additive\n---\n# x\n")
	r.Write(".abcd/work/issues/open/iss-90-found-while-shipping.md",
		"---\nid: \"iss-90\"\nseverity: \"major\"\n---\n\nfound while shipping.\n")
	r.Commit("ship an intent and capture a major")

	out, err := shipIn(t, r, "launch", "ship")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("exit = %d, want 1 (the cut refuses)\n%s", code, out)
	}
	if !strings.Contains(string(out), "iss-90") {
		t.Errorf("the refusal does not name the record:\n%s", out)
	}
}

// AC7: the same cut with the major deferred out loud passes, and the report
// names the deferred record.
func TestLaunchShipForADeclaredBinaryPassesAndNamesADeferral(t *testing.T) {
	r := binaryShipFixture(t)
	r.Write(".abcd/development/intents/shipped/itd-73-x.md", "---\nid: itd-73\nimpact: additive\n---\n# x\n")
	r.Write(".abcd/work/issues/open/iss-90-found-while-shipping.md",
		"---\nid: \"iss-90\"\nseverity: \"major\"\ndeferred_after: \"v0.4.0\"\n"+
			"deferral_reason: \"the fix needs a schema migration\"\n---\n\nheld over.\n")
	r.Commit("ship an intent and defer what shipping it turned up")

	out, err := shipIn(t, r, "launch", "ship")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	if !strings.Contains(string(out), "deferred: iss-90") || !strings.Contains(string(out), "schema migration") {
		t.Errorf("the report does not name the deferred record and its reason:\n%s", out)
	}
}

// A declared binary has no plugin payload, so --payload-dir is refused before
// anything is read or written.
func TestLaunchShipRefusesAPayloadDirForANonPluginKind(t *testing.T) {
	r := binaryShipFixture(t)
	dest := filepath.Join(t.TempDir(), "payload")
	_, err := shipIn(t, r, "launch", "ship", "--changelog-json", "-", "--payload-dir", dest)
	if code := exitCodeOf(err); code != 2 || !strings.Contains(err.Error(), "binary") {
		t.Fatalf("exit %d (%v), want 2 naming the declared kind", code, err)
	}
	if _, statErr := os.Stat(dest); statErr == nil {
		t.Error("the refused ship staged a payload directory")
	}
}

// AC4 at the front door: the repository's own release workflow is named as left
// alone, and the stanza to add to it is printed.
func TestLaunchScaffoldForABinaryWithItsOwnReleaseWorkflowPrintsTheStanza(t *testing.T) {
	r := binaryShipFixture(t)
	own := "name: release\non:\n  push:\n    tags: ['v*']\njobs:\n  build:\n    runs-on: macos-latest\n    steps:\n      - run: make dmg\n"
	r.Write(".github/workflows/release.yml", own)
	r.Commit("the repository's own release workflow")

	out, err := shipIn(t, r, "launch", "scaffold")
	if err != nil {
		t.Fatalf("scaffold: %v\n%s", err, out)
	}
	for _, want := range []string{"kind binary", "[kept] .github/workflows/release.yml", "[written] .github/workflows/abcd-release-gate.yml",
		"uses: ./.github/workflows/abcd-release-gate.yml", "publish: false"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("the scaffold report does not say %q:\n%s", want, out)
		}
	}
	if got := readFileString(t, filepath.Join(r.Root(), ".github/workflows/release.yml")); got != own {
		t.Errorf("the repository's own release workflow changed:\n%s", got)
	}
}

// The scaffold header names a Go toolchain only for a Go module; a declared
// binary with no go.mod has none to name.
func TestLaunchScaffoldHeaderNamesNoGoToolchainWithoutGoMod(t *testing.T) {
	r := binaryShipFixture(t)
	out, err := shipIn(t, r, "launch", "scaffold")
	if err != nil {
		t.Fatalf("scaffold: %v\n%s", err, out)
	}
	header, _, _ := strings.Cut(string(out), "\n")
	if !strings.Contains(header, "kind binary, branch ") || strings.Contains(header, "go ") {
		t.Errorf("the scaffold header for a repository with no go.mod: %q, want the kind and branch and no go toolchain", header)
	}
}
