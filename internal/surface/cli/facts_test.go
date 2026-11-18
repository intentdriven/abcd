package cli

import (
	"reflect"
	"slices"
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
// its allowed values reach `abcd explain` (itd-2610301346085246 ac-1); a
// list flag whose members, or one segment of whose values, come from a closed
// set is declared through a list variant and held by listSets instead; the
// flags left out take an open word (--label) or no value at all
// (--hold-moscow, reserved and refused whole).
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
			if got == nil || flagEach(f) != "" {
				return // a list variant's set is listSets' to hold
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

// listSet is one list flag declared through a list variant of enumFlag: the
// slice its reader checks, what in each value the slice applies to, a value
// whose checked member or segment is a given word, and the reader itself.
type listSet struct {
	vals  []string
	each  string
	value func(word string) string
	read  func(value string) error
}

// listSets is every list flag declared through enumListFlag, enumSliceFlag or
// enumSegmentFlag, with its reader (iss-2611162111195582). A new one is added
// here, so its annotation, its help, its facts and its reader are held to one
// slice, and its reader is shown to refuse a value outside it.
func listSets() map[[2]string]listSet {
	whole := func(word string) string { return word }
	sets := map[[2]string]listSet{
		{"abcd ahoy install", "readme-parts"}: {
			vals: ahoy.ReadmePartsMembers(), each: surface.EachMember, value: whole,
			read: func(v string) error { _, err := ahoy.ParseReadmeParts(v); return err },
		},
		{"abcd ahoy install", "install-tool"}: {
			vals: ahoy.DependencyTools, each: surface.EachMember, value: whole,
			read: func(v string) error { _, err := installToolNames([]string{v}); return err },
		},
	}
	// Each delegating verb's tier --route, by the agent it dispatches; the
	// reader is the parse resolve hands every value to.
	for path, agent := range map[string]string{
		"abcd disembark graveyard":       graveyardAgent,
		"abcd disembark press-release":   pressReleaseAgent,
		"abcd disembark principles":      principlesAgent,
		"abcd disembark review":          reviewAgent,
		"abcd intent audit":              auditAgent,
		"abcd intent audit ingest":       auditAgent,
		"abcd intent consistency":        auditAgent,
		"abcd intent consistency ingest": auditAgent,
		"abcd launch ship":               changelogAgent,
		"abcd reading ingest":            readingAgentPrefix + string(reading.Positions()[0]),
	} {
		sets[[2]string{path, routeFlagName}] = listSet{
			vals: enumWords(oracle.Tiers()), each: routeTierSegment,
			value: func(tier string) string { return agent + "=" + tier },
			read: func(v string) error {
				_, err := oracle.ParseRoutes([]string{v}, []string{agent}, nil)
				return err
			},
		}
	}
	return sets
}

// TestListFlagAnnotationsAreTheReadersSlices holds each list flag's closed
// set, as its annotation, its help line and its facts carry it, to the slice
// its reader checks, says in the facts what in each value the set applies to,
// and holds the reader to accepting every word of the set and refusing one
// outside it. Every list-variant flag in the tree has an entry in listSets.
func TestListFlagAnnotationsAreTheReadersSlices(t *testing.T) {
	want := listSets()
	facts := CommandFacts()
	annotated := 0
	for _, cmd := range allCommands(NewRootCommand()) {
		path := cmd.CommandPath()
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
			each := flagEach(f)
			if each == "" {
				return
			}
			annotated++
			set, ok := want[[2]string{path, f.Name}]
			if !ok {
				t.Errorf("%s --%s carries a closed set for each %s, and no entry in listSets names it", path, f.Name, each)
				return
			}
			if got := flagAllowed(f); !reflect.DeepEqual(got, set.vals) {
				t.Errorf("%s --%s: annotation %v, its reader checks %v", path, f.Name, got, set.vals)
			}
			if each != set.each {
				t.Errorf("%s --%s: the set applies to each %q, want %q", path, f.Name, each, set.each)
			}
			if !strings.Contains(f.Usage, enumHelp(set.vals)) {
				t.Errorf("%s --%s: the help %q does not render the set %q", path, f.Name, f.Usage, enumHelp(set.vals))
			}
			if surface.FlagKind(f.Value.Type()) != surface.FlagList && f.Value.Type() != "string" {
				t.Errorf("%s --%s: a list variant declared a %s flag", path, f.Name, f.Value.Type())
			}
			ff, _ := factsFor(facts, path)
			for _, fl := range ff.Flags {
				if fl.Name == f.Name && (!reflect.DeepEqual(fl.Allowed, set.vals) || fl.Each != set.each) {
					t.Errorf("%s --%s: the facts allow %v for each %q, its reader checks %v for each %q",
						path, f.Name, fl.Allowed, fl.Each, set.vals, set.each)
				}
			}
			for _, word := range set.vals {
				if err := set.read(set.value(word)); err != nil {
					t.Errorf("%s --%s: the reader refuses %q, a word of the set: %v", path, f.Name, set.value(word), err)
				}
			}
			if err := set.read(set.value("not-in-the-set")); err == nil {
				t.Errorf("%s --%s: the reader accepts %q, outside the set", path, f.Name, set.value("not-in-the-set"))
			}
		})
	}
	if annotated != len(want) {
		t.Errorf("the tree carries %d list flags with a closed set; listSets names %d", annotated, len(want))
	}
}

// TestReadmePartsChoicesNamesTheMembers holds the summary's spelling of what
// --readme-parts takes to the set the flag declares, in any order, so the hint
// a run prints cannot name a part the parse refuses or leave one out.
func TestReadmePartsChoicesNamesTheMembers(t *testing.T) {
	got := strings.Split(ahoy.ReadmePartsChoices, "|")
	want := ahoy.ReadmePartsMembers()
	slices.Sort(got)
	slices.Sort(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ReadmePartsChoices names %v; the flag declares %v", got, want)
	}
}

// TestListFlagVariantsRecordTheSetAndWhatItAppliesTo pins each list variant's
// help line, its copy of the caller's slice, and what it records the set as
// applying to; enumFlag records no such thing.
func TestListFlagVariantsRecordTheSetAndWhatItAppliesTo(t *testing.T) {
	vals := []string{"a", "b"}
	cmd := &cobra.Command{Use: "x"}
	var one, comma string
	var slice, array []string
	enumFlag(cmd.Flags(), &one, "pick", "", "pick: ", vals, "")
	enumListFlag(cmd.Flags(), &comma, "parts", "parts, each one of ", vals, ".")
	enumSliceFlag(cmd.Flags(), &slice, "tool", "a tool, one of ", vals, " (repeatable)")
	enumSegmentFlag(cmd.Flags(), &array, "route", "tier", "route: <agent>=<tier>, tier one of ", vals, "")
	for _, tc := range []struct {
		name, help, each, kind string
	}{
		{"pick", "pick: a | b", "", "string"},
		{"parts", "parts, each one of a | b.", surface.EachMember, "string"},
		{"tool", "a tool, one of a | b (repeatable)", surface.EachMember, "stringSlice"},
		{"route", "route: <agent>=<tier>, tier one of a | b", "tier", "stringArray"},
	} {
		f := cmd.Flags().Lookup(tc.name)
		if f.Usage != tc.help || flagEach(f) != tc.each || f.Value.Type() != tc.kind {
			t.Errorf("--%s: help %q, each %q, type %s; want %q, %q, %s", tc.name, f.Usage, flagEach(f), f.Value.Type(), tc.help, tc.each, tc.kind)
		}
	}
	vals[0] = "z"
	for _, name := range []string{"pick", "parts", "tool", "route"} {
		if got := flagAllowed(cmd.Flags().Lookup(name)); !reflect.DeepEqual(got, []string{"a", "b"}) {
			t.Errorf("--%s: the annotation followed a write to the caller's slice: %v", name, got)
		}
	}
	if err := cmd.Flags().Parse([]string{"--parts", "a,b", "--tool", "a,b", "--tool", "a", "--route", "x=a,b"}); err != nil {
		t.Fatal(err)
	}
	if comma != "a,b" || !reflect.DeepEqual(slice, []string{"a", "b", "a"}) || !reflect.DeepEqual(array, []string{"x=a,b"}) {
		t.Errorf("the list variants read %q, %v, %v; want the value whole, split on commas and repeated, and repeated whole", comma, slice, array)
	}
}
