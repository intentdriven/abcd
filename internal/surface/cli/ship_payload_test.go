package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/core/release"
	"github.com/intentdriven/abcd/internal/gittest"
)

// shipRenderableRepo is shipReadyRepo plus the two config artefacts a payload
// render needs: the adr-19 version-location contract (WHERE the version goes)
// and the payload includes (WHAT ships).
func shipRenderableRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := shipReadyRepo(t)
	r.Write(".abcd/config/version-location.json",
		`{"manifest_path": ".claude-plugin/plugin.json", "json_pointer": "/version"}`+"\n")
	r.Write(".abcd/config/launch-payload.json",
		`{"includes": [".claude-plugin", "CHANGELOG.md"]}`+"\n")
	r.Commit("the release configuration")
	return r
}

// TestLaunchShipRendersTheVersionedPayload is the wiring detector for the
// release-payload render: a ship that lands the changelog heading also stages a
// payload whose manifests carry the DERIVED version, while the working tree's
// own manifests stay byte-identical and version-absent (adr-19).
func TestLaunchShipRendersTheVersionedPayload(t *testing.T) {
	r := shipRenderableRepo(t)
	dest := filepath.Join(t.TempDir(), "payload")
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")

	before, err := os.ReadFile(filepath.Join(r.Root(), ".claude-plugin/plugin.json"))
	if err != nil {
		t.Fatal(err)
	}

	out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload, "--payload-dir", dest)
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s\n%v", code, out, err)
	}
	if !strings.Contains(string(out), "payload:") {
		t.Errorf("the render is not reported:\n%s", out)
	}

	staged := readJSON(t, filepath.Join(dest, ".claude-plugin/plugin.json"))
	if got := staged["version"]; got != "0.4.1" {
		t.Errorf("staged plugin.json version = %v, want the derived 0.4.1", got)
	}
	market := readJSON(t, filepath.Join(dest, ".claude-plugin/marketplace.json"))
	entry := market["plugins"].([]any)[0].(map[string]any)
	if entry["version"] != "0.4.1" {
		t.Errorf("staged marketplace version = %v, want 0.4.1", entry["version"])
	}
	changelog, ok := entry["changelog"].(map[string]any)
	if !ok || changelog["version"] != "0.4.1" || changelog["tier"] != "patch" {
		t.Errorf("staged marketplace changelog entry = %v, want version 0.4.1 tier patch", entry["changelog"])
	}

	after, err := os.ReadFile(filepath.Join(r.Root(), ".claude-plugin/plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Error("adr-19: the working tree's plugin.json was mutated by the ship")
	}
	if res := launch.CheckLockstep(launch.TreeDev, r.Root(), filepath.Join(r.Root(), ".abcd/config/version-location.json")); !res.OK {
		t.Errorf("the working tree must stay version-absent after a ship, got %+v", res)
	}
	if res := launch.CheckLockstep(launch.TreePublic, dest, filepath.Join(r.Root(), ".abcd/config/version-location.json")); !res.OK {
		t.Errorf("the staged payload must satisfy the public polarity, got %+v", res)
	}
}

// TestLaunchShipPayloadDirNeedsTheIngestStep pins the flag's scope: the emit
// step composes nothing and writes nothing, so asking it for a payload is an
// operand error rather than a half-rendered directory.
func TestLaunchShipPayloadDirNeedsTheIngestStep(t *testing.T) {
	r := shipRenderableRepo(t)
	dest := filepath.Join(t.TempDir(), "payload")

	out, err := shipIn(t, r, "launch", "ship", "--payload-dir", dest)
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("exit = %d, want 2\n%s", code, out)
	}
	if !strings.Contains(err.Error(), "--changelog-json") {
		t.Errorf("the error should name the missing operand, got %v", err)
	}
	if _, statErr := os.Stat(dest); statErr == nil {
		t.Error("nothing may be staged when the flag combination is rejected")
	}
}

// TestLaunchShipWithoutPayloadDirStagesNothing keeps the render opt-in: the
// existing ship path is unchanged when no destination is named.
func TestLaunchShipWithoutPayloadDirStagesNothing(t *testing.T) {
	r := shipRenderableRepo(t)
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")

	out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload)
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	if strings.Contains(string(out), "payload:") {
		t.Errorf("no payload was requested, so none may be reported:\n%s", out)
	}
}

// TestLaunchShipRefusedPayloadWritesNothing is the atomicity detector for the
// two-step ship: every reason the render can REFUSE must be found before the
// dated CHANGELOG heading is written, because a written heading is a durable
// release record that permanently refuses the retry as release-in-flight.
//
// Each case drives the shipped verb end to end and then asserts the whole of the
// filesystem contract the refusal claims: CHANGELOG.md byte-identical, and no
// half-staged payload left behind.
func TestLaunchShipRefusedPayloadWritesNothing(t *testing.T) {
	cases := []struct {
		name string
		// setup prepares the repository and returns the payload destination.
		setup func(t *testing.T, r *gittest.Repo) string
		want  string
	}{
		{
			name: "the destination is not empty",
			setup: func(t *testing.T, _ *gittest.Repo) string {
				dest := filepath.Join(t.TempDir(), "payload")
				if err := os.MkdirAll(dest, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dest, "leftover.txt"), []byte("x\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				return dest
			},
			want: "not empty",
		},
		{
			name: "a declared hook is not in the payload",
			setup: func(t *testing.T, r *gittest.Repo) string {
				r.Write("hooks/hooks.json",
					`{"hooks":{"SessionStart":[{"hooks":[{"type":"command","command":"$CLAUDE_PLUGIN_ROOT/scripts/go.sh"}]}]}}`+"\n")
				r.Write("scripts/go.sh", "#!/bin/sh\nexit 0\n")
				r.Write(".abcd/config/launch-payload.json",
					`{"includes": [".claude-plugin", "hooks", "CHANGELOG.md"]}`+"\n")
				r.Commit("a hook whose script the payload excludes")
				return filepath.Join(t.TempDir(), "payload")
			},
			want: "is not in the payload",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := shipRenderableRepo(t)
			dest := tc.setup(t, r)
			payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")

			before, err := os.ReadFile(filepath.Join(r.Root(), "CHANGELOG.md"))
			if err != nil {
				t.Fatal(err)
			}

			out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload, "--payload-dir", dest)
			if code := exitCodeOf(err); code != 2 {
				t.Fatalf("exit = %d, want 2\n%s\n%v", code, out, err)
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("the refusal should name %q, got %v", tc.want, err)
			}

			after, err := os.ReadFile(filepath.Join(r.Root(), "CHANGELOG.md"))
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != string(before) {
				t.Errorf("a refused render wrote the release record anyway:\n%s", after)
			}
			for _, staged := range []string{".claude-plugin/plugin.json", ".claude-plugin/marketplace.json"} {
				if _, statErr := os.Stat(filepath.Join(dest, staged)); statErr == nil {
					t.Errorf("a refused render left %s staged in the destination", staged)
				}
			}
		})
	}
}

// TestRollbackCutRestoresTheRecord covers the backstop behind the precheck: if a
// refusal ever does slip past it and land after the write, the release record
// goes back and the half-staged directory goes away, and the operator is TOLD
// which of the two happened.
//
// It is exercised directly because the precheck is what makes that path
// unreachable through the verb — and an untested rollback is exactly the code
// that fails the one time it runs.
func TestRollbackCutRestoresTheRecord(t *testing.T) {
	cases := []struct {
		name string
		// breakIt makes the rollback fail; nil leaves it able to succeed.
		breakIt func(t *testing.T, repoRoot, dest string)
		want    string
	}{
		{
			name: "the record goes back and the staging goes away",
			want: "rolled back",
		},
		{
			name: "an unrestorable record is reported, not swallowed",
			breakIt: func(t *testing.T, repoRoot, _ string) {
				// A directory where CHANGELOG.md belongs: the restore cannot
				// write it, which is precisely the case an operator must be told
				// to recover by hand.
				if err := os.RemoveAll(filepath.Join(repoRoot, "CHANGELOG.md")); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Join(repoRoot, "CHANGELOG.md", "x"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			want: "THE ROLLBACK FAILED",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := shipReadyRepo(t)
			before, err := os.ReadFile(filepath.Join(r.Root(), "CHANGELOG.md"))
			if err != nil {
				t.Fatal(err)
			}
			undo := ingestForRollback(t, r)
			dest := filepath.Join(t.TempDir(), "payload")
			if err := os.MkdirAll(dest, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dest, "staged.json"), []byte("{}\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if tc.breakIt != nil {
				tc.breakIt(t, r.Root(), dest)
			}

			got := rollbackCut(r.Root(), dest, undo)
			if !strings.Contains(got, tc.want) {
				t.Errorf("rollback report = %q, want it to mention %q", got, tc.want)
			}
			if tc.breakIt != nil {
				return
			}
			restored, err := os.ReadFile(filepath.Join(r.Root(), "CHANGELOG.md"))
			if err != nil {
				t.Fatal(err)
			}
			if string(restored) != string(before) {
				t.Errorf("the release record was not restored, got %q", restored)
			}
			if _, err := os.Stat(dest); err == nil {
				t.Error("the staging directory the render created must be removed")
			}
		})
	}
}

// ingestForRollback runs a real ingest and returns its undo, so the rollback is
// exercised against the writes a cut actually makes.
func ingestForRollback(t *testing.T, r *gittest.Repo) release.UndoPlan {
	t.Helper()
	raw, err := os.ReadFile(composedPayload(t, t.TempDir(), "v0.4.1", "itd-73"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := ingestCut(r.Root(), raw, time.Date(2026, 7, 21, 9, 30, 0, 0, time.UTC))
	if err != nil || !res.Written {
		t.Fatalf("ingest: written=%v err=%v", res.Written, err)
	}
	return res.Undo
}

// TestRollbackCutRestoresThePageAndArchive: the rollback a refused render
// triggers undoes the whole cut — the page comes back, the archive goes, and the
// tree is as it was.
func TestRollbackCutRestoresThePageAndArchive(t *testing.T) {
	r := shipReadyRepo(t)
	r.Write("RELEASE.md", "# Release 0.4.0 (2026-07-01)\n\nThe base. (itd-1)\n")
	r.Commit("the previous release page")
	before := cliTreeDigest(t, r.Root())

	undo := ingestForRollback(t, r)
	if _, err := os.Stat(filepath.Join(r.Root(), ".abcd/development/releases/0.4.0.md")); err != nil {
		t.Fatalf("the cut did not archive the outgoing page: %v", err)
	}
	dest := filepath.Join(t.TempDir(), "payload")
	if got := rollbackCut(r.Root(), dest, undo); !strings.Contains(got, "rolled back") {
		t.Fatalf("rollback report = %q", got)
	}
	if after := cliTreeDigest(t, r.Root()); after != before {
		t.Error("the rollback did not restore the page and remove the archive")
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return doc
}

// TestLaunchDryRunSanitisesRefusalReasons is the S2 regression: a bundle reason in
// the dry-run render embeds a raw repo filename, and a control-char-rejected path
// carries the offending bytes. The human render must route each reason through the
// terminal sanitiser (like the citation line), so a committed filename cannot inject
// raw escapes into the maintainer's release preview or a CI log.
func TestLaunchDryRunSanitisesRefusalReasons(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config", "launch-payload.json"),
		[]byte(`{"includes": ["commands"]}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "commands"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A filename carrying a raw ESC (git permits it); the bundle rejects it as
	// control_char and puts the raw path into WouldRefuseOn.
	evil := filepath.Join(repo, "commands", "evil\x1b[31mRED.md")
	if err := os.WriteFile(evil, []byte("# x\n"), 0o644); err != nil {
		t.Skipf("cannot create control-char filename: %v", err)
	}

	t.Chdir(repo)
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"launch", "--dry-run"}, &stdout, &stderr); code != 0 {
		t.Fatalf("dry-run exit = %d, want 0\nstderr:%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "would refuse on") {
		t.Fatalf("dry-run did not render a refusal reason:\n%s", out)
	}
	if strings.ContainsRune(out, '\x1b') {
		t.Errorf("dry-run output leaked a raw ESC from a repo filename:\n%q", out)
	}
}
