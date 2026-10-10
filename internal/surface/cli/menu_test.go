package cli

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/launch"
)

// menu_test.go holds itd-2610090831227812 (spc-2610100613109045, step 1): the
// person's `/abcd:` menu lists the pages whose `block:` is `people`, and every
// page whose `block:` is `agents` carries the host's `user-invocable: false`,
// which hides it from that menu and keeps its name. A page's `block:` stays the
// one statement of whose command it is; the first test holds the host's key to
// it page by page, and the second holds the plugin's person list equal to the
// command line's (the product thinker's ruling of 2026-10-09: no cap for either
// surface, the two kept in step).

// userInvocableKey is the host's frontmatter key that, set to false, leaves a
// command off the person's menu and refuses it when a person types it.
const userInvocableKey = "user-invocable"

// pageMenuDefects names every page, keyed by its name under commands/, whose
// frontmatter disagrees with the menu: a page declares `block:` as `people` or
// `agents`; a people page carries no user-invocable key; an agents page carries
// `user-invocable: false` and nothing else in that key. Each defect names the
// page. The result is sorted, so a failure reads the same on every run.
func pageMenuDefects(pages map[string]string) []string {
	names := make([]string, 0, len(pages))
	for name := range pages {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []string
	for _, name := range names {
		page := pluginCommandsDir + "/" + name + ".md"
		fields, _, err := launch.PageFieldsForTest(pages[name])
		if err != nil {
			out = append(out, fmt.Sprintf("%s: its frontmatter cannot be read: %v", page, err))
			continue
		}
		block, hasBlock := fields["block"]
		key, hasKey := fields[userInvocableKey]
		switch {
		case !hasBlock:
			out = append(out, fmt.Sprintf("%s declares no `block:`; every page says `block: people` or `block: agents`", page))
		case block == blockPeople && hasKey:
			out = append(out, fmt.Sprintf("%s says `block: people` but carries `%s: %s`; a person's page carries no such key, or the host hides it from the person's menu", page, userInvocableKey, key))
		case block == blockPeople:
		case block == blockAgents && !hasKey:
			out = append(out, fmt.Sprintf("%s says `block: agents` but carries no `%s: false`, so the host lists it on the person's menu", page, userInvocableKey))
		case block == blockAgents && key != "false":
			out = append(out, fmt.Sprintf("%s says `block: agents` but `%s: %s`; an agent's page carries `%s: false` and nothing else", page, userInvocableKey, key, userInvocableKey))
		case block == blockAgents:
		default:
			out = append(out, fmt.Sprintf("%s says `block: %s`; a page is `people` or `agents`", page, block))
		}
	}
	return out
}

// TestCommandPagesMatchTheMenu is acceptance criterion A5: a page whose
// `block:` and the host's user-invocable key disagree fails here, named. Every
// page is in the walk, the board's included.
func TestCommandPagesMatchTheMenu(t *testing.T) {
	pages := commandFileBodies(t)
	board, ok := pages[dispatcherPage]
	if !ok {
		t.Fatalf("the walk holds no %s/%s.md; the board's page must be in it", pluginCommandsDir, dispatcherPage)
	}
	if fields, _, err := launch.PageFieldsForTest(board); err != nil || fields["block"] != blockPeople {
		t.Errorf("%s/%s.md must say `block: people`: the board is a person's command (got %q, %v)", pluginCommandsDir, dispatcherPage, fields["block"], err)
	}
	for _, defect := range pageMenuDefects(pages) {
		t.Error(defect)
	}
}

// TestPageMenuDefectsNamesEachDefect is the menu test's negative control: on a
// synthetic page set, each defect is named with its page, and the pages that
// follow the rule are named nowhere.
func TestPageMenuDefectsNamesEachDefect(t *testing.T) {
	pages := map[string]string{
		"person":       "---\nname: person\nblock: people\n---\n# body\n",
		"agent":        "---\nname: agent\nblock: agents\nuser-invocable: false\n---\n# body\n",
		"blockless":    "---\nname: blockless\n---\n# body\n",
		"no-head":      "# a page with no frontmatter at all\n",
		"odd-block":    "---\nname: odd-block\nblock: hosts\n---\n",
		"keyed-person": "---\nname: keyed-person\nblock: people\nuser-invocable: false\n---\n",
		"bare-agent":   "---\nname: bare-agent\nblock: agents\n---\n",
		"true-agent":   "---\nname: true-agent\nblock: agents\nuser-invocable: true\n---\n",
		"unclosed":     "---\nname: unclosed\nblock: people\n",
	}
	want := map[string]string{
		"blockless":    "declares no `block:`",
		"no-head":      "declares no `block:`",
		"odd-block":    "`block: hosts`",
		"keyed-person": "a person's page carries no such key",
		"bare-agent":   "carries no `user-invocable: false`",
		"true-agent":   "`user-invocable: true`",
		"unclosed":     "never closed",
	}
	got := pageMenuDefects(pages)
	if len(got) != len(want) {
		t.Errorf("pageMenuDefects named %d defects, want %d:\n%s", len(got), len(want), strings.Join(got, "\n"))
	}
	for name, phrase := range want {
		page := pluginCommandsDir + "/" + name + ".md"
		found := false
		for _, d := range got {
			found = found || (strings.HasPrefix(d, page+" ") || strings.HasPrefix(d, page+":")) && strings.Contains(d, phrase)
		}
		if !found {
			t.Errorf("no defect names %s with %q:\n%s", page, phrase, strings.Join(got, "\n"))
		}
	}
	for _, good := range []string{"person", "agent"} {
		for _, d := range got {
			if strings.HasPrefix(d, pluginCommandsDir+"/"+good+".md") {
				t.Errorf("%s follows the rule but is named: %s", good, d)
			}
		}
	}
}

// TestPluginPersonListEqualsTheCLIs holds the plugin's person list equal to the
// command line's: the pages whose `block:` is `people` and that back a CLI verb
// against the verbs `abcd --help` lists under the person's groups, less cobra's
// `help` and `completion`. The pages in pagesWithNoVerb back no verb, so they
// are left out while they exist; the board's page is among them, since the
// board is the bare root and no listed verb. Each failure names the verb on one
// list and not the other.
func TestPluginPersonListEqualsTheCLIs(t *testing.T) {
	help, _ := executedHelp(t, "--help")
	cli := map[string]bool{}
	for _, v := range personVerbs(help) {
		cli[v] = true
	}
	if len(cli) == 0 {
		t.Fatalf("read no verbs out of the person's help; the comparison would pass vacuously:\n%s", help)
	}

	plugin := map[string]bool{}
	for name, body := range commandFileBodies(t) {
		if _, noVerb := pagesWithNoVerb[name]; noVerb {
			continue
		}
		fields, _, err := launch.PageFieldsForTest(body)
		if err != nil {
			t.Errorf("%s/%s.md: its frontmatter cannot be read: %v", pluginCommandsDir, name, err)
			continue
		}
		if fields["block"] == blockPeople {
			plugin[name] = true
		}
	}

	var onlyPlugin, onlyCLI []string
	for name := range plugin {
		if !cli[name] {
			onlyPlugin = append(onlyPlugin, name)
		}
	}
	for name := range cli {
		if !plugin[name] {
			onlyCLI = append(onlyCLI, name)
		}
	}
	sort.Strings(onlyPlugin)
	sort.Strings(onlyCLI)
	for _, name := range onlyPlugin {
		t.Errorf("%s is on the plugin's person list (%s/%s.md says `block: people`) but not on `abcd --help`'s", name, pluginCommandsDir, name)
	}
	for _, name := range onlyCLI {
		t.Errorf("%s is on `abcd --help`'s person list but not on the plugin's (%s/%s.md does not say `block: people`)", name, pluginCommandsDir, name)
	}
}
