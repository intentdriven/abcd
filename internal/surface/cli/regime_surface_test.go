package cli

// regime_surface_test.go is itd-184's ac-3, and it has its own file so the
// ingest verb's surface tests and this guard never contend for one.
//
// The criterion: no registered flag and no registered configuration key sets or
// overrides a run's supply regime. The regime is the definition's property —
// stated in the definition file's `regime:` key and resolved from the position by
// construction (internal/core/reading) — and the whole point of putting it there
// is that an operator cannot choose the licence a reading reads under.
//
// The enumeration is PROGRAMMATIC (itd-195), on both halves.
//
// The command tree is walked through commandSurface, the repository's one
// canonical cobra walk — the same one the release compatibility gate reads —
// rather than through a second walk written here.
//
// The configuration side walks every key of every JSON file tracked under
// .abcd/, enumerated from the git INDEX rather than from chosen directories, so
// a configuration file added tomorrow is walked tomorrow without anyone
// remembering to name it here. The two largest schema types are additionally
// walked by reflection over their json tags, which reaches a key a schema
// declares that no committed file happens to carry.
//
// TWO WRITTEN LISTS SURVIVE, and both are stated rather than implied, because a
// header that claims more than the code establishes is the defect this criterion
// was split out to catch.
//
//  1. readingOperands, the reading verb's pinned operand set. It is a written
//     list ON PURPOSE and it fails CLOSED: any addition to that verb turns this
//     red, so it cannot fall behind the surface — it is a tripwire, not an
//     enumeration.
//  2. generatedBaselineSuffix, the one WRITTEN exclusion from the file walk. It
//     fails OPEN: it can only ever REMOVE the machine-written baselines from the
//     walk, and a configuration file added anywhere under .abcd/ is walked unless
//     it is named for a baseline.
//
// The file walk carries one further exclusion, and it is not a written list: the
// readings record family, taken from issueschema.ReadingsRecordDir rather than
// from a literal, so it moves when the directory moves. See readingsRecordPrefix
// for why a record a verb writes is not an operator surface.
//
// Disclosed residue, as itd-184 states it: what the walk cannot see is a channel
// that was never registered — an environment variable read ad hoc, say. Nothing
// here adds a mechanism for one, and the walk sees every channel that is. Two
// narrower edges sit inside that residue. A key declared by a schema type
// OUTSIDE the two walked by reflection, and written into no tracked file, is
// unregistered in both senses until a file carries it — at which point the index
// walk reaches it. And a knob added to a machine-written baseline is skipped;
// nothing reads a baseline as configuration today, and if something ever does,
// the exclusion above is where that changes.

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/spf13/cobra"
)

// regimeToken is the thing no operator surface may carry. It is matched
// case-insensitively against flag names, flag shorthands and configuration keys.
const regimeToken = "regime"

// readingOperands is the reading verb's closed operand set, as the verb's own
// refusal messages state it: a position and a target state and nothing else,
// plus the two write-side operands and the root's inherited render switch.
//
// It is pinned deliberately. The name check below catches a flag CALLED regime;
// this catches a flag that would set one under any other name, by making any
// addition to the one verb that runs at a position fail until somebody says what
// the new operand does. A new operand on this verb is a decision, not an
// accident.
var readingOperands = map[string][]string{
	"abcd reading": {},
	// The two operands the design admits, plus the two write-side ones. scope
	// was added by itd-199 under adr-58 and is withdrawn by
	// adr-2609021016286571, which supersedes it: the design fixes the
	// invocation at a position and a target state (framework v4 section 8.2 and
	// ruling M8; companion v4 section 4.1), and the committed preset for the
	// position supplies what the reading is handed. This pin fails CLOSED on
	// any addition precisely so a new operand has to say what it does before it
	// ships, and it did its job for scope: the operand was declared here, which
	// is what made its departure from the design legible enough to withdraw.
	"abcd reading assemble": {"dry-run", "out", "position", "target"},
}

// readingOperandMismatches compares the walked tree against readingOperands and
// returns one message per pinned command whose operand set has moved.
//
// It is a function rather than an inline loop so the guard and the mutation
// that proves the guard can fail read the SAME comparison: a mutation checked
// by a second copy of the rule proves that copy, not the rule.
func readingOperandMismatches(commands []surface.Command) []string {
	var out []string
	for _, cmd := range commands {
		want, pinned := readingOperands[cmd.Path]
		if !pinned {
			continue
		}
		got := make([]string, 0, len(cmd.Flags))
		for _, f := range cmd.Flags {
			got = append(got, f.Name)
		}
		sort.Strings(got)
		if strings.Join(got, ",") == strings.Join(want, ",") {
			continue
		}
		out = append(out, fmt.Sprintf("%q declares operands %v, want exactly %v; a new operand on "+
			"the verb that runs at a position has to say what it does before it can ship, because "+
			"the regime is not one of them", cmd.Path, got, want))
	}
	return out
}

// allCommands flattens a cobra tree, so a mutation can be applied to one node.
func allCommands(cmd *cobra.Command) []*cobra.Command {
	out := []*cobra.Command{cmd}
	for _, child := range cmd.Commands() {
		out = append(out, allCommands(child)...)
	}
	return out
}

// generatedBaselineSuffix names the machine-written caches under .abcd/, which
// are the one thing the configuration walk skips.
//
// A baseline is not an operator surface: it is written by the binary
// (internal/core/lint/baseline.go, internal/core/site/paths.go) and read back by
// it, and nobody sets a knob by editing one. It also cannot be searched the way
// a configuration file can, because the citations baseline uses CITED URLS as
// its map keys -- 51 of them today -- so a paper whose URL says "regimes" would
// turn this guard red while naming no knob at all. Skipping it removes a false
// red, not a real channel.
//
// This suffix is the ONE written rule in the enumeration, and it is stated here
// rather than implied. It fails OPEN: a configuration file added anywhere under
// .abcd/ is walked without an edit here, and the only ways out of the walk are to
// be named for a baseline or to sit in the readings record family below.
const generatedBaselineSuffix = "-baseline.json"

// readingsRecordPrefix is the readings record family, the second thing the
// configuration walk skips — and unlike the baseline suffix it is DERIVED, from
// issueschema.ReadingsRecordDir rather than from a literal, so it follows the
// directory if the directory ever moves.
//
// A run record is not configuration. `abcd reading ingest` writes
// .abcd/development/readings/<run>/run.json and refusal.json itself, from the
// position's agent definition, and each carries the supply regime the definition
// stamped: the regime is the definition's property and the run record is the
// output contract's durable half (framework v4 section 4 and 8.5; brief
// 04-surfaces/23-reading). An operator cannot set a regime by editing one —
// there is nothing to edit until the verb has already written it — so the family
// is not an operator surface and reading it as one reports a key nobody can
// write. It failed exactly that way on Iteration 2's first committed run
// (iss-2609022252265901); before then no run record was tracked, so the
// over-reach was latent.
//
// Like the suffix above it fails OPEN, and it is narrower: it names one record
// family by its own constant, so nothing outside that directory can fall out of
// the walk through it.
const readingsRecordPrefix = issueschema.ReadingsRecordDir + "/"

// configKey is one configuration key as the walk found it, with the path through
// the file or schema that reaches it.
type configKey struct {
	schema string
	path   string
}

// walkConfigFiles returns every key of every committed configuration file, keys
// nested inside objects and arrays included. This is the enumeration of the
// registered configuration surface: a key an operator can actually write.
//
// The file set comes from the git INDEX -- every tracked .json under .abcd/ --
// rather than from a glob of chosen directories. Two directories was a written
// list, and it had already fallen behind by four files: personas.json, the
// release surface snapshot, the release-gate manifest and the ruleset mirror all
// sit outside .abcd/ and .abcd/config/, so a regime key in any of them passed.
// Reading the index also makes "committed" literally true: an untracked local
// .abcd/scratch.json is nobody's registered configuration and is not walked.
func walkConfigFiles(t *testing.T, repoRoot string) []configKey {
	t.Helper()
	var out []configKey
	var files, excluded, records, belowTheOldGlobs int
	var walk func(file string, v any, prefix string)
	walk = func(file string, v any, prefix string) {
		switch node := v.(type) {
		case map[string]any:
			for k, child := range node {
				key := k
				if prefix != "" {
					key = prefix + "." + k
				}
				out = append(out, configKey{schema: file, path: key})
				walk(file, child, key)
			}
		case []any:
			for _, child := range node {
				walk(file, child, prefix)
			}
		}
	}
	if !gitutil.InRepo(repoRoot) {
		t.Fatal("the configuration walk reads the git index, and this is not a repository checkout")
	}
	listed, err := gitutil.Run(repoRoot, "ls-files", "-z", "--", ".abcd")
	if err != nil {
		t.Fatalf("list the tracked .abcd files: %v", err)
	}
	for _, rel := range strings.Split(listed, "\x00") {
		if !strings.HasSuffix(rel, ".json") {
			continue
		}
		if strings.HasSuffix(path.Base(rel), generatedBaselineSuffix) {
			excluded++
			continue
		}
		// A record the verb writes is not an operator surface; see
		// readingsRecordPrefix. Counted apart from `excluded`, whose guard is
		// about the baseline rule swallowing the configuration tree.
		if strings.HasPrefix(rel, readingsRecordPrefix) {
			records++
			continue
		}
		raw, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		var doc any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("decode %s: %v", rel, err)
		}
		files++
		dir := path.Dir(rel)
		if dir != ".abcd" && dir != ".abcd/config" {
			belowTheOldGlobs++
		}
		walk(rel, doc, "")
	}

	// Three guards, each against a way this walk could report the criterion
	// satisfied by seeing nothing.
	if files < 40 {
		t.Fatalf("the configuration walk found %d tracked .abcd json file(s) (%d baseline(s) and %d readings "+
			"record(s) skipped); the repository carries more, so the walk is broken", files, excluded, records)
	}
	if belowTheOldGlobs == 0 {
		t.Fatalf("every one of the %d walked files sits in .abcd/ or .abcd/config/, which is the two-directory "+
			"glob this walk replaced; the index enumeration is not reaching the rest of the tree", files)
	}
	if excluded > 3 {
		t.Fatalf("the %q rule excluded %d files; it exists to skip the machine-written baselines and cannot "+
			"be allowed to swallow the configuration tree", generatedBaselineSuffix, excluded)
	}
	return out
}

// walkConfigKeys returns every json key reachable in a configuration schema,
// following struct fields, slices, maps and pointers. Reflection over the type
// rather than a written list, for ac-3's reason: a key added tomorrow is walked
// tomorrow, without anyone remembering to add it here.
func walkConfigKeys(schema string, root reflect.Type) []configKey {
	seen := map[reflect.Type]bool{}
	var out []configKey
	var walk func(t reflect.Type, prefix string)
	walk = func(t reflect.Type, prefix string) {
		for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice ||
			t.Kind() == reflect.Array || t.Kind() == reflect.Map {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct || seen[t] {
			return
		}
		seen[t] = true
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if name == "" || name == "-" {
				name = f.Name
			}
			key := name
			if prefix != "" {
				key = prefix + "." + name
			}
			out = append(out, configKey{schema: schema, path: key})
			walk(f.Type, key)
		}
	}
	walk(root, "")
	return out
}

// TestNoOperatorSurfaceSetsARegime is itd-184's ac-3.
//
// It is proved capable of failing by MUTATION rather than by passing:
// registering a `--regime` flag on `reading assemble` turns it red, and removing
// it turns it green again. A guard nobody has watched fail is a guard that
// asserts nothing.
func TestNoOperatorSurfaceSetsARegime(t *testing.T) {
	commands := commandSurface(NewRootCommand())

	// The walk must actually have walked. Without this, a tree that failed to
	// build, or a visitor that silently visited nothing, would report the
	// criterion satisfied by seeing no surface at all — which is exactly the
	// vacuity this guard exists to avoid.
	flagCount := 0
	for _, cmd := range commands {
		flagCount += len(cmd.Flags)
	}
	if len(commands) < 10 {
		t.Fatalf("the command walk found %d commands; the tree is larger than that, so the walk is broken",
			len(commands))
	}
	if flagCount < 20 {
		t.Fatalf("the flag walk found %d flags across %d commands; the walk is broken", flagCount, len(commands))
	}

	seenReadingCommands := map[string]bool{}
	for _, cmd := range commands {
		for _, f := range cmd.Flags {
			if strings.Contains(strings.ToLower(f.Name), regimeToken) {
				t.Errorf("%q registers the flag --%s: a reading's supply regime is its definition's "+
					"property, stated in the definition file, and no operand may set or override it",
					cmd.Path, f.Name)
			}
			if strings.Contains(strings.ToLower(f.Shorthand), regimeToken) {
				t.Errorf("%q registers the shorthand -%s on --%s, which names a regime",
					cmd.Path, f.Shorthand, f.Name)
			}
		}

		if _, pinned := readingOperands[cmd.Path]; !pinned {
			continue
		}
		seenReadingCommands[cmd.Path] = true
	}
	for _, msg := range readingOperandMismatches(commands) {
		t.Error(msg)
	}
	for path := range readingOperands {
		if !seenReadingCommands[path] {
			t.Errorf("the command walk never reached %q, so its operand set was never checked", path)
		}
	}

	// The pin fails CLOSED, and that is the whole of what it claims
	// (itd-2609021003095168 ac-5; adr-2609021016286571, whose consequence is
	// that itd-184's pin is updated to the two operands and goes on failing
	// closed on any addition). A pin nobody has watched refuse an addition is a
	// declaration, not a tripwire, so a third operand is added to a tree built
	// for the purpose and the mismatch is required to name it.
	t.Run("a third operand fails the pin closed", func(t *testing.T) {
		root := NewRootCommand()
		var added string
		var found bool
		for _, cmd := range allCommands(root) {
			if cmd.CommandPath() != "abcd reading assemble" {
				continue
			}
			cmd.Flags().StringVar(&added, "framing", "", "a third operand nobody decided on")
			found = true
		}
		if !found {
			t.Fatal("the tree carries no `abcd reading assemble`, so the mutation proved nothing")
		}
		msgs := readingOperandMismatches(commandSurface(root))
		if len(msgs) == 0 {
			t.Fatal("a third operand on `abcd reading assemble` passed the pin; the pin fails " +
				"closed, so an operand nobody decided on cannot ship green")
		}
		if !strings.Contains(strings.Join(msgs, "\n"), "framing") {
			t.Errorf("the pin refused without naming the added operand:\n%s", strings.Join(msgs, "\n"))
		}
	})

	keys := walkConfigFiles(t, repoRootFromTest(t))
	keys = append(keys, walkConfigKeys("record-lint schema", reflect.TypeOf(lint.Config{}))...)
	keys = append(keys, walkConfigKeys("rules schema", reflect.TypeOf(rules.RuleSet{}))...)
	if len(keys) < 500 {
		t.Fatalf("the configuration walk found %d keys; the committed configuration is larger than that, "+
			"so the walk is broken", len(keys))
	}
	for _, k := range keys {
		if strings.Contains(strings.ToLower(k.path), regimeToken) {
			t.Errorf("%s carries the key %q: a configuration file is an operator surface too, and the "+
				"regime is not settable from one", k.schema, k.path)
		}
	}
}

// TestConfigurationWalkSkipsTheReadingsRecordFamily fixes the BOUNDARY of what
// walkConfigFiles reads as configuration, in both directions.
//
// A run record is not a configuration file. `abcd reading ingest` writes
// .abcd/development/readings/<run>/run.json and its refusal.json itself, from
// the position's agent definition, and each one carries the supply regime the
// definition stamped: framework v4 section 4 puts the regime on the definition
// and carries it on the output contract, and 8.5 makes the run record that
// contract's durable half (brief 04-surfaces/23-reading: the supply regime is
// the definition's, "with no operand and no configuration key able to reach
// it"). An operator cannot set a regime by editing one — there is nothing to
// edit before the verb writes it — so the record family is not an operator
// surface, and the guard has no business reading it as one. The over-reach went
// unnoticed only because no real run had been committed yet
// (iss-2609022252265901).
//
// The second half is the one that matters: narrowing the walk must not narrow it
// over configuration. A regime key in .abcd/config/ is still reported.
func TestConfigurationWalkSkipsTheReadingsRecordFamily(t *testing.T) {
	repo := t.TempDir()
	gitCmd(t, repo, "init", "-q", "-b", "main")

	writeJSON := func(rel, body string) {
		t.Helper()
		full := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// The walk carries three anti-vacuity guards of its own, and they are the
	// point of it: a fixture below their thresholds would prove the skip by
	// making the walk see nothing at all.
	for i := 0; i < 45; i++ {
		writeJSON(fmt.Sprintf(".abcd/config/pad-%02d.json", i), `{"outer": {"inner": 1}}`)
	}
	writeJSON(".abcd/work/rulesets/main.json", `{"rules": []}`)

	// What the verb writes, from the definition.
	readings := issueschema.ReadingsRecordDir
	writeJSON(readings+"/rdg-x/run.json", `{"run_id": "rdg-x", "position": "widening", "regime": "generative"}`)
	writeJSON(readings+"/rdg-x/refusal.json", `{"run_id": "rdg-x", "regime": "generative", "reason": "no items"}`)

	// What an operator edits.
	writeJSON(".abcd/config/x.json", `{"regime": "generative"}`)

	gitCmd(t, repo, "add", "-A")

	keys := walkConfigFiles(t, repo)

	operatorSurfaceReported := false
	for _, k := range keys {
		if strings.HasPrefix(k.schema, readings+"/") {
			t.Errorf("the configuration walk reported %s key %q: a run record is the output contract's "+
				"durable half, written by the verb from the position's definition, and no operator sets "+
				"a regime by editing one — the readings record family is not an operator surface",
				k.schema, k.path)
		}
		if k.schema == ".abcd/config/x.json" && strings.Contains(strings.ToLower(k.path), regimeToken) {
			operatorSurfaceReported = true
		}
	}
	if !operatorSurfaceReported {
		t.Fatal("the configuration walk did not report the regime key in .abcd/config/x.json; skipping a " +
			"record family must never narrow the walk over configuration, which is the surface it exists to read")
	}
}
