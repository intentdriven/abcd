package cli

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
)

// hookPlaneParents are the parents a host hook reaches. On the hook plane an exit
// status is not a diagnostic, it is an INSTRUCTION: the host reads 2 as "block
// this action". `guard hook` runs on PreToolUse before every shell command, and
// `hook prompt-router` runs on UserPromptSubmit before every prompt.
var hookPlaneParents = []string{"guard", "hook"}

// TestHookPlaneParentsFailOpenOnUnknownSubverb is iss-267. iss-266 made every
// parent refuse an unknown sub-verb at cobra's usage status, exit 2 — correct for
// a terminal, wrong here. `guard hook`'s contract (spc-16, itd-103 AC 1) is
// fail-open-loud: exit 2 is reserved for "the guard decided to block", and every
// path that is NOT a decision exits 1 so the command still runs and the warning
// is still seen. An unknown sub-verb is not a decision, so refusing it at 2 makes
// abcd claim a hazard verdict it never reached, and blocks every shell command in
// the session. The PreToolUse wrapper cannot catch this: it treats 2 as a
// recognised code, so its "FAILED TO RUN … UNGUARDED" net never fires.
//
// The refusal must still be LOUD and non-zero — iss-266's guarantee that a
// mistyped sub-verb never reads as success is unchanged. Only the code moves,
// from the host's blocking status to its non-blocking one.
func TestHookPlaneParentsFailOpenOnUnknownSubverb(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, parent := range hookPlaneParents {
		t.Run(parent, func(t *testing.T) {
			var out, errOut strings.Builder
			code := Run([]string{parent, "zzz-not-a-subverb"}, &out, &errOut)
			if code == 0 {
				t.Fatalf("abcd %s zzz-not-a-subverb exited 0; an unknown sub-verb must still refuse", parent)
			}
			if code == 2 {
				t.Fatalf("abcd %s zzz-not-a-subverb exited 2 — the host's BLOCKING status. "+
					"An unknown sub-verb is not a decision to block; it must fail open at exit 1 (iss-267).\nstderr:\n%s",
					parent, errOut.String())
			}
			if code != 1 {
				t.Fatalf("abcd %s zzz-not-a-subverb exited %d, want 1 (loud, non-blocking)", parent, code)
			}
			if errOut.Len() == 0 {
				t.Fatalf("abcd %s zzz-not-a-subverb said nothing on stderr; failing open silently is the "+
					"other half of the contract this must not break", parent)
			}
		})
	}
}

// hookReachablePaths returns every command path a host hook actually invokes,
// read from hooks/hooks.json, plus each parent on the way to it. Derived rather
// than hand-listed so a manifest change is picked up without anyone remembering.
func hookReachablePaths(t *testing.T) [][]string {
	t.Helper()
	var out [][]string
	seen := map[string]bool{}
	for _, path := range manifestInvocations(t) {
		for i := 1; i <= len(path); i++ {
			key := strings.Join(path[:i], " ")
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, append([]string{}, path[:i]...))
		}
	}
	return out
}

// TestHookPlaneFailsOpenOnEveryUsageError is iss-269. iss-267 fixed the
// unknown-SUB-VERB path on the two parents; the same argument covers the rest of
// the usage surface, and it was left exiting 2 — the host's BLOCKING status.
//
// A stray positional on a leaf (`guard hook zzz`) and an unknown flag anywhere on
// the plane (`abcd guard hook --strict` against a binary that predates the flag)
// are the same failure as the sub-verb case: abcd cannot answer, so it must say so
// without blocking. The flag case is the one a future manifest change makes
// reachable — hooks.json ships with the plugin clone while the binary comes from
// the latest release, so the manifest runs ahead.
func TestHookPlaneFailsOpenOnEveryUsageError(t *testing.T) {
	// Read the manifest BEFORE moving off the package directory: the path to it is
	// relative, and t.Chdir would break it.
	paths := hookReachablePaths(t)
	t.Chdir(t.TempDir())
	for _, path := range paths {
		name := strings.Join(path, " ")
		for _, stray := range []string{"zzz-not-a-subverb", "--zzz-not-a-flag"} {
			t.Run(strings.ReplaceAll(name, " ", "_")+"/"+strings.TrimLeft(stray, "-"), func(t *testing.T) {
				args := append(append([]string{}, path...), stray)
				var out, errOut strings.Builder
				code := Run(args, &out, &errOut)
				if code == 0 {
					t.Fatalf("abcd %s exited 0; a usage error must still refuse", strings.Join(args, " "))
				}
				if code == 2 {
					t.Fatalf("abcd %s exited 2 — the host's BLOCKING status. On the hook plane a usage "+
						"error abcd cannot answer must fail open at exit 1 (iss-269).\nstderr:\n%s",
						strings.Join(args, " "), errOut.String())
				}
				if code != 1 {
					t.Fatalf("abcd %s exited %d, want 1 (loud, non-blocking)", strings.Join(args, " "), code)
				}
				if errOut.Len() == 0 {
					t.Fatalf("abcd %s said nothing on stderr", strings.Join(args, " "))
				}
			})
		}
	}
}

// TestHookPlaneFlagGroupRefusalFailsOpen is iss-2609251755278758. A flag-group
// refusal (two flags of a mutually exclusive group set at once) comes from
// neither FlagErrorFunc nor Args: markUsageErrorsExitTwo checks the groups in
// the PreRunE it installs on every command and codes the refusal 2, and the
// hook plane reset the other two and not that one. No hook command declares a
// group today, so the test declares one on each hook-reachable command, as a
// future flag would, and asks that the refusal still fail open at exit 1.
func TestHookPlaneFlagGroupRefusalFailsOpen(t *testing.T) {
	paths := hookReachablePaths(t)
	t.Chdir(t.TempDir())
	for _, path := range paths {
		name := strings.Join(path, " ")
		t.Run(strings.ReplaceAll(name, " ", "_"), func(t *testing.T) {
			root := NewRootCommand()
			cmd := findByPath(root, path)
			if cmd == nil {
				t.Fatalf("abcd %s is not in the command tree", name)
			}
			if !cmd.Runnable() {
				t.Skipf("abcd %s runs nothing, so no PreRunE of it runs", name)
			}
			cmd.Flags().Bool("zz-a", false, "")
			cmd.Flags().Bool("zz-b", false, "")
			cmd.MarkFlagsMutuallyExclusive("zz-a", "zz-b")
			root.SetArgs(append(append([]string{}, path...), "--zz-a", "--zz-b"))
			root.SetIn(strings.NewReader(""))
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			err := root.Execute()
			if err == nil {
				t.Fatalf("abcd %s --zz-a --zz-b ran; a flag-group violation must still refuse", name)
			}
			code := 1
			var coded interface{ ExitCode() int }
			if errors.As(err, &coded) {
				code = coded.ExitCode()
			}
			if code != 1 {
				t.Fatalf("abcd %s --zz-a --zz-b exited %d, want 1: on the hook plane a usage error abcd "+
					"cannot answer must not be the host's blocking status (iss-269)\n%v", name, code, err)
			}
			if !strings.Contains(err.Error(), "none of the others can be") && !strings.Contains(err.Error(), "zz-a") {
				t.Fatalf("the refusal no longer names the group it refused: %v", err)
			}
		})
	}
}

// TestGuardCheckKeepsItsFaultExit pins the boundary of the concession above.
// `guard check` is the human/scriptable verb and its contract is the OPPOSITE:
// a fault exits 2 so a caller never reads silence as clearance (17-guard.md). It
// sits under the same parent as `guard hook` and must not be swept along.
func TestGuardCheckKeepsItsFaultExit(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, stray := range []string{"zzz-not-a-subverb", "--zzz-not-a-flag"} {
		var out, errOut strings.Builder
		if code := Run([]string{"guard", "check", stray}, &out, &errOut); code != 2 {
			t.Errorf("abcd guard check %s exited %d, want 2 — a script must not read a usage "+
				"fault as anything softer", stray, code)
		}
	}
}

// binaryInvocationRe finds each place hooks.json runs the abcd binary and
// captures the sub-verb path that follows, stopping at the first shell
// metacharacter. The alternatives are the three spellings the wrappers use: the
// resolved-binary variable, the plugin-root variable, and the full plugin-root
// path SessionStart uses directly.
var binaryInvocationRe = regexp.MustCompile(`(?:"\$g"|"\$r/abcd"|"\$CLAUDE_PLUGIN_ROOT/abcd")((?:[[:space:]]+[a-z][a-z0-9-]*)+)`)

// TestHooksManifestNamesLiveSubverbs pins hooks/hooks.json against the live
// command tree.
//
// The skew this defends runs in ONE direction: a user's plugin clone updates by
// `git pull` while the binary updates by release download, so the manifest is the
// half that runs AHEAD. That decides the doctrine. An alias added to a new binary
// does nothing for a user whose binary is old, so the remedy is not "rename and
// alias the manifest across" — it is the reverse: **the manifest's spellings are
// frozen, and a rename is absorbed by an alias in the binary**. Editing
// hooks.json to a new spelling is the move that breaks every older binary.
//
// So this test asks only that every spelling the manifest names still resolves in
// this binary — through an alias just as happily as through a primary name. It
// fails the build in the same change that would strand a skewed pair, instead of
// in a user's session.
// manifestInvocations decodes hooks/hooks.json and returns every abcd invocation
// path it contains. Shared by the tests that pin the manifest against the live
// tree and that hold the hook plane's exit contract.
func manifestInvocations(t *testing.T) [][]string {
	t.Helper()
	raw, err := os.ReadFile("../../../hooks/hooks.json")
	if err != nil {
		t.Fatalf("reading the hooks manifest: %v", err)
	}
	// Decode rather than scan the raw bytes: the command is a JSON string, so the
	// shell quoting is escaped on disk and a regex over the file would match the
	// escaped spelling instead of the command the host actually runs.
	var manifest struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("the hooks manifest is not valid JSON: %v", err)
	}
	var events, commands []string
	for event, entries := range manifest.Hooks {
		for _, entry := range entries {
			for _, h := range entry.Hooks {
				events = append(events, event)
				commands = append(commands, resolveHookCommand(t, h.Command))
			}
		}
	}
	if len(commands) == 0 {
		t.Fatal("the hooks manifest decoded to zero commands — the struct no longer matches its shape")
	}
	var out [][]string
	seen := map[string]bool{}
	// Scan EACH command separately and require every one to yield an invocation.
	// Joining them and checking the total only catches a scan that fails
	// completely; a quoting drift in one command would hide that one invocation
	// while the others kept the test green, leaving it unpinned forever.
	for i, command := range commands {
		matches := binaryInvocationRe.FindAllStringSubmatch(command, -1)
		if len(matches) == 0 {
			t.Errorf("the %s hook command contains no recognisable abcd invocation — "+
				"the scan is broken, not the manifest. Every hook command runs the binary; if this one "+
				"spells the invocation a new way, teach binaryInvocationRe that spelling rather than "+
				"leaving the command unchecked.", events[i])
			continue
		}
		for _, m := range matches {
			path := strings.Fields(m[1])
			if key := strings.Join(path, " "); !seen[key] {
				seen[key] = true
				out = append(out, path)
			}
		}
	}
	return out
}

// TestHooksManifestNamesLiveSubverbs pins hooks/hooks.json against the live
// command tree.
//
// The skew this defends runs in ONE direction: a user's plugin clone updates by
// `git pull` while the binary updates by release download, so the manifest is the
// half that runs AHEAD. That decides the doctrine. An alias added to a new binary
// does nothing for a user whose binary is old, so the remedy is not "rename and
// alias the manifest across" — it is the reverse: the manifest's spellings are
// frozen, and a rename is absorbed by an alias in the binary. Editing hooks.json
// to a new spelling is the move that breaks every older binary.
func TestHooksManifestNamesLiveSubverbs(t *testing.T) {
	root := NewRootCommand()
	for _, path := range manifestInvocations(t) {
		joined := strings.Join(path, " ")
		// Fully-consumed path, NOT a name comparison. Find resolves aliases but
		// Name() returns the primary, so comparing names would reject a rename
		// that carries a compatibility alias — the one remedy that survives the
		// skew. Leftover args mean Find stopped at a parent, the prefix case.
		cmd, rest, err := root.Find(path)
		if err != nil || cmd == nil || cmd == root || len(rest) > 0 {
			t.Errorf("hooks/hooks.json invokes `abcd %s`, which this binary does not answer.\n"+
				"The manifest runs ahead of the binary on a user's machine (clone pulls, binary lags), "+
				"so its spellings are frozen: absorb the rename with an Aliases entry on the renamed "+
				"command and leave hooks.json alone — editing it to the new spelling is what strands "+
				"every older binary (iss-267).", joined)
		}
	}
}
