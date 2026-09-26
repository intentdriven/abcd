package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLaunchDryRunWithNoDeclarationNamesItsHome is itd-2609150819432059 AC1: a
// managed repository that has not declared its artefact kind is told where the
// declaration lives and which kinds it accepts — never a missing-file error —
// and nothing is written into it.
func TestLaunchDryRunWithNoDeclarationNamesItsHome(t *testing.T) {
	r := shipFixture(t)
	out, err := shipIn(t, r, "launch", "--dry-run")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("exit = %d, want 1\n%s", code, out)
	}
	for _, want := range []string{".abcd/config/artefact.json", "plugin, binary, application", "abcd ahoy install"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q:\n%v", want, err)
		}
	}
	if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no such file") {
		t.Errorf("the refusal reads as a missing-file error:\n%v", err)
	}
	if _, statErr := os.Stat(filepath.Join(r.Root(), ".abcd", ".work.local")); statErr == nil {
		t.Error("a repository with no declaration must not have a report written into it")
	}
}

// TestLaunchDryRunInANonPayloadPluginNamesTheReleasePath is iss-2608270559313719:
// a repository declaring a plugin with no launch payload is told which release
// path it does have, not handed a missing-file error.
func TestLaunchDryRunInANonPayloadPluginNamesTheReleasePath(t *testing.T) {
	r := shipFixture(t)
	r.Write(".abcd/config/artefact.json", `{"kind": "plugin"}`+"\n")
	r.Commit("declare the plugin kind")
	out, err := shipIn(t, r, "launch", "--dry-run")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("exit = %d, want 1\n%s", code, out)
	}
	for _, want := range []string{"declares kind plugin", "no launch payload", "launch scaffold", "CHANGELOG", "auto-release", "binary or application"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q:\n%v", want, err)
		}
	}
	if _, statErr := os.Stat(filepath.Join(r.Root(), ".abcd", ".work.local")); statErr == nil {
		t.Error("a repository with no launch payload must not have a report written into it")
	}
}

// TestLaunchDryRunForADeclaredBinarySaysWhichTreeItScanned is AC8 at the front
// door: a declared non-plugin kind previews with no payload config, and the
// report says which tree was scanned.
func TestLaunchDryRunForADeclaredBinarySaysWhichTreeItScanned(t *testing.T) {
	r := shipFixture(t)
	r.Remove(".claude-plugin/plugin.json")
	r.Remove(".claude-plugin/marketplace.json")
	r.Write(".abcd/config/artefact.json", `{"kind": "binary"}`+"\n")
	r.Write("main.go", "package main\n\nfunc main() {}\n")
	r.Commit("a managed binary")
	out, err := shipIn(t, r, "launch", "--dry-run")
	if err != nil {
		t.Fatalf("a declared binary must preview: %v\n%s", err, out)
	}
	for _, want := range []string{"artefact kind:  binary", "scanned tree:   the tree the release tag would archive"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("the preview does not say %q:\n%s", want, out)
		}
	}
	for _, not := range []string{"launch-payload", "marketplace.json", "plugin.json"} {
		if strings.Contains(string(out), "would refuse on: "+not) {
			t.Errorf("a plugin-only read refused a binary (%s):\n%s", not, out)
		}
	}
}
