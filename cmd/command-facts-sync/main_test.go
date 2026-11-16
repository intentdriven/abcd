package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/intentdriven/abcd/internal/surface/cli"
)

// main_test.go — the generator of the command facts' derived copies
// (spc-2610301431075964 step 3), run over a fixture tree of facts and a
// checkout of its own, so each rule of the copies is held on its own: what
// each JSON file holds, where each block stands on a page, the one-line block,
// the page's own rung, the stale copy -check names, the orphan file the sync
// removes, and each fault it refuses rather than guess past. The drift of the
// committed copies against the live tree is held in
// internal/surface/cli/commandfacts_drift_test.go.

// fixtureFacts is a small tree: the root with its persistent --json; a verb
// with a sub-command that reads a filled-in input and has an example; a verb
// with no flag and no input; a visible verb with no page; and a hidden verb.
func fixtureFacts() []surface.Facts {
	return surface.CanonicalFacts([]surface.Facts{
		{Path: "abcd", Usage: "abcd [flags]", Subcommands: []string{"alpha", "beta", "quiet"},
			Flags: []surface.FlagFact{{Name: "json", Type: surface.FlagBool, Persistent: true}}},
		{Path: "abcd alpha", Usage: "abcd alpha [flags]", Subcommands: []string{"record"}},
		{Path: "abcd alpha record", Usage: "abcd alpha record --verdict-json <file> [flags]",
			Flags:   []surface.FlagFact{{Name: "verdict-json", Type: surface.FlagString, Required: true}},
			Inputs:  []surface.InputFact{{Flag: "verdict-json", Kind: surface.InputJSON, Name: "alpha verdict", SchemaVersion: 3, VersionIn: surface.VersionField}},
			Example: "abcd alpha record --verdict-json verdict.json"},
		{Path: "abcd alpha secret", Hidden: true, Usage: "abcd alpha secret"},
		{Path: "abcd beta", Usage: "abcd beta [flags]",
			Flags: []surface.FlagFact{{Name: "dry-run", Type: surface.FlagBool}}},
		{Path: "abcd quiet", Usage: "abcd quiet"},
		{Path: "abcd hook", Hidden: true, Usage: "abcd hook [flags]", Subcommands: []string{"tick"}},
		{Path: "abcd hook tick", Usage: "abcd hook tick",
			Flags: []surface.FlagFact{{Name: "now", Type: surface.FlagBool}}},
	})
}

// pluginPage is a page as the plugin ships one: frontmatter, the H1 heading, a
// fenced plugin-root invocation and the closing user-input line.
func pluginPage(verb string) string {
	return "---\nname: " + verb + "\nblock: people\n---\n\n# `/abcd:" + verb + "` — the " + verb + " page\n\n" +
		"What the page says first.\n\n```bash\n\"${CLAUDE_PLUGIN_ROOT}/abcd\" " + verb + " --json\n```\n\n" +
		"**User input:** $ARGUMENTS\n"
}

// checkout writes pages into a fresh root, keyed by verb.
func checkout(t *testing.T, pages map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, cli.CommandPagesDir), 0o755); err != nil {
		t.Fatal(err)
	}
	for verb, page := range pages {
		write(t, root, cli.CommandPagesDir+"/"+verb+".md", page)
	}
	return root
}

func write(t *testing.T, root, rel, data string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func exists(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

func factsFile(verb string) string { return cli.CommandFactsDir + "/" + verb + ".json" }

// A sync writes one JSON file per visible top-level verb, its visible subtree
// in canonical order, and one for the root alone; a verb with no page gets its
// file and no page; a hidden verb, and a hidden sub-command, are in no file.
// A second sync finds nothing to do.
func TestASyncWritesOneFilePerVisibleVerbAndThenFindsNothingToDo(t *testing.T) {
	root := checkout(t, map[string]string{"abcd": pluginPage("abcd"), "alpha": pluginPage("alpha"), "quiet": pluginPage("quiet")})
	facts := fixtureFacts()
	if err := run(root, facts, false); err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"abcd":  {"abcd"},
		"alpha": {"abcd alpha", "abcd alpha record"},
		"beta":  {"abcd beta"},
		"quiet": {"abcd quiet"},
	}
	for verb, paths := range want {
		var doc cli.CommandFactsDocument
		if err := json.Unmarshal([]byte(read(t, root, factsFile(verb))), &doc); err != nil {
			t.Fatalf("%s: %v", factsFile(verb), err)
		}
		if doc.FormatVersion != surface.FactsFormatVersion {
			t.Errorf("%s: format_version %d, want %d", factsFile(verb), doc.FormatVersion, surface.FactsFormatVersion)
		}
		var got []string
		for _, f := range doc.Commands {
			got = append(got, f.Path)
		}
		if strings.Join(got, ",") != strings.Join(paths, ",") {
			t.Errorf("%s holds %v, want %v", factsFile(verb), got, paths)
		}
	}
	if exists(root, factsFile("hook")) {
		t.Errorf("the hidden verb's file %s was written", factsFile("hook"))
	}
	if exists(root, cli.CommandPagesDir+"/beta.md") {
		t.Error("the sync created a page for a verb that has none")
	}
	alpha := read(t, root, factsFile("alpha"))
	if !strings.Contains(alpha, `"schema_version": 3`) {
		t.Errorf("the file does not carry the input's schema version:\n%s", alpha)
	}
	if !strings.Contains(alpha, `--verdict-json <file>`) {
		t.Errorf("the file escapes the usage line's angle brackets:\n%s", alpha)
	}

	changes, err := plan(root, facts)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 0 {
		t.Errorf("a second sync would still change %d files: %v", len(changes), changes)
	}
}

// The ask-the-binary block stands straight after the H1 heading, naming the
// page's verb in the page's own rung, and the command-facts block stands at
// the end, above the closing user-input line, with each command's usage line,
// its input and schema version, and its example. The hand-written text is
// kept as it was.
func TestTheBlocksStandAfterTheHeadingAndAboveTheUserInput(t *testing.T) {
	root := checkout(t, map[string]string{"alpha": pluginPage("alpha")})
	if err := run(root, fixtureFacts(), false); err != nil {
		t.Fatal(err)
	}
	page := read(t, root, cli.CommandPagesDir+"/alpha.md")
	heading := "# `/abcd:alpha` — the alpha page\n\n"
	ask := cli.AskTheBinaryBlock("alpha", `"${CLAUDE_PLUGIN_ROOT}/abcd"`)
	if !strings.Contains(page, heading+ask+"\n\nWhat the page says first.\n") {
		t.Errorf("the ask-the-binary block does not stand straight after the heading:\n%s", page)
	}
	if !strings.Contains(ask, "run `\"${CLAUDE_PLUGIN_ROOT}/abcd\" explain alpha`") {
		t.Errorf("the ask-the-binary block does not name the page's verb in its rung:\n%s", ask)
	}
	if !strings.HasSuffix(page, cli.GeneratedEnd+"\n\n**User input:** $ARGUMENTS\n") {
		t.Errorf("the command-facts block does not stand just above the user-input line:\n%s", page)
	}
	start := strings.Index(page, cli.CommandFactsBegin)
	if start < 0 {
		t.Fatalf("the page carries no command-facts block:\n%s", page)
	}
	block := page[start:]
	for _, want := range []string{
		cli.CommandFactsBegin + "\n**Command facts, format version 1.** Where this block and `\"${CLAUDE_PLUGIN_ROOT}/abcd\" explain alpha` from the running binary disagree, follow the binary.\n",
		"- `abcd alpha [flags]`\n",
		"- `abcd alpha record --verdict-json <file> [flags]`\n",
		"  - Reads `--verdict-json`, the alpha verdict, at schema_version 3; `\"${CLAUDE_PLUGIN_ROOT}/abcd\" explain alpha record --input verdict-json` prints its shape.\n",
		"  - An example call is `abcd alpha record --verdict-json verdict.json`.\n",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("the command-facts block does not carry %q:\n%s", want, block)
		}
	}
	if strings.Contains(block, "secret") {
		t.Errorf("the block names a hidden sub-command:\n%s", block)
	}
}

// A verb whose commands declare no flag and read no input gets a one-line
// block; a page that never names the plugin-root binary is sent to the binary
// on PATH; a page with no closing user-input line takes the block at its end.
func TestAQuietVerbGetsAOneLineBlockInThePagesOwnRung(t *testing.T) {
	page := "---\nname: quiet\n---\n\n# Quiet\n\nRun `abcd quiet`.\n"
	root := checkout(t, map[string]string{"quiet": page})
	if err := run(root, fixtureFacts(), false); err != nil {
		t.Fatal(err)
	}
	got := read(t, root, cli.CommandPagesDir+"/quiet.md")
	want := "---\nname: quiet\n---\n\n# Quiet\n\n" +
		cli.AskTheBinaryBegin + "\nBefore you fill in an input for this command, run `abcd explain quiet`, and where its answer and this page disagree, follow the binary, since this page may be older than it.\n" + cli.GeneratedEnd + "\n\n" +
		"Run `abcd quiet`.\n\n" +
		cli.CommandFactsBegin + "\n**Command facts, format version 1.** Where this block and `abcd explain quiet` from the running binary disagree, follow the binary.\n" + cli.GeneratedEnd + "\n"
	if got != want {
		t.Errorf("the page reads\n%s\nwant\n%s", got, want)
	}
}

// -check names every stale copy and writes nothing: a missing file, an edited
// block, and a file for a verb the tree no longer has. The sync then rewrites
// the block where it stands and removes the orphan file.
func TestCheckNamesEachStaleCopyAndTheSyncMendsIt(t *testing.T) {
	root := checkout(t, map[string]string{"alpha": pluginPage("alpha")})
	facts := fixtureFacts()
	if err := run(root, facts, false); err != nil {
		t.Fatal(err)
	}
	synced := read(t, root, cli.CommandPagesDir+"/alpha.md")
	edited := strings.Replace(synced, "An example call is", "An example is", 1)
	write(t, root, cli.CommandPagesDir+"/alpha.md", edited)
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(factsFile("beta")))); err != nil {
		t.Fatal(err)
	}
	write(t, root, factsFile("gone"), "{}\n")

	err := run(root, facts, true)
	if !errors.Is(err, errOutOfDate) {
		t.Fatalf("-check returned %v, want the out-of-date error", err)
	}
	for _, rel := range []string{cli.CommandPagesDir + "/alpha.md", factsFile("beta"), factsFile("gone")} {
		if !strings.Contains(err.Error(), rel) {
			t.Errorf("-check does not name %s: %v", rel, err)
		}
	}
	if read(t, root, cli.CommandPagesDir+"/alpha.md") != edited || exists(root, factsFile("beta")) || !exists(root, factsFile("gone")) {
		t.Error("-check wrote to the checkout")
	}

	if err := run(root, facts, false); err != nil {
		t.Fatal(err)
	}
	if got := read(t, root, cli.CommandPagesDir+"/alpha.md"); got != synced {
		t.Errorf("the sync did not rewrite the edited block where it stands:\n%s", got)
	}
	if !exists(root, factsFile("beta")) || exists(root, factsFile("gone")) {
		t.Error("the sync did not write the missing file or did not remove the orphan")
	}
}

// Every fault is refused and nothing is written: a page with no H1 heading, a
// block opened twice, a block never closed, a block out of its place, and a
// page for a verb the tree has no visible command for.
func TestTheSyncRefusesWhatItWouldHaveToGuess(t *testing.T) {
	alpha := pluginPage("alpha")
	ask := cli.AskTheBinaryBlock("alpha", `"${CLAUDE_PLUGIN_ROOT}/abcd"`)
	facts := cli.CommandFactsBlock(fixtureFacts(), "alpha", `"${CLAUDE_PLUGIN_ROOT}/abcd"`)
	for _, c := range []struct {
		name, verb, page, want string
	}{
		{"no heading", "alpha", "---\nname: alpha\n---\n\nNo heading here.\n", "no H1 heading"},
		{"opened twice", "alpha", strings.Replace(alpha, "What the page says first.", ask+"\n\n"+ask, 1), "it must carry one"},
		{"never closed", "alpha", strings.Replace(alpha, "What the page says first.", cli.CommandFactsBegin+"\nleft open", 1), "never closes it"},
		{"ask out of place", "alpha", strings.Replace(alpha, "**User input:**", ask+"\n\n**User input:**", 1), "away from its place"},
		{"facts out of place", "alpha", strings.Replace(alpha, "What the page says first.", "What the page says first.\n\n"+facts, 1), "away from its place"},
		{"hidden verb", "hook", pluginPage("hook"), "names no visible command"},
		{"no such verb", "gamma", pluginPage("gamma"), "names no visible command"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := checkout(t, map[string]string{c.verb: c.page})
			err := run(root, fixtureFacts(), false)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v, want a fault saying %q", err, c.want)
			}
			if !strings.Contains(err.Error(), cli.CommandPagesDir+"/"+c.verb+".md") {
				t.Errorf("the fault does not name the page: %v", err)
			}
			if read(t, root, cli.CommandPagesDir+"/"+c.verb+".md") != c.page || exists(root, cli.CommandFactsDir) {
				t.Error("a refused sync wrote to the checkout")
			}
		})
	}
}
