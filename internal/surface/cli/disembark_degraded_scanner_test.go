package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The armed half of `disembark pack`'s fail-closed secret scan.
//
// The guard already refused a degraded scanner config and no test asserted it.
// That is a bad place to leave unarmed: a lifeboat is a copy of a repository's
// record written OUT of the repository, usually so it can be handed somewhere
// else, and the scan is the only thing between the source's secrets and that
// copy. With the source's own detectors silently dropped, the pack would report a
// clean scan it never performed with the rules the repository actually declared.
//
// This test lives in its own file rather than beside the other disembark tests
// because internal/surface/cli/cli.go — where the guard is — is being edited by a
// concurrent session; nothing here touches that file.

// degradePackScanner writes a per-repo .abcd/config/pii.json that cannot be
// parsed. scanner.New still returns a USABLE scanner on this path, falling back
// to the bundled pattern set, so the repository's own detectors vanish with no
// in-band signal; Unavailable() is the only one, and the guard under test is the
// only thing that reads it.
func degradePackScanner(t *testing.T, repoRoot string) {
	t.Helper()
	dir := filepath.Join(repoRoot, ".abcd", "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pii.json"), []byte("{ this is not json"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestDisembarkPackRefusesADegradedScanner: the pack must not ship a lifeboat
// scanned under a weakened ruleset, and must leave nothing behind at the
// destination when it refuses.
func TestDisembarkPackRefusesADegradedScanner(t *testing.T) {
	source := t.TempDir()
	degradePackScanner(t, source)
	// A minimal record so the pack has something it could have written, and so a
	// pass here would be a pass on real work rather than on an empty source.
	if err := os.MkdirAll(filepath.Join(source, ".abcd", "development"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "AGENTS.md"), []byte("# Router\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "lifeboat")

	var stdout, stderr bytes.Buffer
	code := runPack(t, &stdout, &stderr, "disembark", "pack", source, dest)
	if code == 0 {
		t.Fatalf("pack succeeded under a degraded scanner config\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
	}
	if code != 2 {
		t.Errorf("a check that could not run is an environment fault: want exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "scanner unavailable") {
		t.Errorf("the refusal must name the scanner, so the operator fixes the config rather than the source tree:\n%s", stderr.String())
	}
	// The proof that does not depend on the command's own account of itself.
	if entries, err := os.ReadDir(dest); err == nil && len(entries) > 0 {
		t.Errorf("a refusal wrote %d entr(ies) into the destination: %v", len(entries), entries)
	}
}

// runPack runs one invocation and mirrors Run's error surface, so a test reads
// the diagnostic a caller sees rather than an error value the process never
// prints.
func runPack(t *testing.T, stdout, stderr *bytes.Buffer, args ...string) int {
	t.Helper()
	root := NewRootCommand()
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetIn(strings.NewReader(""))
	err := root.Execute()
	if err == nil {
		return 0
	}
	code := 1
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		code = coded.ExitCode()
	}
	if msg := scrubPaths(err); msg != "" {
		stderr.WriteString("abcd: " + msg + "\n")
	}
	return code
}
