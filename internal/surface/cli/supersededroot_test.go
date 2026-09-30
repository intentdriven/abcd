package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The third failure shape of iss-2608230943088357, recorded as
// iss-2609020113012227: a plugin root is named for the commit it was installed
// from, so a hash-pinned path baked into a skill page is designed to expire —
// and because nothing prunes the old roots, following the expired path runs a
// SUPERSEDED binary that answers a known verb confidently, exit 0, no error.
//
// The two shapes main already fixed are loud (an unknown flag, an unknown
// command). This one is silent, and the surface it was observed on is the one
// whose entire job is reporting which version is installed.

// rootAt builds a valid plugin root at dir: the hooks/ directory the layout
// check keys on, and the binary beside it.
func rootAt(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "abcd"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// twoRoots stands a session up with a current plugin root and a superseded one
// beside it, and points the running-binary seam at the superseded one — the
// state a command page's pinned path produces between an update and a reload.
func twoRoots(t *testing.T) (superseded, current string) {
	t.Helper()
	base := t.TempDir()
	superseded = rootAt(t, filepath.Join(base, "0e22abfd6739"))
	current = rootAt(t, filepath.Join(base, "8f68ffb34558"))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", current)
	t.Setenv("CLAUDE_PLUGIN_DATA", "")
	t.Chdir(t.TempDir())
	setExecutable(t, filepath.Join(superseded, "abcd"))
	return superseded, current
}

// decodeVersion runs `abcd --version --json` and decodes it.
func decodeVersion(t *testing.T) map[string]any {
	t.Helper()
	out := runCLI(t, "--version", "--json")
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	return got
}

// TestVersionNamesASupersededPluginRoot is the defect: the verb whose job is
// reporting the installed version answers from a root this session does not
// serve, and must say so rather than answering confidently.
func TestVersionNamesASupersededPluginRoot(t *testing.T) {
	twoRoots(t)
	got := decodeVersion(t)
	note, _ := got["superseded_root"].(string)
	if note == "" {
		t.Fatalf("version reported no superseded-root note: %v", got)
	}
	// Both roots are named by the commit they were installed from; the note
	// carries both, so a reader can tell which answer they got.
	for _, want := range []string{"0e22abfd6739", "8f68ffb34558"} {
		if !strings.Contains(note, want) {
			t.Errorf("note does not name %q: %s", want, note)
		}
	}
	// The report itself is unchanged: this is a note beside the answer, never a
	// refusal and never a different answer.
	if got["staleness"] == nil {
		t.Errorf("the version report lost a field: %v", got)
	}
}

// TestVersionPlainRenderCarriesTheNote: a field only --json shows is invisible
// to everyone who does not know to ask, which is the whole failure here.
func TestVersionPlainRenderCarriesTheNote(t *testing.T) {
	twoRoots(t)
	out := string(runCLI(t, "--version"))
	if !strings.Contains(out, "0e22abfd6739") || !strings.Contains(out, "8f68ffb34558") {
		t.Errorf("the plain render does not name the two roots:\n%s", out)
	}
}

// TestVersionSaysNothingWhenTheRootsAgree: the note is evidence, not decoration.
// A binary running from the root this session resolves has nothing to disclose.
func TestVersionSaysNothingWhenTheRootsAgree(t *testing.T) {
	_, current := twoRoots(t)
	setExecutable(t, filepath.Join(current, "abcd"))
	got := decodeVersion(t)
	if note, ok := got["superseded_root"]; ok {
		t.Errorf("a binary in this session's own plugin root must say nothing, got %v", note)
	}
}

// TestVersionSaysNothingWhenTheBinaryIsInNoPluginRoot: a PATH copy, a source
// build, a go-run binary — none of them is a superseded ROOT, and claiming one
// would be a fact the disk does not carry.
func TestVersionSaysNothingWhenTheBinaryIsInNoPluginRoot(t *testing.T) {
	twoRoots(t)
	loose := t.TempDir()
	if err := os.WriteFile(filepath.Join(loose, "abcd"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	setExecutable(t, filepath.Join(loose, "abcd"))
	got := decodeVersion(t)
	if note, ok := got["superseded_root"]; ok {
		t.Errorf("a binary outside every plugin root must say nothing, got %v", note)
	}
}

// TestAhoyNamesASupersededPluginRoot is the sibling surface: bare `abcd ahoy` renders
// the same vintage and staleness pair through the same comparator, so the same
// superseded root produces the same confident wrong answer there.
func TestAhoyNamesASupersededPluginRoot(t *testing.T) {
	twoRoots(t)
	out := runCLI(t, "ahoy", "--json")
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	note, _ := got["superseded_root"].(string)
	if note == "" {
		t.Fatalf("ahoy reported no superseded-root note: %s", out)
	}
	if !strings.Contains(note, "0e22abfd6739") {
		t.Errorf("note does not name the superseded root: %s", note)
	}
}

// sourceCheckoutAt turns a plugin root into a source checkout of abcd, which is
// what isSourceCheckout keys on: the repository's own entry point.
func sourceCheckoutAt(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "cmd", "abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmd", "abcd", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestVersionSaysNothingWhenTheServingRootIsASourceCheckout: a source checkout
// is a valid plugin root — hooks/ sits at the top of this very repository — so
// running the `make build` artefact from a checkout while a harness session
// resolves its own cache root satisfies the divergence test. The note is false
// in that direction (a checkout is not named for a commit it was installed
// from, and its "re-run through this session's plugin root" remedy points at
// the binary the dogfooding rule calls stale), so it must not fire.
func TestVersionSaysNothingWhenTheServingRootIsASourceCheckout(t *testing.T) {
	superseded, _ := twoRoots(t)
	sourceCheckoutAt(t, superseded)
	got := decodeVersion(t)
	if note, ok := got["superseded_root"]; ok {
		t.Errorf("a source checkout is not a superseded plugin root, got %v", note)
	}
}

// TestVersionStillNamesASupersededRootWhenTheSessionIsASourceCheckout: the
// guard is keyed on the root that SERVED the answer, not on the session's. A
// provisioned root answering into a source-checkout session is still an answer
// from a root this session does not serve, and the note still applies.
func TestVersionStillNamesASupersededRootWhenTheSessionIsASourceCheckout(t *testing.T) {
	_, current := twoRoots(t)
	sourceCheckoutAt(t, current)
	got := decodeVersion(t)
	note, _ := got["superseded_root"].(string)
	if note == "" {
		t.Fatalf("a provisioned root serving a source-checkout session must still be named: %v", got)
	}
	if !strings.Contains(note, "0e22abfd6739") {
		t.Errorf("note does not name the serving root: %s", note)
	}
}

// TestAhoyPlainRenderCarriesTheNote: bare `ahoy` without --json is what a person
// reads, so the note is in the plain render too, not only in the envelope.
func TestAhoyPlainRenderCarriesTheNote(t *testing.T) {
	twoRoots(t)
	out := string(runCLI(t, "ahoy"))
	if !strings.Contains(out, "0e22abfd6739") || !strings.Contains(out, "8f68ffb34558") {
		t.Errorf("the plain ahoy render does not name the two roots:\n%s", out)
	}
}

// TestSupersededRootNoteIsSanitised: the two root names are directory names read
// off the disk, and a directory name may carry any byte but a slash. A terminal
// escape or a bidirectional override in one would reach the terminal through the
// plain render, and an agent relaying the note would carry it on, so each name
// passes through termsafe.Sanitize before it is printed.
func TestSupersededRootNoteIsSanitised(t *testing.T) {
	base := t.TempDir()
	hostile := "0e22abfd\x1b[31m‮6739"
	superseded := rootAt(t, filepath.Join(base, hostile))
	current := rootAt(t, filepath.Join(base, "8f68ffb34558"))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", current)
	t.Setenv("CLAUDE_PLUGIN_DATA", "")
	t.Chdir(t.TempDir())
	setExecutable(t, filepath.Join(superseded, "abcd"))

	for _, args := range [][]string{{"--version"}, {"ahoy"}} {
		out := string(runCLI(t, args...))
		if !strings.Contains(out, "0e22abfd?[31m?6739") {
			t.Errorf("%v: the note does not carry the sanitised root name:\n%q", args, out)
		}
		if strings.ContainsRune(out, '\x1b') || strings.ContainsRune(out, '‮') {
			t.Errorf("%v: a control or bidirectional character reached the render:\n%q", args, out)
		}
	}
	note, _ := decodeVersion(t)["superseded_root"].(string)
	if strings.ContainsRune(note, '\x1b') || strings.ContainsRune(note, '‮') {
		t.Errorf("the --json note carries a control or bidirectional character: %q", note)
	}
}

// TestCommandPagesRelayTheNoteAsPrinted: the note is built from sanitised names,
// so a page that told the agent to relay it "verbatim" would read as a claim that
// the bytes on disk reach the person unchanged, and would invite rebuilding the
// names from a path, which undoes the sanitising. Each page that runs a surface
// carrying the note says to relay it as abcd printed it, names what the
// sanitising removed, and forbids reconstructing the names.
func TestCommandPagesRelayTheNoteAsPrinted(t *testing.T) {
	for _, page := range []string{"version.md", "ahoy.md"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "commands", page))
		if err != nil {
			t.Fatal(err)
		}
		// Rewrapped prose: compare on single spaces, in one case.
		text := strings.ToLower(strings.Join(strings.Fields(string(data)), " "))
		for _, want := range []string{
			"superseded_root",
			"as abcd printed it",
			"control and bidirectional characters",
			"never rebuild",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("commands/%s does not say %q about the superseded-root note", page, want)
			}
		}
	}
}
