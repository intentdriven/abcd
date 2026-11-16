package lint

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
)

// commandpagefacts_test.go — the prose lint of the command pages
// (spc-2610301431075964 step 4): each finding kind with its line, the
// historical marker, generated blocks, persistent flags, hidden commands, and
// the near misses the lint must leave alone.

// pageFactsFixture is a small tree in the facts' shape: a root with a
// persistent --json and a record-id argument, an input of version 2 and one of
// version 1 under the same flag name, a group taking a state word, a group
// taking text, a hidden group with a persistent flag, and a receipt flag that
// is no input.
func pageFactsFixture() []surface.Facts {
	return []surface.Facts{
		{Path: "abcd", Usage: "abcd [<record-id>] [flags]", Flags: []surface.FlagFact{
			{Name: "json", Type: surface.FlagBool, Persistent: true},
			{Name: "version", Type: surface.FlagBool},
		}},
		{Path: "abcd ideate", Usage: "abcd ideate [flags]"},
		{Path: "abcd ideate record", Usage: "abcd ideate record <idea-slug> --verdict-json <file|-> [--dry-run] [flags]",
			Flags: []surface.FlagFact{
				{Name: "dry-run", Type: surface.FlagBool},
				{Name: "public", Type: surface.FlagBool},
				{Name: "verdict-json", Type: surface.FlagString},
			},
			Inputs: []surface.InputFact{{Flag: "verdict-json", Kind: surface.InputJSON, Name: "ideate verdict", SchemaVersion: 2, VersionIn: surface.VersionField}}},
		{Path: "abcd docs", Usage: "abcd docs [flags]"},
		{Path: "abcd docs cite", Usage: "abcd docs cite [flags]"},
		{Path: "abcd docs cite confirm", Usage: "abcd docs cite confirm [url...] [flags]",
			Flags: []surface.FlagFact{{Name: "receipt", Type: surface.FlagString}}},
		{Path: "abcd docs fidelity", Usage: "abcd docs fidelity [flags]"},
		{Path: "abcd docs fidelity record", Usage: "abcd docs fidelity record --verdict-json <file|-> [flags]",
			Flags:  []surface.FlagFact{{Name: "verdict-json", Type: surface.FlagString}},
			Inputs: []surface.InputFact{{Flag: "verdict-json", Kind: surface.InputJSON, Name: "fidelity verdict", SchemaVersion: 1, VersionIn: surface.VersionField}}},
		{Path: "abcd mode", Usage: "abcd mode [<state>] [flags]"},
		{Path: "abcd mode queue", Usage: "abcd mode queue [flags]"},
		{Path: "abcd capture", Usage: "abcd capture [text] [flags]"},
		{Path: "abcd capture resolve", Usage: "abcd capture resolve <iss-N> <note> --impact <kind> [flags]",
			Flags: []surface.FlagFact{{Name: "impact", Type: surface.FlagString}}},
		{Path: "abcd hook", Hidden: true, Usage: "abcd hook [flags]",
			Flags: []surface.FlagFact{{Name: "event", Type: surface.FlagString, Persistent: true}}},
		{Path: "abcd hook dogfood-binary", Usage: "abcd hook dogfood-binary [flags]",
			Flags: []surface.FlagFact{{Name: "quiet", Type: surface.FlagBool}}},
		{Path: "abcd implement", Usage: "abcd implement [flags]"},
		{Path: "abcd implement load", Usage: "abcd implement load [flags] -- <command>...",
			Flags: []surface.FlagFact{{Name: "wait", Type: surface.FlagBool}, {Name: "timeout", Type: surface.FlagDuration}}},
		{Path: "abcd banlist", Usage: "abcd banlist [flags]"},
		{Path: "abcd banlist add", Usage: "abcd banlist add --private|--public <key> <pattern|-> [flags]",
			Flags: []surface.FlagFact{{Name: "private", Type: surface.FlagBool}, {Name: "public", Type: surface.FlagBool}}},
		{Path: "abcd status", Usage: "abcd status [flags]"},
	}
}

// findingLines is each finding as "line: message" for a failure to print.
func findingLines(fs []CommandPageFinding) string {
	var b strings.Builder
	for _, f := range fs {
		b.WriteString(f.String() + "\n")
	}
	return b.String()
}

func TestCommandPageFacts(t *testing.T) {
	cases := []struct {
		name string
		page string
		text string
		// want is each expected finding as its line and a fragment of its
		// message, in line order; empty means the page is clean.
		want []struct {
			line int
			frag string
		}
	}{
		{
			name: "a sub-command the tree lacks, in a span",
			text: "# Title\n\nRun `abcd ideate recrod <slug>` to record it.\n",
			want: []struct {
				line int
				frag string
			}{{3, "`abcd ideate` has no sub-command `recrod`"}},
		},
		{
			name: "a verb the tree lacks, in a fenced line of the plugin-root rung",
			text: "# Title\n\n```bash\n\"${CLAUDE_PLUGIN_ROOT}/abcd\" status\n\"${CLAUDE_PLUGIN_ROOT}/abcd\" source ledger --flip 3\n```\n",
			want: []struct {
				line int
				frag string
			}{{5, "`abcd` has no sub-command `source`"}},
		},
		{
			name: "a flag the command lacks",
			text: "# Title\n\n```\nabcd ideate record slug --verdict-json v.json --dry-run --force\n```\n",
			want: []struct {
				line int
				frag string
			}{{4, "`--force` is not a flag of `abcd ideate record`"}},
		},
		{
			name: "a flag of a sibling command is not the command's",
			text: "`abcd docs cite confirm --verdict-json x`\n",
			want: []struct {
				line int
				frag string
			}{{1, "`--verdict-json` is not a flag of `abcd docs cite confirm`"}},
		},
		{
			name: "a stale schema_version beside the input's call",
			text: "# Title\n\n## Record\n\n```bash\nabcd ideate record slug --verdict-json v.json\n```\n\n```json\n{\n  \"schema_version\": 1,\n  \"idea\": \"x\"\n}\n```\n",
			want: []struct {
				line int
				frag string
			}{{11, "`\"schema_version\": 1` stands next to `--verdict-json` (line 6), which `abcd ideate record` reads at schema_version 2"}},
		},
		{
			name: "a stale schema_version in prose beside a bare mention, read against the page's own verb",
			page: "commands/ideate.md",
			text: "Pass the verdict with `--verdict-json`; a payload of `\"schema_version\": 1` is refused.\n",
			want: []struct {
				line int
				frag string
			}{{1, "reads at schema_version 2"}},
		},
		{
			name: "the same bare mention on another verb's page reads that verb's input",
			page: "commands/docs.md",
			text: "Pass the verdict with `--verdict-json`, as `{\"schema_version\": 1}`.\n",
		},
		{
			name: "every finding kind is exempt on a line carrying the marker",
			text: "# Title\n\n## Record\n\n`abcd ideate recrod` was the old name. <!-- facts-lint: historical -->\n" +
				"`abcd ideate record --force` was refused. <!-- facts-lint: historical -->\n\n" +
				"`abcd ideate record s --verdict-json v.json` reads version 2.\n\n" +
				"A `\"schema_version\": 1` payload is refused. <!-- facts-lint: historical -->\n",
		},
		{
			name: "the marker on a wrapped span's second line exempts the span",
			text: "The receipt opens `abcd updated from\n<old> to <new>`. <!-- facts-lint: historical -->\n",
		},
		{
			name: "generated blocks are skipped",
			text: "# Title\n\n<!-- generated: ask-the-binary -->\nRun `abcd ideate recrod`.\n<!-- /generated -->\n\n" +
				"<!-- generated: command-facts -->\n- `abcd ideate record --force`\n<!-- /generated -->\n",
		},
		{
			name: "a persistent flag of the root, and of a hidden ancestor, is registered",
			text: "`abcd ideate record s --verdict-json v --json` and `abcd hook dogfood-binary --event x --quiet --json` and `abcd status --help`\n",
		},
		{
			name: "a hidden command is a command, and its unregistered flag is reported",
			text: "`abcd hook dogfood-binary --loud`\n",
			want: []struct {
				line int
				frag string
			}{{1, "`--loud` is not a flag of `abcd hook dogfood-binary`"}},
		},
		{
			name: "a persistent flag of another branch is not registered",
			text: "`abcd status --event x`\n",
			want: []struct {
				line int
				frag string
			}{{1, "`--event` is not a flag of `abcd status`"}},
		},
		{
			name: "prose that only mentions abcd",
			text: "# abcd\n\n`abcd` is the binary, and abcd's pages name it. The checkout is `$ABCD`, read `$ABCD/AGENTS.md`.\n" +
				"The program `abcd-source-ledger-flip` and `abcdx status --bogus` are not abcd.\n",
		},
		{
			name: "a flag of another tool in a pipeline or after a comment",
			text: "```bash\nabcd status --json | jq --raw-output .x\nabcd status --json 2>&1 >/dev/null --nope\nabcd status   # --verbose for more\nabcd status && git push --force-with-lease\n```\n",
		},
		{
			name: "an invocation after a pipe is read",
			text: "```bash\nprintf %s '<p>' | abcd banlist add --private key - --bogus\n```\n",
			want: []struct {
				line int
				frag string
			}{{2, "`--bogus` is not a flag of `abcd banlist add`"}},
		},
		{
			name: "words after a bare -- are passed through",
			text: "`abcd implement load --wait --timeout 25m -- go test --count=1 -run X ./...`\n",
		},
		{
			name: "a quoted argument holding a flag is an argument",
			text: "`abcd capture \"the --frobnicate flag is gone\"` and `abcd capture resolve iss-1 'note --x' --impact fix`\n",
		},
		{
			name: "arguments are not sub-commands",
			text: "`abcd itd-2610301346085246` and `abcd <record-id>` and `abcd mode managed` and `abcd mode queue --json` and `abcd ideate record my-slug --verdict-json -`\n",
		},
		{
			name: "a word after a text-taking group is a misspelt sub-command",
			text: "`abcd capture resolv iss-1 note`\n",
			want: []struct {
				line int
				frag string
			}{{1, "`abcd capture` has no sub-command `resolv`"}},
		},
		{
			name: "usage-line shapes: brackets, alternatives, placeholders and punctuation",
			text: "`abcd banlist add --private|--public <key> <pattern|->` and `abcd ideate record <slug> [--public | --dry-run] --verdict-json <file|->`, then `abcd --<flag>`.\n",
		},
		{
			name: "every binary rung is read",
			text: "```\ngo run ./cmd/abcd status --a\nscripts/abcd.sh status --b\n./scripts/abcd.sh status --c\n```\n",
			want: []struct {
				line int
				frag string
			}{{2, "`--a`"}, {3, "`--b`"}, {4, "`--c`"}},
		},
		{
			name: "a span may wrap onto the next line, and a fenced line may continue",
			text: "Then `abcd\nideate recrod` runs.\n\n```bash\nabcd ideate record slug \\\n  --verdict-json v.json \\\n  --gone\n```\n",
			want: []struct {
				line int
				frag string
			}{{1, "`abcd ideate` has no sub-command `recrod`"}, {5, "`--gone`"}},
		},
		{
			name: "a closing fence ends a fenced line's continuation, and the spans after it are read",
			text: "# Title\n\n```bash\nabcd status \\\n```\n\nThen `abcd ideate recrod`.\n",
			want: []struct {
				line int
				frag string
			}{{7, "`abcd ideate` has no sub-command `recrod`"}},
		},
		{
			name: "a fence indented under a list item is read as a fence",
			text: "1. Record it:\n\n     ```bash\n     abcd ideate recrod\n     ```\n",
			want: []struct {
				line int
				frag string
			}{{4, "`abcd ideate` has no sub-command `recrod`"}},
		},
		{
			name: "a schema_version beside a call that passes no input is not checked",
			text: "## Receipts\n\n```bash\nabcd docs cite confirm --receipt receipts.json\n```\n\n```json\n{\"schema_version\": 7}\n```\n",
		},
		{
			name: "a schema_version is not read against an input in another section",
			text: "## Record\n\n`abcd ideate record s --verdict-json v`\n\n## Answers\n\nThe file is `{\"schema_version\": 1, \"answers\": []}`.\n",
		},
		{
			name: "a schema_version before its call in the same section is read against it",
			text: "## Record\n\n```json\n{\"schema_version\": 3}\n```\n\nPass it as `abcd ideate record s --verdict-json v`.\n",
			want: []struct {
				line int
				frag string
			}{{4, "schema_version 2"}},
		},
		{
			name: "a heading inside a fence does not open a section",
			text: "## Record\n\n`abcd ideate record s --verdict-json v`\n\n```\n# not a heading\n```\n\n`{\"schema_version\": 1}`\n",
			want: []struct {
				line int
				frag string
			}{{9, "schema_version 2"}},
		},
	}
	facts := pageFactsFixture()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page := tc.page
			if page == "" {
				page = "commands/example.md"
			}
			got := CommandPageFacts(page, tc.text, facts)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d findings, want %d:\n%s", len(got), len(tc.want), findingLines(got))
			}
			for i, w := range tc.want {
				if got[i].Page != page || got[i].Line != w.line || !strings.Contains(got[i].Message, w.frag) {
					t.Errorf("finding %d is %q; want %s:%d containing %q", i, got[i].String(), page, w.line, w.frag)
				}
			}
		})
	}
}

// The function holds no state between calls and reads only what it is given:
// an empty tree reports nothing, and the same input gives the same findings.
func TestCommandPageFactsIsPure(t *testing.T) {
	text := "`abcd ideate recrod` and `abcd status --bogus`\n"
	if got := CommandPageFacts("p.md", text, nil); len(got) != 0 {
		t.Errorf("an empty tree reported %d findings:\n%s", len(got), findingLines(got))
	}
	facts := pageFactsFixture()
	a := findingLines(CommandPageFacts("p.md", text, facts))
	b := findingLines(CommandPageFacts("p.md", text, facts))
	if a != b || a == "" {
		t.Errorf("two calls disagree or report nothing:\n%s---\n%s", a, b)
	}
}

func TestTakesWordArgument(t *testing.T) {
	for _, tc := range []struct {
		path, usage string
		want        bool
	}{
		{"abcd", "abcd [<record-id>] [flags]", false},
		{"abcd build", "abcd build <itd-N|iss-N> [--session <id>] [flags]", false},
		{"abcd capture", "abcd capture [text] [flags]", false},
		{"abcd intent audit", "abcd intent audit [<itd-N>] | audit --owed [flags]", false},
		{"abcd reflect", "abcd reflect <release-tag> [flags]", false},
		{"abcd update", "abcd update [tag] [flags]", false},
		{"abcd ahoy", "abcd ahoy [flags]", false},
		{"abcd mode", "abcd mode [<state>] [flags]", true},
		{"abcd ahoy shared-folder", "abcd ahoy shared-folder [<folder>] [--person <account>] [flags]", true},
		{"abcd guard allow", "abcd guard allow --once --command <line> [flags]", false},
	} {
		if got := takesWordArgument(surface.Facts{Path: tc.path, Usage: tc.usage}); got != tc.want {
			t.Errorf("takesWordArgument(%q) = %v, want %v", tc.usage, got, tc.want)
		}
	}
}
