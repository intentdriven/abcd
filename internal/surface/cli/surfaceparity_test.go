package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// surfaceparity_test.go — the iss-44 surface-parity detector.
//
// The claim it holds is the "wired or it isn't done" boundary, made checkable:
// a verb the binary registers is reachable from the plugin markdown, or the test
// carries an explicit scoping note saying why it is CLI-only. Three failures the
// 2026-07-29 marketplace-install test reproduced are each a case of this check
// being absent — a sub-verb tree the command file never mentions (iss-44), a
// namespace segment nobody meant to declare (iss-161), and a document the loader
// reads as a command (iss-160) — so the check is one file rather than three.
//
// It reads the real command tree and the real commands/ directory, because a
// synthetic fixture would assert only that the assertion compiles.

// pluginCommandsDir is the plugin command surface, repo-relative. It is FLAT: a
// harness maps each subdirectory of it to an extra namespace segment, so a verb
// file one level down under commands/abcd/ registers as `/abcd:abcd:<verb>` and
// every `/abcd:<verb>` the docs name is then an unknown command (iss-161).
const pluginCommandsDir = "commands"

// bareCommandFile is the command file backing the bare `/abcd` status board
// rather than a `/abcd:<verb>` surface.
const bareCommandFile = "abcd"

// cliOnlyVerbs are the binary verbs deliberately absent from the plugin surface,
// each with the scoping note that makes the absence a decision rather than a
// gap. A verb added here must be a verb a plugin user has no business invoking —
// never a verb whose command file merely has not been written yet.
var cliOnlyVerbs = map[string]string{
	"changelog":  "deterministic release-cut input consumed by `launch ship`; the plugin surface orchestrates the cut through commands/launch.md",
	"hook":       "operator-internal host adapter set, live-wired from hooks/hooks.json and invoked by the harness, never by a user",
	"rules":      "operator-internal rule injection driven by the prompt hook; its bare render is read-only diagnostics",
	"spec":       "internal spec-store tooling for the intent lifecycle; the user-facing half is commands/intent.md",
	"statusline": "harness-invoked status-line render, wired by `ahoy install` and run by the harness on every refresh with its payload on stdin, never by a user; the row it prints and the offer that wires it are documented in commands/abcd.md and commands/ahoy.md",
}

// hostDelegatedCommands are the command files with no Go verb at all: the whole
// workflow runs in the host agent, so the parity check must not read a missing
// binary verb as drift.
var hostDelegatedCommands = map[string]bool{
	"consult":           true,
	"ingest":            true,
	"prepare-this-repo": true,
}

// commandFileBodies reads every command file in the surface, keyed by verb.
func commandFileBodies(t *testing.T) map[string]string {
	t.Helper()
	dir := filepath.Join(testRepoRoot(), pluginCommandsDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the plugin command surface %s: %v", pluginCommandsDir, err)
	}
	bodies := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("reading %s/%s: %v", pluginCommandsDir, e.Name(), err)
		}
		bodies[strings.TrimSuffix(e.Name(), ".md")] = string(data)
	}
	return bodies
}

// TestPluginCommandSurfaceRegistersOnlyCommands is the iss-160 + iss-161
// detector. A harness registers every markdown file under commands/ as a slash
// command — with no frontmatter requirement and no name exemption, which
// agents/README.md registering as an agent independently shows (iss-110) — and
// maps each subdirectory to a namespace segment. So the surface is exactly what
// the directory is: one flat level, every file a command, nothing else.
func TestPluginCommandSurfaceRegistersOnlyCommands(t *testing.T) {
	dir := filepath.Join(testRepoRoot(), pluginCommandsDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the plugin command surface %s: %v", pluginCommandsDir, err)
	}
	for _, e := range entries {
		switch {
		case e.IsDir():
			t.Errorf("%s/%s/ is a subdirectory of the command surface: a harness maps it to a "+
				"namespace segment, so every verb inside registers as /abcd:%s:<verb> rather than "+
				"the documented /abcd:<verb> (iss-161). Command files live flat under %s/",
				pluginCommandsDir, e.Name(), e.Name(), pluginCommandsDir)
		case !strings.HasSuffix(e.Name(), ".md"):
			t.Errorf("%s/%s is not a markdown command file; the surface carries commands only",
				pluginCommandsDir, e.Name())
		case strings.EqualFold(e.Name(), "README.md"):
			t.Errorf("%s/README.md registers as a spurious /abcd:README command — the loader reads "+
				"every markdown file here as one (iss-160). Documentation of this directory belongs "+
				"outside the auto-discovery root, in the brief's surface registry",
				pluginCommandsDir)
		}
	}
}

// TestPluginSurfaceReachesEveryBinaryVerb is the iss-44 parity check proper:
// every verb and sub-verb the binary registers is either named by its command
// file or carries a scoping note here. `ahoy` is the instance that motivated it
// — the command file drove the bare read-only detect while the binary registered
// install, uninstall, doctor, dry-run, and identity-check, so a plugin user
// could not reach the read-only ones at all.
func TestPluginSurfaceReachesEveryBinaryVerb(t *testing.T) {
	snap, err := SurfaceSnapshot(testRepoRoot())
	if err != nil {
		t.Fatalf("SurfaceSnapshot: %v", err)
	}
	bodies := commandFileBodies(t)

	verbs := map[string]bool{}
	for _, cmd := range snap.Commands {
		parts := strings.Fields(cmd.Path)
		if len(parts) < 2 {
			continue // the root command itself
		}
		verb := parts[1]
		verbs[verb] = true

		body, hasFile := bodies[verb]
		if !hasFile {
			if cliOnlyVerbs[verb] == "" {
				t.Errorf("`%s` has no %s/%s.md and no scoping note: either write the command file or "+
					"record in cliOnlyVerbs why a plugin user has no business invoking it",
					cmd.Path, pluginCommandsDir, verb)
			}
			continue
		}
		if len(parts) < 3 {
			continue // the parent verb itself, backed by its file
		}
		if cmd.MovedTo != "" {
			// A moved spelling (itd-2609212130136102): its page names the
			// successor, never the stub, so there is nothing to reach.
			continue
		}
		sub := parts[2]
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(sub) + `\b`).MatchString(body) {
			t.Errorf("`%s` is registered by the binary but %s/%s.md never mentions `%s`: the sub-verb "+
				"is unreachable from the plugin surface (iss-44)", cmd.Path, pluginCommandsDir, verb, sub)
		}
	}

	// The mirror direction: a command file naming a verb the binary does not
	// register is a surface promising something no engine answers.
	names := make([]string, 0, len(bodies))
	for name := range bodies {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if name == bareCommandFile || verbs[name] || hostDelegatedCommands[name] {
			continue
		}
		t.Errorf("%s/%s.md has no binary verb and is not recorded as host-delegated: the plugin "+
			"surface names a command nothing answers", pluginCommandsDir, name)
	}
}

// TestPluginCommandPathRemedyIsRunnable is the iss-44 remedy instance. Every
// command file tells the reader what to do when the `abcd` binary is not on
// PATH, and the answer `make build` is wrong everywhere it appears: that target
// cross-compiles bin/abcd-<goos>-<arch>, which is not a PATH-resolvable `abcd`,
// so following the remedy leaves the reader exactly where they started. The
// runnable answer is the `go run ./cmd/abcd …` fallback the same paragraphs
// already carry.
func TestPluginCommandPathRemedyIsRunnable(t *testing.T) {
	for verb, body := range commandFileBodies(t) {
		file := pluginCommandsDir + "/" + verb + ".md"
		if strings.Contains(body, "make build") {
			t.Errorf("%s offers `make build` as a remedy: it produces bin/abcd-<goos>-<arch>, never a "+
				"PATH-resolvable `abcd`, so the instruction cannot fix what it claims to (iss-44). "+
				"Use the `go run ./cmd/abcd …` fallback", file)
		}
		if strings.Contains(body, "not on `PATH`") && !strings.Contains(body, "go run ./cmd/abcd") {
			t.Errorf("%s names the not-on-PATH case without the runnable `go run ./cmd/abcd …` "+
				"fallback", file)
		}
	}
}
