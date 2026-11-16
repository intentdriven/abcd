package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/afterupdate"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/checklist"
	"github.com/intentdriven/abcd/internal/core/condition"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/interview"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/provenance"
	"github.com/intentdriven/abcd/internal/core/reading"
	coresession "github.com/intentdriven/abcd/internal/core/session"
	"github.com/intentdriven/abcd/internal/core/site"
	"github.com/intentdriven/abcd/internal/core/source"
	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// The command facts (spc-2610301431075964 step 1) come from the same walk as
// the compatibility snapshot, and every closed set they name is the slice the
// flag's reader checks against.

// TestCommandFactsCoverTheSnapshotTree holds the two halves of the one walk
// to one tree: the same commands, each with the same flags and sentence.
func TestCommandFactsCoverTheSnapshotTree(t *testing.T) {
	commands, facts := surfaceWalk(NewRootCommand())
	if len(commands) != len(facts) {
		t.Fatalf("the walk yielded %d snapshot entries and %d facts", len(commands), len(facts))
	}
	canonical := surface.CanonicalFacts(facts)
	for _, c := range commands {
		f, ok := factsFor(canonical, c.Path)
		if !ok {
			t.Errorf("%s is in the snapshot and has no facts", c.Path)
			continue
		}
		if f.Hidden != c.Hidden || f.Sentence != c.Sentence {
			t.Errorf("%s: facts say hidden=%v sentence=%q, the snapshot hidden=%v sentence=%q",
				c.Path, f.Hidden, f.Sentence, c.Hidden, c.Sentence)
		}
		// Both sides sorted by name: the snapshot through its constructor,
		// the facts through CanonicalFacts.
		var snapFlags, factFlags []string
		for _, fl := range f.Flags {
			factFlags = append(factFlags, fl.Name)
		}
		snap := surface.NewSnapshot([]surface.Command{c}, nil).Commands[0]
		for _, fl := range snap.Flags {
			snapFlags = append(snapFlags, fl.Name)
		}
		if !reflect.DeepEqual(snapFlags, factFlags) {
			t.Errorf("%s: the snapshot records flags %v, the facts %v", c.Path, snapFlags, factFlags)
		}
	}
}

// TestCommandFactsReadTheCommand holds each command's facts to what its help
// prints: the usage line, the example, and the visible sub-commands.
func TestCommandFactsReadTheCommand(t *testing.T) {
	facts := CommandFacts()
	for _, cmd := range allCommands(NewRootCommand()) {
		f, ok := factsFor(facts, cmd.CommandPath())
		if !ok {
			t.Errorf("no facts for %s", cmd.CommandPath())
			continue
		}
		cmd.LocalFlags() // merge the inherited flags, as executing the command does
		if f.Usage != cmd.UseLine() {
			t.Errorf("%s: usage %q, the help prints %q", f.Path, f.Usage, cmd.UseLine())
		}
		if f.Example != strings.TrimSpace(cmd.Example) {
			t.Errorf("%s: example %q, the help prints %q", f.Path, f.Example, cmd.Example)
		}
		var subs []string
		for _, c := range cmd.Commands() {
			if !c.Hidden {
				subs = append(subs, c.Name())
			}
		}
		if len(subs) != len(f.Subcommands) {
			t.Errorf("%s: sub-commands %v, the tree has %v", f.Path, f.Subcommands, subs)
		}
	}
}

// TestCommandFactsFlagDetail pins the parts of a flag fact a walk could get
// wrong: the folded type, a default kept only when it is not the zero value,
// a persistent flag recorded once where it is declared, and an interview's
// answers file named as an input.
func TestCommandFactsFlagDetail(t *testing.T) {
	facts := CommandFacts()
	flag := func(path, name string) surface.FlagFact {
		t.Helper()
		f, ok := factsFor(facts, path)
		if !ok {
			t.Fatalf("no facts for %s", path)
		}
		for _, fl := range f.Flags {
			if fl.Name == name {
				return fl
			}
		}
		t.Fatalf("%s has no --%s in its facts", path, name)
		return surface.FlagFact{}
	}
	if got := flag("abcd", "json"); !got.Persistent || got.Type != surface.FlagBool || got.Default != "" {
		t.Errorf("abcd --json: %+v, want a persistent bool with no default recorded", got)
	}
	if got := flag("abcd intent consistency", "max-chunks"); got.Type != surface.FlagInt || got.Default != "8" {
		t.Errorf("abcd intent consistency --max-chunks: %+v, want an int defaulting to 8", got)
	}
	if got := flag("abcd intent consistency ingest", "findings-json"); got.Type != surface.FlagList {
		t.Errorf("abcd intent consistency ingest --findings-json: %+v, want a list (it is repeatable)", got)
	}
	for _, f := range facts {
		if f.Path == "abcd" {
			continue
		}
		for _, fl := range f.Flags {
			if fl.Name == "json" && fl.Persistent {
				t.Errorf("%s records the root's persistent --json again; it is recorded where it is declared", f.Path)
			}
		}
	}
	interview, _ := factsFor(facts, "abcd intent interview")
	found := false
	for _, in := range interview.Inputs {
		if in.Flag == answersFlag && in.Kind == surface.InputAnswers && in.SchemaVersion == 0 {
			found = true
		}
	}
	if !found {
		t.Errorf("abcd intent interview's facts name no answers input: %+v", interview.Inputs)
	}
}

// closedSets is every flag declared through enumFlag, with the slice its
// reader checks a value against. A new closed-set flag is added here, so the
// annotation, the help and the reader are held to one slice. A flag whose
// values are a closed set its reader checks is declared through enumFlag, so
// its allowed values reach `abcd explain` (itd-2610301346085246 ac-1); the
// flags left out take a list (--readme-parts, --route), an open word
// (--label) or no value at all (--hold-moscow, reserved and refused whole).
func closedSets() map[[2]string][]string {
	return map[[2]string][]string{
		{"abcd capture", "severity"}:                issueschema.Severities,
		{"abcd capture", "category"}:                issueschema.Categories,
		{"abcd capture", "source"}:                  issueschema.Sources,
		{"abcd implement outage record", "service"}: implement.OutageServices(),
		{"abcd implement outage record", "kind"}:    implement.OutageKinds(),
		{"abcd implement load", "site"}:             implement.LoadSites(),
		{"abcd library add", "permission"}:          source.Permissions,
		{"abcd library declassify", "permission"}:   source.Permissions,
		{"abcd library ledger", "influence"}:        source.Influences,

		{"abcd", "view"}:   boardViews,
		{"abcd", "format"}: boardForms,

		{"abcd ahoy connect", "home"}:               oracle.KeyHomes(),
		{"abcd ahoy credential", "home"}:            credential.Homes(),
		{"abcd ahoy install", "visibility"}:         ahoy.VisibilityChoices(),
		{"abcd ahoy install", "docs-target"}:        ahoy.DocsTargetWritable(),
		{"abcd ahoy install", "oracle-backend"}:     ahoy.OracleBackendChoices(),
		{"abcd ahoy install", "scan-deep"}:          ahoy.ScanDeepChoices(),
		{"abcd ahoy install", "live-sessions"}:      ahoy.LiveSessionsValues(),
		{"abcd ahoy install", "answered-in"}:        interview.AnsweredInPlaces(),
		{"abcd intent interview", "answered-in"}:    interview.AnsweredInPlaces(),
		{"abcd reflect interview", "answered-in"}:   interview.AnsweredInPlaces(),
		{"abcd banlist add", "severity"}:            banlist.Severities(),
		{"abcd capture disposition", "state"}:       issueschema.DispositionStates,
		{"abcd capture resolve", "impact"}:          changelog.ImpactNames(),
		{"abcd intent", "impact"}:                   changelog.IntentImpactNames(),
		{"abcd intent plan", "impact"}:              changelog.IntentImpactNames(),
		{"abcd spec close", "impact"}:               changelog.IntentImpactNames(),
		{"abcd intent", "production-mode"}:          provenance.ModeNames(),
		{"abcd intent plan", "production-mode"}:     provenance.ModeNames(),
		{"abcd spec close", "production-mode"}:      provenance.ModeNames(),
		{"abcd capture", "production-mode"}:         provenance.ModeNames(),
		{"abcd capture promote", "production-mode"}: provenance.ModeNames(),
		{"abcd capture resolve", "production-mode"}: provenance.ModeNames(),
		{"abcd capture wontfix", "production-mode"}: provenance.ModeNames(),
		{"abcd checklist set", "state"}:             enumWords(checklist.States()),
		{"abcd history capture", "kind"}:            history.Routes(),
		{"abcd history reconstruct", "mode"}:        history.ReconstructModes(),
		{"abcd implement join", "role"}:             enumWords(implement.Roles()),
		{"abcd implement outage probe", "model"}:    probeVerdicts,
		{"abcd intent condition", "disposition"}:    condition.Enum,
		{"abcd intent reclassify", "kind"}:          intent.ReclassifyKinds(),
		{"abcd launch manifests", "tree"}:           launch.LockstepTrees(),
		{"abcd reading assemble", "position"}:       enumWords(reading.Positions()),
		{"abcd session setup", "where"}:             coresession.Wheres(),
		{"abcd site setup", "provider"}:             site.Providers(),
		{"abcd update news", "answer"}:              afterupdate.Answers(),
	}
}

// TestEnumFlagAnnotationsAreTheReadersSlices holds each closed-set flag's
// annotation, its facts and its help line to the slice its reader checks, and
// every annotated flag in the tree to an entry above.
func TestEnumFlagAnnotationsAreTheReadersSlices(t *testing.T) {
	want := closedSets()
	facts := CommandFacts()
	annotated := 0
	for _, cmd := range allCommands(NewRootCommand()) {
		path := cmd.CommandPath()
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
			got := flagAllowed(f)
			if got == nil {
				return
			}
			annotated++
			slice, ok := want[[2]string{path, f.Name}]
			if !ok {
				t.Errorf("%s --%s carries a closed set no entry in closedSets names", path, f.Name)
				return
			}
			if !reflect.DeepEqual(got, slice) {
				t.Errorf("%s --%s: annotation %v, its reader checks %v", path, f.Name, got, slice)
			}
			if !strings.Contains(f.Usage, enumHelp(slice)) {
				t.Errorf("%s --%s: the help %q does not render the set %q", path, f.Name, f.Usage, enumHelp(slice))
			}
			ff, _ := factsFor(facts, path)
			for _, fl := range ff.Flags {
				if fl.Name == f.Name && !reflect.DeepEqual(fl.Allowed, slice) {
					t.Errorf("%s --%s: the facts allow %v, its reader checks %v", path, f.Name, fl.Allowed, slice)
				}
			}
		})
	}
	if annotated != len(want) {
		t.Errorf("the tree carries %d closed-set flags; closedSets names %d", annotated, len(want))
	}
}

// TestEnumFlagCopiesTheSlice keeps the annotation from aliasing the caller's
// slice, so a later write to the backing array cannot change what the facts
// say without changing what the reader checks.
func TestEnumFlagCopiesTheSlice(t *testing.T) {
	vals := []string{"a", "b"}
	cmd := &cobra.Command{Use: "x"}
	var v string
	enumFlag(cmd.Flags(), &v, "pick", "", "pick: ", vals, " (default a)")
	f := cmd.Flags().Lookup("pick")
	if f.Usage != "pick: a | b (default a)" {
		t.Errorf("help %q", f.Usage)
	}
	vals[0] = "z"
	if got := flagAllowed(f); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("the annotation followed a write to the caller's slice: %v", got)
	}
}

// factsFor is the facts for one command path out of a set.
func factsFor(all []surface.Facts, path string) (surface.Facts, bool) {
	for _, f := range all {
		if f.Path == path {
			return f, true
		}
	}
	return surface.Facts{}, false
}
