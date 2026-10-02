package cli

import (
	"bytes"
	"strings"
	"testing"
)

// help_truth_test.go — help sentences held to what the verb does, in both
// directions: the sentence says it, and the verb does it.

// TestHistorySeparationHelpNamesItsWritesAndRefusal: `history separation`
// resolves the store as every history read does (a missing store is created, a
// legacy corpus moved into it) and refuses outside a git checkout, so its help
// may claim neither that it writes nothing nor that it never refuses.
func TestHistorySeparationHelpNamesItsWritesAndRefusal(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"history", "separation", "--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("history separation --help exited %d: %s", code, stderr.String())
	}
	help := stdout.String()
	for _, claim := range []string{"Writes nothing", "never refuses"} {
		if strings.Contains(help, claim) {
			t.Errorf("history separation --help claims %q, which the verb does not hold:\n%s", claim, help)
		}
	}
	for _, want := range []string{"Writes a missing store or a legacy corpus move", "refuses outside a git checkout"} {
		if !strings.Contains(help, want) {
			t.Errorf("history separation --help does not say %q:\n%s", want, help)
		}
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"history", "separation"}, &stdout, &stderr); code == 0 {
		t.Fatalf("history separation outside a git checkout exited 0; the help says it refuses there\nstdout: %s", stdout.String())
	}
}

// TestUpdateCheckHelpClaimsNoMonopolyOnTheNetwork: `update --check` is not the
// only network touch besides the update itself (`launch --fetch-baseline`
// downloads a released plugin archive), so its flag help says only that it
// reaches the network when invoked.
func TestUpdateCheckHelpClaimsNoMonopolyOnTheNetwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"update", "--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("update --help exited %d: %s", code, stderr.String())
	}
	help := stdout.String()
	if strings.Contains(help, "the only network touch") {
		t.Errorf("update --help claims --check is the only network touch:\n%s", help)
	}
	if !strings.Contains(help, "it reaches the network only when invoked") {
		t.Errorf("update --help does not say --check reaches the network only when invoked:\n%s", help)
	}

	stdout.Reset()
	if code := Run([]string{"launch", "--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("launch --help exited %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "--fetch-baseline") {
		t.Fatalf("launch no longer carries --fetch-baseline; the network sentences in update --help and commands/version.md name it")
	}
}
