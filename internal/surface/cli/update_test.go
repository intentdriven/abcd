package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/update"
)

// runUpdateHermetic drives `abcd update` in a hermetic PATH/HOME with the
// updater-construction seam counted, so a dispatch refusal is proven to
// happen BEFORE anything network-capable exists.
func runUpdateHermetic(t *testing.T, args ...string) (string, error, int) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("ABCD_BIN_TARGET", "")
	t.Setenv("PATH", filepath.Join(home, ".local", "bin"))

	calls := 0
	orig := newUpdater
	newUpdater = func() updater { calls++; return orig() }
	t.Cleanup(func() { newUpdater = orig })

	var out bytes.Buffer
	cmd := NewRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(append([]string{"update"}, args...))
	err := cmd.Execute()
	return out.String(), err, calls
}

// TestUpdateRefusesWithNothingOnPath: the absent shape refuses loudly, names
// the install remedy, exits non-zero — and never constructs the updater, so
// the refusal path is provably network-free (the adr-38 seam).
func TestUpdateRefusesWithNothingOnPath(t *testing.T) {
	out, err, calls := runUpdateHermetic(t)
	if err == nil {
		t.Fatal("a refusal must exit non-zero")
	}
	if !strings.Contains(out, "nothing to update") || !strings.Contains(out, "install") {
		t.Errorf("the refusal is not loud or names no remedy:\n%s", out)
	}
	if calls != 0 {
		t.Errorf("the refusal path constructed the network updater %d times; the dispatch must refuse first", calls)
	}
}

// TestUpdateRefusesPluginRootEntry: an owned plugin-root symlink names the
// host's plugin-update path and touches nothing.
func TestUpdateRefusesPluginRootEntry(t *testing.T) {
	home := t.TempDir()
	pluginRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(pluginRoot, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "abcd"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(binDir, "abcd")
	if err := os.Symlink(filepath.Join(pluginRoot, "abcd"), link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", pluginRoot)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("ABCD_BIN_TARGET", "")
	t.Setenv("PATH", binDir)

	calls := 0
	orig := newUpdater
	newUpdater = func() updater { calls++; return orig() }
	t.Cleanup(func() { newUpdater = orig })

	var out bytes.Buffer
	cmd := NewRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"update"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("the plugin-root shape must refuse")
	}
	if !strings.Contains(out.String(), "plugin update") {
		t.Errorf("the refusal does not name the plugin-update path:\n%s", out.String())
	}
	if calls != 0 {
		t.Errorf("the plugin-root refusal constructed the updater %d times", calls)
	}
	if fi, serr := os.Lstat(link); serr != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the refusal touched the entry: %v %v", fi, serr)
	}
}

// TestUpdateSeamUntouchedByOtherVerbs is the adr-38 zero-network extension:
// no verb but update ever constructs the updater.
func TestUpdateSeamUntouchedByOtherVerbs(t *testing.T) {
	calls := 0
	orig := newUpdater
	newUpdater = func() updater { calls++; return orig() }
	t.Cleanup(func() { newUpdater = orig })

	for _, args := range [][]string{{"--version"}, {"rules"}} {
		cmd := NewRootCommand()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(args)
		_ = cmd.Execute()
	}
	if calls != 0 {
		t.Errorf("a non-update verb constructed the updater %d times", calls)
	}
}

// TestUpdateRejectsMalformedTag: a path-shaped tag refuses before any request.
func TestUpdateRejectsMalformedTag(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "abcd"), []byte("\x7fELFfake"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("ABCD_BIN_TARGET", "")
	t.Setenv("PATH", binDir)

	var out bytes.Buffer
	cmd := NewRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"update", "v1/../evil"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "not a release tag") {
		t.Fatalf("a malformed tag must refuse by shape, got: %v\n%s", err, out.String())
	}
}

// TestUpdateReceiptNamesTheUnpublishedBuildItReplaced: when the file abcd
// replaced belongs to no release the forge still serves, the receipt must say
// so by digest and name the proof that let abcd touch it at all. Rendering
// "updated ~/.local/bin/abcd:  -> v0.7.0" with an empty left-hand side would
// read as a bug in the very receipt that has to be trusted here
// (iss-2609012000222546).
func TestUpdateReceiptNamesTheUnpublishedBuildItReplaced(t *testing.T) {
	digest := strings.Repeat("ab", 32)
	var out bytes.Buffer
	renderUpdateReport(&out, false, update.Report{
		Action:     update.ActionSwapped,
		Origin:     "https://example.invalid/abcd",
		TargetPath: "~/.local/bin/abcd",
		OldDigest:  digest,
		Ownership:  update.OwnedByRunningExecutable,
		NewVersion: "v0.7.0",
		Digest:     strings.Repeat("cd", 32),
	})
	got := out.String()
	if !strings.Contains(got, "an unpublished build") {
		t.Errorf("the receipt must name what it replaced when no version could be derived:\n%s", got)
	}
	if !strings.Contains(got, digest) {
		t.Errorf("the receipt must carry the replaced file's digest:\n%s", got)
	}
	if !strings.Contains(got, update.OwnedByRunningExecutable.Prose()) {
		t.Errorf("the receipt must name the ownership proof that allowed the swap:\n%s", got)
	}
}

// TestUpdateReceiptKeepsTheOrdinaryVersionLine: the digest line is the
// exception, not the new normal — a provable old build still renders as a
// plain old -> new version pair with no digest noise.
func TestUpdateReceiptKeepsTheOrdinaryVersionLine(t *testing.T) {
	var out bytes.Buffer
	renderUpdateReport(&out, false, update.Report{
		Action:     update.ActionSwapped,
		Origin:     "https://example.invalid/abcd",
		TargetPath: "~/.local/bin/abcd",
		OldVersion: "v0.6.9",
		Ownership:  update.OwnedByManifest,
		NewVersion: "v0.7.0",
		Digest:     strings.Repeat("cd", 32),
	})
	got := out.String()
	// The swap's first line is the one wording every swap shares (CJ1b):
	// the bootstrap's success notice leads with the same line.
	if first, _, _ := strings.Cut(got, "\n"); first != update.UpdatedLine("v0.6.9", "v0.7.0") {
		t.Errorf("the receipt must open with %q; got %q", update.UpdatedLine("v0.6.9", "v0.7.0"), first)
	}
	if !strings.Contains(got, "~/.local/bin/abcd") {
		t.Errorf("the receipt must still name the path it swapped:\n%s", got)
	}
	if strings.Contains(got, "unpublished") {
		t.Errorf("a provable old build must not be reported as unpublished:\n%s", got)
	}
}

// TestUpdateJSONRefusalIsOneDocument — iss-2609282105241960. `update --json` on
// a dispatch refusal printed the receipt and then Run's error envelope: two
// JSON documents where a machine reader expects one. The refusal is ONE
// document that is both the receipt the chapter describes (action, the
// refusal's shape, detail and remedy) and the refusal the global --json
// contract describes (`"abcd": "error"`, the error, the exit code), and it
// carries no empty origin, since no release origin was reached.
func TestUpdateJSONRefusalIsOneDocument(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("ABCD_BIN_TARGET", "")
	t.Setenv("PATH", filepath.Join(home, ".local", "bin"))

	var stdout, stderr bytes.Buffer
	code := Run([]string{"update", "--json"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit = %d, want 1 for a refusal; stdout %q stderr %q", code, stdout.String(), stderr.String())
	}
	dec := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); err != io.EOF {
		t.Fatalf("stdout holds more than one JSON document (second: %s, err %v):\n%s", extra, err, stdout.String())
	}
	if doc["abcd"] != "error" || doc["action"] != "refused" {
		t.Errorf("the document is not both the refusal envelope and the receipt: %v", doc)
	}
	if ec, _ := doc["exit_code"].(float64); ec != 1 {
		t.Errorf("exit_code = %v, want 1", doc["exit_code"])
	}
	if msg, _ := doc["error"].(string); !strings.Contains(msg, "update refused (absent)") {
		t.Errorf("error = %q, want it to name the refusal", doc["error"])
	}
	ref, _ := doc["refusal"].(map[string]any)
	if ref == nil || ref["shape"] != "absent" || ref["remedy"] == "" || ref["remedy"] == nil {
		t.Errorf("the refusal block does not name its shape and remedy: %v", doc["refusal"])
	}
	if _, ok := doc["origin"]; ok {
		t.Errorf("a refusal raised before any fetch carries an origin key: %v", doc)
	}
	if stderr.Len() != 0 {
		t.Errorf("--json wrote to stderr: %q", stderr.String())
	}
}

// progressRecordingUpdater stands in for the release client on the swap path:
// it resolves nothing over the network, records the progress writer the verb
// handed it, and writes a progress line to that writer when it has one, as
// the real client does.
type progressRecordingUpdater struct {
	progress io.Writer
	applied  bool
}

func (f *progressRecordingUpdater) ResolveTag(requested string) (string, error) {
	return requested, nil
}

func (f *progressRecordingUpdater) Apply(target, tag string, progress io.Writer) (update.Report, error) {
	f.applied = true
	f.progress = progress
	if progress != nil {
		_, _ = io.WriteString(progress, "\r  downloading abcd-test-arch "+tag+": 100%\n")
	}
	return swappedReceipt(tag), nil
}

func swappedReceipt(tag string) update.Report {
	return update.Report{
		Action:     update.ActionSwapped,
		Origin:     "https://example.invalid/abcd",
		Tag:        tag,
		TargetPath: "~/.local/bin/abcd",
		OldVersion: "v0.6.1",
		NewVersion: tag,
		Digest:     strings.Repeat("cd", 32),
		Ownership:  update.OwnedByManifest,
	}
}

// runUpdateSwap drives `abcd update <tag>` down the swap path — a regular file
// on a hermetic PATH, which the dispatch lets through — with the release client
// replaced by the recording fake, and stderr the writer given.
func runUpdateSwap(t *testing.T, stderr io.Writer) (*progressRecordingUpdater, string, int) {
	t.Helper()
	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "abcd"), []byte("\x7fELFfake"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("ABCD_BIN_TARGET", "")
	t.Setenv("PATH", binDir)

	fake := &progressRecordingUpdater{}
	orig := newUpdater
	newUpdater = func() updater { return fake }
	t.Cleanup(func() { newUpdater = orig })

	var stdout bytes.Buffer
	code := Run([]string{"update", "v0.6.2"}, &stdout, stderr)
	return fake, stdout.String(), code
}

// TestUpdatePipedPrintsNoProgress is the non-TTY silence test spc-32 promised
// (criterion 9, iss-2609300015353414): with the output piped — stdout and
// stderr both a pipe or a buffer, never a terminal — the swap is handed no
// progress writer, stdout carries the receipt and nothing else, and stderr
// stays empty. The gate reads the stream progress is written to, so this holds
// whatever the test process's own stderr happens to be attached to.
func TestUpdatePipedPrintsNoProgress(t *testing.T) {
	var stderr bytes.Buffer
	fake, stdout, code := runUpdateSwap(t, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0 for a swap; stdout %q stderr %q", code, stdout, stderr.String())
	}
	if !fake.applied {
		t.Fatal("the swap path was not reached; the test proves nothing about progress")
	}
	if fake.progress != nil {
		t.Errorf("a non-terminal stderr was handed a progress writer: %T", fake.progress)
	}
	var want bytes.Buffer
	renderUpdateReport(&want, false, swappedReceipt("v0.6.2"))
	if stdout != want.String() {
		t.Errorf("stdout carries more than the receipt:\n got %q\nwant %q", stdout, want.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("piped stderr carries output: %q", stderr.String())
	}
}

// TestUpdateTerminalStderrGetsProgress is the other half, so the silence above
// cannot be met by a gate that never opens: a stderr the terminal check
// accepts is handed the progress writer. The check is a termios get that only
// a real terminal answers, so the isTTY seam stands /dev/null in for one.
func TestUpdateTerminalStderrGetsProgress(t *testing.T) {
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("no %s to stand in for a terminal: %v", os.DevNull, err)
	}
	defer devNull.Close()
	prev := isTTY
	isTTY = func(f *os.File) bool { return f == devNull }
	t.Cleanup(func() { isTTY = prev })
	fake, stdout, code := runUpdateSwap(t, devNull)
	if code != 0 {
		t.Fatalf("exit = %d, want 0 for a swap; stdout %q", code, stdout)
	}
	if fake.progress == nil {
		t.Error("a terminal stderr was handed no progress writer")
	}
	var want bytes.Buffer
	renderUpdateReport(&want, false, swappedReceipt("v0.6.2"))
	if stdout != want.String() {
		t.Errorf("progress reached stdout:\n got %q\nwant %q", stdout, want.String())
	}
}
