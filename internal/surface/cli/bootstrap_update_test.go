package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/update"
)

// The ruling CJ1 has the installer record the release it replaced, in the
// binary-meta it writes at the swap, and CJ1b has it say "abcd updated from X
// to Y" ONCE, when the swap completes, so the session check needs to neither
// show it nor write anything (iss-2609291942520919).

// firstLine (hooks_sessionstart_test.go) is the one line of a hook's stderr the
// harness relays (iss-208), which is why the update statement must sit there.

// TestBootstrapNewReleaseRecordsAndReportsTheUpdateOnce: a cache swap from an
// older release records that release as previous_tag and leads its notice with
// the shared update line; the next root, served from the cache, and the fast
// path that follows say nothing about an update.
func TestBootstrapNewReleaseRecordsAndReportsTheUpdateOnce(t *testing.T) {
	data := t.TempDir()
	old := []byte("#!/bin/sh\n# old release\nexit 0\n")
	fresh := []byte("#!/bin/sh\n# new release\nexit 0\n")
	seedBootstrapCache(t, data, "v9.9.8", old)
	fx := bootstrapServer(t, fresh, bootstrapManifest(fresh))
	want := update.UpdatedLine("v9.9.8", bootstrapTag)

	root := bootstrapRoot(t)
	out, code := runBootstrapWithData(t, root, data, fx, "")
	if code != 0 {
		t.Fatalf("a new release must install, got %d (output %q)", code, out)
	}
	if !strings.HasPrefix(firstLine(out), want) {
		t.Errorf("the swap's first output line must lead with %q; got %q", want, firstLine(out))
	}
	if n := strings.Count(out, want); n != 1 {
		t.Errorf("the update line must be printed exactly once, got %d in %q", n, out)
	}
	meta := cacheMetaValues(t, data)
	if meta["previous_tag"] != "v9.9.8" {
		t.Errorf("the cache meta must record the replaced release as previous_tag; got %v", meta)
	}
	if _, ok := meta["transition_unseen"]; ok {
		t.Errorf("a run whose output is relayed must not flag the transition as unseen; got %v", meta)
	}

	// The fast path of the same root: nothing at all.
	if out, _ := runBootstrapWithData(t, root, data, fx, ""); strings.Contains(out, "updated from") {
		t.Errorf("the fast path must not repeat the update line; got %q", out)
	}
	// A second root served from the now-current cache: no swap, no line.
	out, code = runBootstrapWithData(t, bootstrapRootNamed(t, strings.Repeat("a", 40)), data, fx, "")
	if code != 0 {
		t.Fatalf("a cache hit must install, got %d (output %q)", code, out)
	}
	if strings.Contains(out, "updated from") {
		t.Errorf("a cache hit swaps no release and must not report an update; got %q", out)
	}
}

// TestBootstrapUnseenSwapFlagsTheTransition: the salvage runs in the
// per-prompt, per-command and pre-compaction hooks discard this script's output
// (hooks/hooks.json passes --unseen there), so a swap made there says so in the
// record, which is what lets the next session start show it once.
func TestBootstrapUnseenSwapFlagsTheTransition(t *testing.T) {
	data := t.TempDir()
	old := []byte("#!/bin/sh\n# old release\nexit 0\n")
	fresh := []byte("#!/bin/sh\n# new release\nexit 0\n")
	seedBootstrapCache(t, data, "v9.9.8", old)
	fx := bootstrapServer(t, fresh, bootstrapManifest(fresh))
	bootstrapRequires(t)
	script := bootstrapFixtureScript(t, fx.base)
	wrapper := filepath.Join(t.TempDir(), "unseen.sh")
	if err := os.WriteFile(wrapper, []byte("#!/bin/sh\nexec '"+script+"' --unseen\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, code := runScript(t, wrapper, bootstrapRoot(t), append(fx.env(), "CLAUDE_PLUGIN_DATA="+data), "")
	if code != 0 {
		t.Fatalf("a new release must install, got %d (output %q)", code, out)
	}
	meta := cacheMetaValues(t, data)
	if meta["previous_tag"] != "v9.9.8" || meta["transition_unseen"] != "yes" {
		t.Errorf("an unseen swap must record previous_tag and transition_unseen=yes; got %v", meta)
	}
}

// TestBootstrapDegradedSwapReportsTheReplacedRelease: without a data dir the
// per-root fetch knows the replaced release only from the root's own record,
// which survives when the binary beside it was removed.
func TestBootstrapDegradedSwapReportsTheReplacedRelease(t *testing.T) {
	root := bootstrapRoot(t)
	prior := "release_tag=v9.9.8\nrelease_sha=" + bootstrapRelease + "\nbinary_sha256=" + strings.Repeat("0", 64) + "\nfetched_at=2026-08-01T00:00:00Z\n"
	if err := os.WriteFile(filepath.Join(root, ".binary-meta"), []byte(prior), 0o644); err != nil {
		t.Fatal(err)
	}
	body := []byte("#!/bin/sh\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))
	out, code := runBootstrap(t, root, fx, "")
	if code != 0 {
		t.Fatalf("the degraded install must succeed, got %d (output %q)", code, out)
	}
	if want := update.UpdatedLine("v9.9.8", bootstrapTag); !strings.HasPrefix(firstLine(out), want) {
		t.Errorf("the degraded swap must lead with %q; got %q", want, firstLine(out))
	}
	if got := metaValues(t, root)["previous_tag"]; got != "v9.9.8" {
		t.Errorf("the root meta must record previous_tag=v9.9.8; got %q", got)
	}
}

// TestBootstrapFirstInstallReportsNoUpdate: a fresh root with no record of an
// earlier release is an install, not an update, and says nothing of one.
func TestBootstrapFirstInstallReportsNoUpdate(t *testing.T) {
	root := bootstrapRoot(t)
	body := []byte("#!/bin/sh\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))
	out, code := runBootstrapWithData(t, root, t.TempDir(), fx, "")
	if code != 0 {
		t.Fatalf("a first install must succeed, got %d (output %q)", code, out)
	}
	if strings.Contains(out, "updated from") {
		t.Errorf("a first install must not report an update; got %q", out)
	}
}

// TestBootstrapUpdateLineIsTheSharedWording: one wording, one place. The script
// cannot import update.UpdatedFormat, so its printf literal is held to it.
func TestBootstrapUpdateLineIsTheSharedWording(t *testing.T) {
	body := mustReadFile(t, bootstrapScript(t))
	if want := "printf '" + update.UpdatedFormat + "'"; strings.Count(body, want) != 1 {
		t.Errorf("hooks/bootstrap.sh must carry the update line exactly once as %s", want)
	}
}

// TestDiscardedSalvageRunsPassUnseen: every hook entry that runs the bootstrap
// with its output thrown away tells it so, and the one entry that relays the
// output (SessionStart) does not, so the update line reaches a reader exactly
// once whichever entry performed the swap.
func TestDiscardedSalvageRunsPassUnseen(t *testing.T) {
	doc := decodedHooksManifest(t)
	for event, entries := range doc.Hooks {
		for _, entry := range entries {
			for _, h := range entry.Hooks {
				if !strings.Contains(h.Command, "hooks/bootstrap.sh") {
					continue
				}
				discarded := strings.Contains(h.Command, `/hooks/bootstrap.sh" >/dev/null 2>&1`) ||
					strings.Contains(h.Command, `/hooks/bootstrap.sh" --unseen >/dev/null 2>&1`)
				unseen := strings.Contains(h.Command, `bootstrap.sh" --unseen`)
				switch {
				case event == "SessionStart" && unseen:
					t.Errorf("SessionStart relays the bootstrap's stderr and must not pass --unseen")
				case event != "SessionStart" && discarded && !unseen:
					t.Errorf("%s discards the bootstrap's output and must pass --unseen", event)
				}
			}
		}
	}
}
