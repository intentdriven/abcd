package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// helpgroups.go — itd-146: `abcd --help` lists the person's verbs under labelled
// groups, and `abcd --help --agent` adds a second block of the verbs agents and
// hosts call, each line naming the page an agent reads next.
//
// Placement is presentation only. No verb is hidden, renamed, moved or nested by
// it, and every verb runs the same whichever block lists it; the group titles
// carry no adr-40 bucket meaning. What makes the placement last is the gate
// around it: the surface snapshot records each placement (surface.Command.Group
// and .Block), TestEveryVisibleVerbHasAGroup fails on a visible top-level verb
// with no group, and TestCommandPagesDeclareTheirBlock holds each command page's
// `block:` frontmatter to the tree.

// The help group ids. The first five are the person's groups, in the order the
// help lists them; groupAgents is the agents-and-hosts block, which is one
// group because it is one list.
const (
	groupSetUp       = "set-up"
	groupRecords     = "records"
	groupChecks      = "checks"
	groupPortability = "portability"
	groupRelease     = "release"
	groupAgents      = "agents"
)

// The two blocks a listed entry belongs to, as the snapshot and the command
// pages' `block:` frontmatter spell them.
const (
	blockPeople = "people"
	blockAgents = "agents"
)

// helpPeopleGroups is the person's groups in the order the help renders them.
var helpPeopleGroups = []*cobra.Group{
	{ID: groupSetUp, Title: "Set-up:"},
	{ID: groupRecords, Title: "Records:"},
	{ID: groupChecks, Title: "Checks:"},
	{ID: groupPortability, Title: "Portability:"},
	{ID: groupRelease, Title: "Release:"},
}

// The lines the root help renders around the groups.
const (
	// helpExpandLine is the one line above the person's groups that says the
	// list expands (criterion 1).
	helpExpandLine = `Run "abcd --help --agent" to expand this list with the verbs agents and hosts call.`
	// helpPeopleTitle heads the person's block when both blocks render.
	helpPeopleTitle = "For people:"
	// helpAgentsTitle heads the agents-and-hosts block. It is also the cobra
	// title of groupAgents, so the one list has one name.
	helpAgentsTitle = "For agents and hosts (each line names the page to read next):"
)

// Annotation keys the placement writes onto a command.
const (
	// annotationHelpBlock marks a SUB-verb listed in its own right; a top-level
	// verb's block derives from its group.
	annotationHelpBlock = "abcd.help.block"
	// annotationHelpPage is the page an agents-block line names.
	annotationHelpPage = "abcd.help.page"
)

// helpPlacement is where one entry is listed. A top-level verb carries a group;
// an entry of the agents block carries the page an agent reads next. A sub-verb
// is listed in its own right only in the agents block (the person's groups list
// top-level verbs), so a sub-verb placement is always an agents-block one.
type helpPlacement struct {
	group string
	page  string
}

// helpPlacements is every placement, keyed by the path below the root. Decision
// 2 of itd-146 places the people's thirteen (build and drain are not built yet)
// and the agent block's nine; the rest are the technical ruling recorded in
// .abcd/work/DECISIONS.md on 2026-09-25, which gives each its reason. cobra's own
// `help` and `completion` are filed under set-up by applyHelpPlacement, because
// they exist only once the tree executes.
var helpPlacements = map[string]helpPlacement{
	// The person's groups.
	"ahoy":      {group: groupSetUp},
	"rules":     {group: groupSetUp},
	"update":    {group: groupSetUp},
	"capture":   {group: groupRecords},
	"decide":    {group: groupRecords},
	"intent":    {group: groupRecords},
	"memory":    {group: groupRecords},
	"spec":      {group: groupRecords},
	"lint":      {group: groupChecks},
	"disembark": {group: groupPortability},
	"embark":    {group: groupPortability},
	"launch":    {group: groupRelease},

	// The agents-and-hosts block.
	"banlist":             {group: groupAgents, page: "commands/banlist.md"},
	"changelog":           {group: groupAgents, page: "commands/launch.md"},
	"docs":                {group: groupAgents, page: "commands/docs.md"},
	"guard":               {group: groupAgents, page: "commands/guard.md"},
	"guard hook":          {page: "commands/guard.md"},
	"history":             {group: groupAgents, page: "commands/history.md"},
	"ideate":              {group: groupAgents, page: "commands/ideate.md"},
	"ideate record":       {page: "commands/ideate.md"},
	"identity":            {group: groupAgents, page: "commands/identity.md"},
	"implement":           {group: groupAgents, page: "commands/implement.md"},
	"inbox":               {group: groupAgents, page: "commands/inbox.md"},
	"intent audit ingest": {page: "commands/intent.md"},
	"mode":                {group: groupAgents, page: "commands/mode.md"},
	"peers":               {group: groupAgents, page: "commands/peers.md"},
	"reading":             {group: groupAgents, page: "commands/reading.md"},
	"report":              {group: groupAgents, page: "commands/report.md"},
	"site":                {group: groupAgents, page: "commands/site.md"},
	"statusline":          {group: groupAgents, page: "commands/ahoy.md"},
	"version":             {group: groupAgents, page: "commands/version.md"},
}

// applyHelpPlacement declares the groups on root, files every placed entry, and
// installs the grouped help render. agent is the root's --agent flag.
//
// A placement naming a path the tree does not register is skipped here and
// named by TestPlacementChangesNoInvocation, so construction never panics over a
// table entry.
func applyHelpPlacement(root *cobra.Command, agent *bool) {
	root.AddGroup(helpPeopleGroups...)
	root.AddGroup(&cobra.Group{ID: groupAgents, Title: helpAgentsTitle})
	root.SetHelpCommandGroupID(groupSetUp)
	root.SetCompletionCommandGroupID(groupSetUp)

	for path, p := range helpPlacements {
		cmd := findByPath(root, strings.Fields(path))
		if cmd == nil {
			continue
		}
		if p.group != "" {
			cmd.GroupID = p.group
		}
		if cmd.Parent() != root && p.page != "" {
			annotate(cmd, annotationHelpBlock, blockAgents)
		}
		if p.page != "" {
			annotate(cmd, annotationHelpPage, p.page)
		}
	}

	// Every other command keeps cobra's help: the grouping is the root's list.
	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd != root {
			defaultHelp(cmd, args)
			return
		}
		renderRootHelp(cmd.OutOrStdout(), root, *agent)
	})
}

func annotate(cmd *cobra.Command, key, value string) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[key] = value
}

// isTopLevel reports whether cmd sits directly under the root.
func isTopLevel(cmd *cobra.Command) bool {
	return cmd.HasParent() && !cmd.Parent().HasParent()
}

// helpGroup is the group the snapshot records: a visible top-level verb's
// GroupID, and nothing for any other command.
func helpGroup(cmd *cobra.Command) string {
	if !isTopLevel(cmd) || cmd.Hidden {
		return ""
	}
	return cmd.GroupID
}

// helpBlock is the block that lists cmd, or "" when no line lists it on its own.
func helpBlock(cmd *cobra.Command) string {
	if isTopLevel(cmd) {
		switch helpGroup(cmd) {
		case "":
			return ""
		case groupAgents:
			return blockAgents
		default:
			return blockPeople
		}
	}
	return cmd.Annotations[annotationHelpBlock]
}

// listed reports whether the root's help lists cmd: an available command, or
// cobra's help command, which cobra marks unavailable but lists all the same.
func listed(cmd *cobra.Command) bool {
	return cmd.IsAvailableCommand() || (isTopLevel(cmd) && cmd.Name() == "help")
}

// ungroupedVerbs names every listed top-level verb whose group is missing or is
// not one root declares — the verbs criterion 3 fails a test on. A hidden verb
// is exempt: it never renders, so a group would be a claim about nothing.
func ungroupedVerbs(root *cobra.Command) []string {
	var out []string
	for _, sub := range root.Commands() {
		if !listed(sub) {
			continue
		}
		if sub.GroupID == "" || !root.ContainsGroup(sub.GroupID) {
			out = append(out, sub.Name())
		}
	}
	sort.Strings(out)
	return out
}

// helpEntry is one line of the root's list.
type helpEntry struct {
	name, short, page string
}

// agentEntries collects the agents-and-hosts block from the whole tree, sorted
// by path, so a sub-verb sits beside its parent.
func agentEntries(root *cobra.Command) []helpEntry {
	var out []helpEntry
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if listed(sub) && helpBlock(sub) == blockAgents {
				out = append(out, helpEntry{
					name:  strings.TrimPrefix(sub.CommandPath(), root.Name()+" "),
					short: sub.Short,
					page:  sub.Annotations[annotationHelpPage],
				})
			}
			walk(sub)
		}
	}
	walk(root)
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// renderRootHelp writes the root's help: the sentence, the description and the
// usage, then the person's groups under the line that says --agent expands
// them, or — with --agent — the person's block and then the agents-and-hosts
// block, then the flags. The layout follows cobra's own template, so everything but the list
// reads as it always has.
func renderRootHelp(w io.Writer, root *cobra.Command, agent bool) {
	groups := map[string][]helpEntry{}
	for _, sub := range root.Commands() {
		if listed(sub) && helpBlock(sub) == blockPeople {
			groups[sub.GroupID] = append(groups[sub.GroupID], helpEntry{name: sub.Name(), short: sub.Short})
		}
	}
	var agents []helpEntry
	if agent {
		agents = agentEntries(root)
	}
	width := 0
	for _, e := range agents {
		width = max(width, len(e.name))
	}
	for _, entries := range groups {
		for _, e := range entries {
			width = max(width, len(e.name))
		}
	}

	// The root's help opens with its sentence, as every verb's does
	// (itd-2609212113220149), and the long help follows it.
	if short := strings.TrimSpace(root.Short); short != "" {
		fmt.Fprintf(w, "%s\n\n", short)
	}
	if long := strings.TrimSpace(root.Long); long != "" {
		fmt.Fprintf(w, "%s\n\n", long)
	}
	fmt.Fprintf(w, "Usage:\n  %s\n  %s [command]\n\n", root.UseLine(), root.CommandPath())
	if agent {
		fmt.Fprintf(w, "%s\n\n", helpPeopleTitle)
	} else {
		fmt.Fprintf(w, "%s\n\n", helpExpandLine)
	}
	for _, g := range helpPeopleGroups {
		entries := groups[g.ID]
		if len(entries) == 0 {
			continue
		}
		fmt.Fprintf(w, "%s\n", g.Title)
		for _, e := range entries {
			fmt.Fprintf(w, "  %-*s  %s\n", width, e.name, e.short)
		}
		fmt.Fprintln(w)
	}
	if agent {
		fmt.Fprintf(w, "%s\n", helpAgentsTitle)
		for _, e := range agents {
			fmt.Fprintf(w, "  %-*s  %s (read %s)\n", width, e.name, e.short, e.page)
		}
		fmt.Fprintln(w)
	}
	if flags := strings.TrimRight(root.LocalFlags().FlagUsages(), " \n"); flags != "" {
		fmt.Fprintf(w, "Flags:\n%s\n\n", flags)
	}
	fmt.Fprintf(w, "Use \"%s [command] --help\" for more information about a command.\n", root.CommandPath())
}
