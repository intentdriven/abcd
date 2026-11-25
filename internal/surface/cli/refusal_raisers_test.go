package cli

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/refusal"
)

// refusal_raisers_test.go — a verb's own code names the commands that raise
// it (`abcd explain <code>` prints them as "Raised by"), and the list is held
// by driving each command into the refusal rather than kept by hand
// (spc-2610301431108175 step 1).

// raiserProbe is one refused call and the code it must carry. The code is
// written out as the string a reader matches on, not as the registry's
// constant, so a path that switches to another code fails here, and so does
// a code whose name changes: the release guard sees only a code that goes
// missing from the ledger (adr-2610301638558260).
type raiserProbe struct {
	args []string
	want string
	// exit is the exit the refusal leaves with: 2 when unset, or 3 for a
	// refusal that says someone else holds what the call needs.
	exit int
	// in, when set, readies the call before it runs: the directory it runs
	// in, the environment it runs under. It is set with from.
	in func(t *testing.T)
}

// from is p run after in readies it.
func (p raiserProbe) from(in func(t *testing.T)) raiserProbe {
	p.in = in
	return p
}

// inDir is a probe's in that runs the call in dir.
func inDir(dir string) func(t *testing.T) {
	return func(t *testing.T) { t.Chdir(dir) }
}

// probe is a refusal at exit 2 that carries want.
func probe(want string, args ...string) raiserProbe { return raiserProbe{args: args, want: want} }

// heldProbe is a refusal at exit 3, someone else holding what the call
// needs, that carries want.
func heldProbe(want string, args ...string) raiserProbe {
	return raiserProbe{args: args, want: want, exit: 3}
}

// wantExit is the exit p's refusal must leave with.
func (p raiserProbe) wantExit() int {
	if p.exit == 0 {
		return 2
	}
	return p.exit
}

// raiserProbeSet builds one family's probes: it makes the fixture its
// refusals need, under t (which a set may chdir), and returns the probes to
// run in it.
type raiserProbeSet func(t *testing.T) []raiserProbe

// raiserProbeSets holds every family's probe set by name. Each family's
// refusal_raisers_<family>_test.go registers its own, so two changes that
// each add a family's probes touch no common line.
var raiserProbeSets = map[string]raiserProbeSet{}

// addRaiserProbeSet adds set to sets under name, refusing a name already
// taken: a second family registering the same name would silently replace
// the first family's probes, and its codes would then go unprobed.
func addRaiserProbeSet(sets map[string]raiserProbeSet, name string, set raiserProbeSet) error {
	if _, taken := sets[name]; taken {
		return fmt.Errorf("two probe files register the probe set %q", name)
	}
	sets[name] = set
	return nil
}

// registerRaiserProbes registers a family's probe set; a probe file calls it
// from a package-level declaration, and a name registered twice panics, so
// the test binary does not start.
func registerRaiserProbes(name string, set raiserProbeSet) bool {
	if err := addRaiserProbeSet(raiserProbeSets, name, set); err != nil {
		panic(err)
	}
	return true
}

// raiserPath is the command path a probe's arguments run: the leading words
// that name commands of the tree.
func raiserPath(args []string) string {
	root := NewRootCommand()
	path := []string{}
	for _, w := range args {
		if findByPath(root, append(slices.Clone(path), w)) == nil {
			break
		}
		path = append(path, w)
	}
	return strings.Join(append([]string{"abcd"}, path...), " ")
}

// TestRaisedByIsWhatTheVerbsRaise drives every command a verb's own code
// names into that refusal, and every refusal it can reach in a fixture
// into its code, and holds the registry's RaisedBy to exactly the pairs
// seen: a listed command that never raises the code fails, and so does a
// command seen raising a code it is not listed for. Usage codes are raised
// by every command and list none. Each family's probes run in the fixture
// its own set makes, as a subtest, in the order of the sets' names.
func TestRaisedByIsWhatTheVerbsRaise(t *testing.T) {
	seen := map[refusal.Code]map[string]bool{}
	for _, name := range slices.Sorted(maps.Keys(raiserProbeSets)) {
		t.Run(name, func(t *testing.T) {
			for _, p := range raiserProbeSets[name](t) {
				if p.in != nil {
					p.in(t)
				}
				code, env, out := codedRefusal(t, p.args...)
				if code != p.wantExit() || env.Code != p.want {
					t.Errorf("abcd %s: exit %d code %q, want exit %d and %s:\n%s", strings.Join(p.args, " "), code, env.Code, p.wantExit(), p.want, out)
					continue
				}
				want := refusal.Code(p.want)
				if seen[want] == nil {
					seen[want] = map[string]bool{}
				}
				seen[want][raiserPath(p.args)] = true
			}
		})
	}
	if t.Failed() {
		return
	}
	for _, e := range refusal.Entries() {
		if strings.HasPrefix(string(e.Code), "usage/") {
			if len(e.RaisedBy) != 0 {
				t.Errorf("%s: a usage code is raised by every command and lists none, got %v", e.Code, e.RaisedBy)
			}
			continue
		}
		var got []string
		for path := range seen[e.Code] {
			got = append(got, path)
		}
		slices.Sort(got)
		listed := slices.Sorted(slices.Values(e.RaisedBy))
		if !slices.Equal(got, listed) {
			t.Errorf("%s: RaisedBy lists %v, but the commands seen raising it are %v", e.Code, listed, got)
		}
	}
}

// TestTwoProbeFilesCannotRegisterOneSetName: a probe set's name is taken
// once, so a family cannot replace another's probes by reusing its name.
func TestTwoProbeFilesCannotRegisterOneSetName(t *testing.T) {
	sets := map[string]raiserProbeSet{}
	none := func(*testing.T) []raiserProbe { return nil }
	if err := addRaiserProbeSet(sets, "demo", none); err != nil {
		t.Fatalf("the first registration was refused: %v", err)
	}
	if err := addRaiserProbeSet(sets, "demo", none); err == nil {
		t.Fatal("a second registration of the same name was accepted")
	}
	if len(raiserProbeSets) == 0 {
		t.Fatal("no probe set is registered")
	}
}
