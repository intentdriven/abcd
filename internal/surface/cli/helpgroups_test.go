package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/spf13/cobra"
)

// helpgroups_test.go holds itd-146: `abcd --help` lists the person's verbs under
// labelled groups, `--help --agent` adds the agents-and-hosts block, and every
// placement is gated. Each test reads the real command tree, because the claim is
// about the tree that ships, not about a fixture.

// executedHelp runs `abcd <args>` on a fresh tree and returns the output and the
// tree AFTER execution: cobra attaches its `help` and `completion` commands only
// while executing, and both are verbs the help must file.
func executedHelp(t *testing.T, args ...string) (string, *cobra.Command) {
	t.Helper()
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatalf("abcd %v: %v\n%s", args, err, out.String())
	}
	return out.String(), root
}

// helpEntryRe reads one listed verb: two spaces of indent, the verb's path (one
// or more words), then at least two spaces of padding before its summary.
var helpEntryRe = regexp.MustCompile(`^  ([a-z][a-z-]*(?: [a-z][a-z-]*)*)  +\S`)

// helpSections splits a rendered help into its titled sections, keeping only the
// entry lines under each title and stopping at the flags.
func helpSections(help string) (titles []string, entries map[string][]string) {
	entries = map[string][]string{}
	current := ""
	for _, line := range strings.Split(help, "\n") {
		if line == "Flags:" {
			break
		}
		if strings.HasSuffix(line, ":") && !strings.HasPrefix(line, " ") {
			current = line
			titles = append(titles, line)
			continue
		}
		if m := helpEntryRe.FindStringSubmatch(line); m != nil && current != "" && current != "Usage:" {
			entries[current] = append(entries[current], m[1])
		}
	}
	return titles, entries
}

var peopleGroupTitles = []string{"Set-up:", "Records:", "Checks:", "Portability:", "Release:"}

// TestRootHelpListsThePersonsGroups is criterion 1: the default help shows the
// person's verbs under the five labelled groups, in order, with the one line
// above them that says `--agent` expands the list — and shows no agent entry.
func TestRootHelpListsThePersonsGroups(t *testing.T) {
	help, _ := executedHelp(t, "--help")

	titles, entries := helpSections(help)
	var groups []string
	for _, title := range titles {
		if title != "Usage:" {
			groups = append(groups, title)
		}
	}
	if strings.Join(groups, " ") != strings.Join(peopleGroupTitles, " ") {
		t.Fatalf("default help sections = %v, want exactly the person's groups %v\n%s", groups, peopleGroupTitles, help)
	}

	expand := strings.Index(help, helpExpandLine)
	first := strings.Index(help, "\nSet-up:\n")
	if expand < 0 || first < 0 || expand > first {
		t.Fatalf("the line saying --agent expands the list must sit above the first group\n%s", help)
	}

	for group, want := range map[string][]string{
		"Records:":     {"capture", "decide", "intent", "memory", "spec"},
		"Checks:":      {"lint"},
		"Portability:": {"disembark", "embark"},
		"Release:":     {"launch"},
	} {
		if strings.Join(entries[group], " ") != strings.Join(want, " ") {
			t.Errorf("%s lists %v, want %v", group, entries[group], want)
		}
	}
	for _, agentVerb := range []string{"implement", "statusline", "changelog", "mode", "reading", "history"} {
		for _, group := range peopleGroupTitles {
			for _, got := range entries[group] {
				if got == agentVerb {
					t.Errorf("%s is an agents-and-hosts verb but the default help lists it under %s", agentVerb, group)
				}
			}
		}
	}
}

// TestRootHelpAgentRendersBothBlocks is criteria 2 and 5 on the help side:
// `--help --agent` renders the person's groups and then the agents-and-hosts
// block; every visible verb is in exactly one block; the three sub-verbs the
// product thinker placed in the agent block are listed there; and each agent line
// names a page that exists and talks about the entry.
func TestRootHelpAgentRendersBothBlocks(t *testing.T) {
	help, root := executedHelp(t, "--help", "--agent")

	people := strings.Index(help, "\n"+helpPeopleTitle+"\n")
	agents := strings.Index(help, "\n"+helpAgentsTitle+"\n")
	if people < 0 || agents < 0 || people > agents {
		t.Fatalf("want the people block and then the agents-and-hosts block\n%s", help)
	}

	_, entries := helpSections(help)
	seen := map[string][]string{}
	for title, names := range entries {
		for _, n := range names {
			seen[n] = append(seen[n], title)
		}
	}
	for _, sub := range root.Commands() {
		if !sub.IsAvailableCommand() && sub.Name() != "help" {
			continue
		}
		if got := seen[sub.Name()]; len(got) != 1 {
			t.Errorf("visible verb %q is listed under %v, want exactly one section", sub.Name(), got)
		}
	}
	for _, sub := range []string{"guard hook", "intent audit ingest", "ideate record"} {
		if got := seen[sub]; len(got) != 1 || got[0] != helpAgentsTitle {
			t.Errorf("%q is listed under %v, want the agents-and-hosts block alone", sub, got)
		}
	}

	var agentLines int
	inAgents := false
	for _, line := range strings.Split(help, "\n") {
		switch {
		case line == helpAgentsTitle:
			inAgents = true
			continue
		case line == "Flags:":
			inAgents = false
		}
		m := helpEntryRe.FindStringSubmatch(line)
		if !inAgents || m == nil {
			continue
		}
		agentLines++
		page := pageNamed(line)
		if page == "" {
			t.Errorf("agent line names no page: %q", line)
			continue
		}
		body, err := os.ReadFile(filepath.Join(testRepoRoot(), filepath.FromSlash(page)))
		if err != nil {
			t.Errorf("%q names %s, which does not exist: %v", m[1], page, err)
			continue
		}
		if !strings.Contains(string(body), m[1]) {
			t.Errorf("%q names %s, which never mentions it", m[1], page)
		}
	}
	if agentLines == 0 {
		t.Fatalf("the agents-and-hosts block lists nothing\n%s", help)
	}
}

// pageNamed returns the page an agent line ends with, or "".
func pageNamed(line string) string {
	m := regexp.MustCompile(`\(read (commands/[a-z-]+\.md)\)$`).FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	return m[1]
}

// TestEveryVisibleVerbHasAGroup is criterion 3's gate on the live tree: a
// visible top-level verb registered with no group — or with a group the root
// does not declare — fails here, naming it. Hidden verbs are exempt: they never
// render, so a group would be a claim about nothing.
func TestEveryVisibleVerbHasAGroup(t *testing.T) {
	_, root := executedHelp(t, "--help")
	if missing := ungroupedVerbs(root); len(missing) != 0 {
		t.Fatalf("visible verbs with no help group: %v — place each in helpPlacements (helpgroups.go)", missing)
	}
}

// TestUngroupedVerbIsNamed proves the gate above can fail: a verb registered
// with no group is named, a hidden one is not, and a verb whose group the root
// never declared is named too.
func TestUngroupedVerbIsNamed(t *testing.T) {
	root := &cobra.Command{Use: "synth"}
	root.AddGroup(&cobra.Group{ID: "records", Title: "Records:"})
	noop := func(*cobra.Command, []string) {}
	root.AddCommand(
		&cobra.Command{Use: "placed", GroupID: "records", Run: noop},
		&cobra.Command{Use: "forgotten", Run: noop},
		&cobra.Command{Use: "secret", Hidden: true, Run: noop},
		&cobra.Command{Use: "stray", GroupID: "nowhere", Run: noop},
	)
	got := ungroupedVerbs(root)
	if strings.Join(got, " ") != "forgotten stray" {
		t.Fatalf("ungroupedVerbs = %v, want [forgotten stray]", got)
	}
}

// TestPlacementChangesNoInvocation is the rest of criterion 3: a verb runs the
// same whichever block lists it. Placement never hides a verb and never touches
// how it runs, so every placed verb stays an available, runnable command, and an
// agents-block verb executes from the CLI exactly as before.
func TestPlacementChangesNoInvocation(t *testing.T) {
	root := NewRootCommand()
	for path := range helpPlacements {
		cmd := findByPath(root, strings.Fields(path))
		if cmd == nil {
			t.Errorf("helpPlacements names %q, which the tree does not register", path)
			continue
		}
		if !cmd.IsAvailableCommand() {
			t.Errorf("%q is placed in the help but is not an available command", path)
		}
	}
	out := runCLI(t, "version", "--json")
	if !bytes.Contains(out, []byte(`"version"`)) {
		t.Fatalf("`abcd version --json` (an agents-block verb) did not run as before:\n%s", out)
	}
}

// TestRulesAndSpecKeepTheirPlaces is criterion 6: `rules` and `spec` stay
// top-level, visible and listed in the person's default help.
func TestRulesAndSpecKeepTheirPlaces(t *testing.T) {
	help, root := executedHelp(t, "--help")
	_, entries := helpSections(help)
	listed := map[string]bool{}
	for _, group := range peopleGroupTitles {
		for _, n := range entries[group] {
			listed[n] = true
		}
	}
	for _, verb := range []string{"rules", "spec"} {
		cmd := findByPath(root, []string{verb})
		if cmd == nil || cmd.Hidden {
			t.Errorf("`abcd %s` must stay a visible top-level verb", verb)
		}
		if !listed[verb] {
			t.Errorf("`abcd %s` is missing from the person's default help\n%s", verb, help)
		}
	}
}

// TestAgentFlagOutsideHelpRefuses: --agent modifies the help and nothing else, so
// passing it to the bare board is a usage error, not a silently ignored flag.
func TestAgentFlagOutsideHelpRefuses(t *testing.T) {
	out, err := runCLIErr(t, "--agent")
	if err == nil {
		t.Fatalf("`abcd --agent` must refuse:\n%s", out)
	}
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("`abcd --agent` exit = %d, want 2 (a usage error)", code)
	}
	if !strings.Contains(err.Error(), "--help --agent") {
		t.Fatalf("the refusal must name the spelling that works, got %v", err)
	}
}

// TestCommandPagesDeclareTheirBlock is criterion 5 on the page side: every
// command page backing a visible top-level verb says in its frontmatter which
// help block lists that verb, and says the one the tree says.
func TestCommandPagesDeclareTheirBlock(t *testing.T) {
	root := NewRootCommand()
	for verb, body := range commandFileBodies(t) {
		cmd := findByPath(root, []string{verb})
		if cmd == nil || cmd.Hidden {
			continue
		}
		head, _ := frontmatter.Split(body)
		field, ok := frontmatter.Fields(strings.Split(head, "\n"))["block"]
		want := helpBlock(cmd)
		switch {
		case !ok:
			t.Errorf("commands/%s.md declares no `block:`; the tree lists it in %q", verb, want)
		case field.Value != want:
			t.Errorf("commands/%s.md says `block: %s`; the tree lists it in %q", verb, field.Value, want)
		}
	}
}

// TestSurfaceSnapshotRecordsHelpPlacement is criterion 4's input: the snapshot
// records the group of a visible top-level verb and the block of every listed
// entry, and nothing for a command that is not listed on its own line.
func TestSurfaceSnapshotRecordsHelpPlacement(t *testing.T) {
	snap, err := SurfaceSnapshot(testRepoRoot())
	if err != nil {
		t.Fatalf("SurfaceSnapshot: %v", err)
	}
	for _, tc := range []struct{ path, group, block string }{
		{"abcd capture", groupRecords, blockPeople},
		{"abcd rules", groupSetUp, blockPeople},
		{"abcd implement", groupAgents, blockAgents},
		{"abcd guard hook", "", blockAgents},
		{"abcd intent audit ingest", "", blockAgents},
		{"abcd capture list", "", ""},
		{"abcd hook", "", ""},
		{"abcd", "", ""},
	} {
		cmd, ok := findCommand(snap, tc.path)
		if !ok {
			t.Fatalf("%q missing from the snapshot", tc.path)
		}
		if cmd.Group != tc.group || cmd.Block != tc.block {
			t.Errorf("%q group/block = %q/%q, want %q/%q", tc.path, cmd.Group, cmd.Block, tc.group, tc.block)
		}
	}
}
