package cli

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/spf13/pflag"
)

// The input registry is held whole (spc-2610301431075964 step 1): a new
// filled-in input cannot ship without its shape, an entry cannot outlive its
// command or flag, and every example passes the check its writing command
// runs.

// TestInputRegistryBuilds holds every entry to a declaration that can be one:
// a root its Go type is, a version of at least one carried where its type
// declares a member for it, and an example.
func TestInputRegistryBuilds(t *testing.T) {
	shapes, err := InputShapes()
	if err != nil {
		t.Fatalf("the input registry does not build: %v", err)
	}
	if len(shapes) != len(inputRegistry) {
		t.Fatalf("InputShapes returned %d shapes for %d entries", len(shapes), len(inputRegistry))
	}
	for _, s := range shapes {
		if len(s.Fields) == 0 {
			t.Errorf("%s --%s: the field list is empty", s.Path, s.Flag)
		}
	}
}

// TestEveryJSONInputFlagIsRegistered walks the whole tree, hidden commands
// and hidden flags included: every flag ending `-json` is a registered input
// or a declared output, so an input added without its shape fails here.
func TestEveryJSONInputFlagIsRegistered(t *testing.T) {
	seen := 0
	for _, cmd := range allCommands(NewRootCommand()) {
		path := cmd.CommandPath()
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
			if !strings.HasSuffix(f.Name, "-json") {
				return
			}
			key := [2]string{path, f.Name}
			_, isInput := inputFor(path, f.Name)
			isOutput := outputOnlyJSONFlags[key]
			switch {
			case isInput && isOutput:
				t.Errorf("%s --%s is registered as an input and declared an output; it is one or the other", path, f.Name)
			case !isInput && !isOutput:
				t.Errorf("%s --%s ends -json and is in neither the input registry nor outputOnlyJSONFlags "+
					"(internal/surface/cli/inputs.go): a filled-in input ships with its Go type, its schema version, "+
					"an example and its check", path, f.Name)
			}
			seen++
		})
	}
	if seen == 0 {
		t.Fatal("the walk found no -json flag at all; it is not reading the tree")
	}
}

// TestInputRegistryNamesRealCommandsAndFlags refuses an entry, or an output
// declaration, whose command or flag the tree no longer has.
func TestInputRegistryNamesRealCommandsAndFlags(t *testing.T) {
	flags := map[[2]string]bool{}
	for _, cmd := range allCommands(NewRootCommand()) {
		path := cmd.CommandPath()
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) { flags[[2]string{path, f.Name}] = true })
	}
	for _, e := range inputRegistry {
		if !flags[[2]string{e.path, e.flag}] {
			t.Errorf("the input registry names %s --%s, which the command tree does not register", e.path, e.flag)
		}
		if !strings.HasSuffix(e.flag, "-json") {
			t.Errorf("the input registry names %s --%s, which does not end -json", e.path, e.flag)
		}
	}
	for key := range outputOnlyJSONFlags {
		if !flags[key] {
			t.Errorf("outputOnlyJSONFlags names %s --%s, which the command tree does not register", key[0], key[1])
		}
	}
}

// TestEveryInputExamplePassesItsCheck is the criterion's test: the example an
// agent is shown passes the owning package's own decoder and validator, and
// states the version the registry names, where the document carries one.
func TestEveryInputExamplePassesItsCheck(t *testing.T) {
	for _, e := range inputRegistry {
		t.Run(e.path+" --"+e.flag, func(t *testing.T) {
			if err := e.check(t.TempDir(), []byte(e.example)); err != nil {
				t.Fatalf("the example does not pass its own check: %v\n%s", err, e.example)
			}
			got, carried := exampleVersion(t, e.example, e.versionIn)
			switch {
			case e.versionIn == surface.VersionNone && carried:
				t.Errorf("the input carries no version, but the example states one (%d)", got)
			case e.versionIn != surface.VersionNone && !carried:
				t.Errorf("the example states no version in %s", e.versionIn)
			case carried && got != e.version:
				t.Errorf("the example states version %d; the registry names %d", got, e.version)
			}
		})
	}
}

// TestEveryInputCheckReadsTheVersion proves each check is the versioned
// decoder and not a lenient parse: the example moved one version past the
// one this binary reads is refused.
func TestEveryInputCheckReadsTheVersion(t *testing.T) {
	for _, e := range inputRegistry {
		if e.versionIn == surface.VersionNone {
			continue
		}
		t.Run(e.path+" --"+e.flag, func(t *testing.T) {
			var doc map[string]any
			if err := json.Unmarshal([]byte(e.example), &doc); err != nil {
				t.Fatal(err)
			}
			next := e.version + 1
			switch e.versionIn {
			case surface.VersionField:
				doc["schema_version"] = next
			case surface.VersionType:
				tag, _ := doc["_type"].(string)
				base, _, _ := strings.Cut(tag, "/")
				doc["_type"] = base + "/" + strconv.Itoa(next)
			}
			raw, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			if err := e.check(t.TempDir(), raw); err == nil {
				t.Errorf("the check accepted the example at version %d; it reads only %d", next, e.version)
			}
		})
	}
}

// exampleVersion is the version an example states where its input carries
// one.
func exampleVersion(t *testing.T, example, versionIn string) (int, bool) {
	t.Helper()
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(example), &doc); err != nil {
		// An array root states nothing.
		return 0, false
	}
	switch versionIn {
	case surface.VersionType:
		var tag string
		if err := json.Unmarshal(doc["_type"], &tag); err != nil {
			return 0, false
		}
		i := strings.LastIndex(tag, "/")
		n, err := strconv.Atoi(strings.TrimPrefix(tag[i+1:], "v"))
		return n, i >= 0 && err == nil
	default:
		raw, ok := doc["schema_version"]
		if !ok {
			return 0, false
		}
		var n int
		if err := json.Unmarshal(raw, &n); err != nil {
			t.Fatalf("the example's schema_version is not a number: %s", raw)
		}
		return n, true
	}
}

// TestCommandFactsCarryTheRegisteredInputs holds the facts to the registry:
// each command that reads an input names it with the registry's version, and
// a command reads no input the registry does not hold.
func TestCommandFactsCarryTheRegisteredInputs(t *testing.T) {
	facts := CommandFacts()
	for _, e := range inputRegistry {
		f, ok := factsFor(facts, e.path)
		if !ok {
			t.Errorf("no facts for %s", e.path)
			continue
		}
		found := false
		for _, in := range f.Inputs {
			if in.Flag != e.flag {
				continue
			}
			found = true
			want := surface.InputFact{Flag: e.flag, Kind: surface.InputJSON, Name: e.name, SchemaVersion: e.version, VersionIn: e.versionIn}
			if in != want {
				t.Errorf("%s: the facts name the input %+v, want %+v", e.path, in, want)
			}
		}
		if !found {
			t.Errorf("%s: the facts name no --%s input", e.path, e.flag)
		}
	}
	for _, f := range facts {
		for _, in := range f.Inputs {
			if in.Kind != surface.InputJSON {
				continue
			}
			if _, ok := inputFor(f.Path, in.Flag); !ok {
				t.Errorf("%s: the facts name --%s, which the registry does not hold", f.Path, in.Flag)
			}
		}
	}
}
