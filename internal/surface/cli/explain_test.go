package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
)

// explain_test.go — `abcd explain` (spc-2610301431075964 step 2). An agent
// asks the running binary how to call a command and gets the command's facts
// in at most 40 lines, every line drawn from the one source of them; asks for
// one input and gets the shape and an example at the version the binary
// reads, which the owning package's own check accepts.

// explainRun runs `abcd explain <args>` through the front door the binary
// uses, under a home of its own, and returns the exit code and both streams.
func explainRun(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(append([]string{"explain"}, args...), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// explainWords is a command path as explain takes it: the words below the
// root, or the root's own name for the root.
func explainWords(path string) []string {
	words := strings.Fields(path)
	if len(words) == 1 {
		return words
	}
	return words[1:]
}

// The first criterion: every command, visible or hidden, is answered in at
// most 40 lines, and the answer's flags, inputs, example and sub-commands are
// the command's facts. The --json answer is the facts themselves.
func TestExplainAnswersEveryCommandFromItsFactsWithinTheBound(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	facts := CommandFacts()
	if len(facts) < 100 {
		t.Fatalf("the facts hold %d commands; the walk is not reading the tree", len(facts))
	}
	for _, f := range facts {
		t.Run(f.Path, func(t *testing.T) {
			code, out, errOut := explainRun(t, explainWords(f.Path)...)
			if code != 0 {
				t.Fatalf("exit %d, stderr %q", code, errOut)
			}
			lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
			if len(lines) > explainMaxLines {
				t.Errorf("the answer runs to %d lines, over the bound of %d:\n%s", len(lines), explainMaxLines, out)
			}
			if !strings.HasPrefix(lines[0], f.Path) || !strings.Contains(lines[0], f.Sentence) {
				t.Errorf("the first line does not name the command and its sentence %q: %q", f.Sentence, lines[0])
			}
			if hidden := explainHidden(facts, f.Path); hidden != strings.Contains(lines[0], "(hidden)") {
				t.Errorf("hidden is %v, but the first line reads %q", hidden, lines[0])
			}
			if !strings.Contains(out, "\nUsage: "+f.Usage+"\n") {
				t.Errorf("the answer does not carry the usage line %q:\n%s", f.Usage, out)
			}
			// A list cut at the bound shows its first flags alone
			// (TestExplainCutsAnOverlongFlagListAtTheBound holds the cut).
			cut := strings.Contains(out, " more; `abcd explain ")
			for _, fl := range f.Flags {
				line := "\n  " + explainFlagLine(fl) + "\n"
				shown := strings.Contains(out, line)
				switch {
				case fl.Hidden && shown:
					t.Errorf("the hidden flag --%s is shown:\n%s", fl.Name, out)
				case !fl.Hidden && !shown && !cut:
					t.Errorf("the flag --%s is not shown as %q:\n%s", fl.Name, strings.TrimSpace(line), out)
				}
			}
			for _, in := range f.Inputs {
				if !strings.Contains(out, "\n  "+explainInputLine(f.Path, in)+"\n") {
					t.Errorf("the answer does not name the input --%s:\n%s", in.Flag, out)
				}
			}
			if f.Example != "" && !strings.Contains(out, "\nExample:\n  "+strings.Split(f.Example, "\n")[0]) {
				t.Errorf("the answer does not carry the example %q:\n%s", f.Example, out)
			}
			for _, sub := range f.Subcommands {
				if !strings.Contains(lines[len(lines)-1], sub) {
					t.Errorf("the last line does not name the sub-command %q: %q", sub, lines[len(lines)-1])
				}
			}

			code, out, errOut = explainRun(t, append(explainWords(f.Path), "--json")...)
			if code != 0 {
				t.Fatalf("--json: exit %d, stderr %q", code, errOut)
			}
			var doc CommandFactsDocument
			if err := json.Unmarshal([]byte(out), &doc); err != nil {
				t.Fatalf("--json does not decode: %v\n%s", err, out)
			}
			if doc.FormatVersion != surface.FactsFormatVersion {
				t.Errorf("format_version %d, want %d", doc.FormatVersion, surface.FactsFormatVersion)
			}
			if len(doc.Commands) != 1 || !reflect.DeepEqual(doc.Commands[0], explainNormal(f)) {
				t.Errorf("--json is not the command's facts:\ngot  %+v\nwant %+v", doc.Commands, f)
			}
		})
	}
}

// explainNormal is f as it comes back from a JSON round trip: an empty
// collection the encoder omits decodes as nil.
func explainNormal(f surface.Facts) surface.Facts {
	raw, _ := json.Marshal(f)
	var out surface.Facts
	_ = json.Unmarshal(raw, &out)
	return out
}

// A flag's closed set is said as what the flag takes: one of the set for a
// flag that takes one value, each one of it for a list whose members are
// checked, and the named segment of each value for a list whose values are
// checked by one segment, so a list is never read as one value of the set.
func TestExplainSaysWhatAClosedSetAppliesTo(t *testing.T) {
	for _, tc := range []struct {
		fl   surface.FlagFact
		want string
	}{
		{surface.FlagFact{Name: "pick", Type: surface.FlagString, Allowed: []string{"a", "b"}},
			"--pick <string> [one of a | b]"},
		{surface.FlagFact{Name: "parts", Type: surface.FlagString, Allowed: []string{"a", "b"}, Each: surface.EachMember},
			"--parts <string> [each one of a | b]"},
		{surface.FlagFact{Name: "route", Type: surface.FlagList, Allowed: []string{"a", "b"}, Each: "tier"},
			"--route <list> [each value's tier one of a | b]"},
		{surface.FlagFact{Name: "open", Type: surface.FlagList},
			"--open <list>"},
	} {
		if got := explainFlagLine(tc.fl); got != tc.want {
			t.Errorf("explainFlagLine(%+v) = %q, want %q", tc.fl, got, tc.want)
		}
	}
}

// A command whose flags would run past the bound shows the first of them and
// one line saying how many more there are and how to get them all.
func TestExplainCutsAnOverlongFlagListAtTheBound(t *testing.T) {
	f := surface.Facts{
		Path:     "abcd widget",
		Sentence: "Make a widget: Writes the widget; refuses a negative size.",
		Usage:    "abcd widget [flags]",
		Inputs:   []surface.InputFact{{Flag: "answers", Kind: surface.InputAnswers, Name: "interview answers"}},
		Example:  "abcd widget --flag-00",
	}
	for i := range 60 {
		f.Flags = append(f.Flags, surface.FlagFact{Name: fmt.Sprintf("flag-%02d", i), Type: surface.FlagBool})
	}
	inherited := []explainInheritedFlag{{flag: surface.FlagFact{Name: "json", Type: surface.FlagBool, Persistent: true}, from: "abcd"}}
	lines := explainLines(f, false, inherited)
	if len(lines) != explainMaxLines {
		t.Fatalf("the answer runs to %d lines, want exactly %d:\n%s", len(lines), explainMaxLines, strings.Join(lines, "\n"))
	}
	shown := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "  --flag-") {
			shown++
		}
	}
	more := 61 - shown // 60 own flags and one inherited
	want := fmt.Sprintf("  and %d more; `abcd explain widget --json` lists every flag", more)
	if !strings.Contains(strings.Join(lines, "\n"), want) {
		t.Errorf("no line says %q:\n%s", want, strings.Join(lines, "\n"))
	}
	if lines[len(lines)-1] != "  abcd widget --flag-00" || lines[len(lines)-4] != "Inputs:" {
		t.Errorf("the inputs and the example are not kept whole after the cut:\n%s", strings.Join(lines, "\n"))
	}
}

// A hidden command is answered, marked on its first line, so a refusal on the
// hook plane can name explain as its next step; so is a sub-command of a
// hidden parent.
func TestExplainAnswersAHiddenCommandMarked(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, args := range [][]string{{"hook"}, {"hook", "session-start"}} {
		code, out, errOut := explainRun(t, args...)
		if code != 0 {
			t.Fatalf("%v: exit %d, stderr %q", args, code, errOut)
		}
		first, _, _ := strings.Cut(out, "\n")
		if want := "abcd " + strings.Join(args, " ") + " (hidden)"; !strings.HasPrefix(first, want) {
			t.Errorf("%v: the first line is %q, want it to open %q", args, first, want)
		}
	}
}

// A path the tree does not have is refused with exit 2, naming the nearest
// real path, or the sub-commands where the path went wrong, and nothing goes
// to stdout. The refusal does not repeat the word, except where the word is
// part of a real command name the refusal names, which is abcd's own text and
// comes through the error boundary whole.
func TestExplainRefusesAPathTheTreeDoesNotHave(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, tc := range []struct {
		args  []string
		want  string
		typed string
	}{
		{[]string{"ideate", "recrod"}, "the nearest real path is `abcd ideate record`", "recrod"},
		{[]string{"disembark", "graveyardz"}, "the nearest real path is `abcd disembark graveyard`", "graveyardz"},
		{[]string{"ideate", "zzzzzzzzzz"}, "its sub-commands are record", "zzzzzzzzzz"},
		{[]string{"decide", "something-else"}, "`abcd decide` takes no sub-command", "something-else"},
		{[]string{"explainz"}, "the nearest real path is `abcd explain`", "explainz"},
		{[]string{"library", "cite-chec"}, "the nearest real path is `abcd library cite-check`", ""},
		{[]string{"hook", "session-star"}, "the nearest real path is `abcd hook session-start`", ""},
		{[]string{"hook", "session-s"}, "session-end, session-start", ""},
	} {
		code, out, errOut := explainRun(t, tc.args...)
		if code != 2 {
			t.Errorf("%v: exit %d, want 2 (stderr %q)", tc.args, code, errOut)
		}
		if out != "" {
			t.Errorf("%v: a refusal wrote to stdout: %q", tc.args, out)
		}
		if !strings.Contains(errOut, tc.want) {
			t.Errorf("%v: the refusal does not say %q: %q", tc.args, tc.want, errOut)
		}
		if strings.Contains(errOut, "not repeated here") {
			t.Errorf("%v: the error boundary withheld part of the refusal: %q", tc.args, errOut)
		}
		if tc.typed != "" && strings.Contains(errOut, tc.typed) {
			t.Errorf("%v: the refusal repeats the word typed: %q", tc.args, errOut)
		}
	}
}

// The root answers by its own name, and a leading root name is read as the
// root, so `abcd explain abcd ideate record` is `abcd explain ideate record`.
func TestExplainReadsALeadingRootName(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	code, root, _ := explainRun(t, "abcd")
	if code != 0 || !strings.HasPrefix(root, "abcd — ") {
		t.Errorf("explain abcd: exit %d, %q", code, root)
	}
	_, plain, _ := explainRun(t, "ideate", "record")
	_, named, _ := explainRun(t, "abcd", "ideate", "record")
	if plain == "" || plain != named {
		t.Errorf("a leading root name changed the answer:\n%s\n---\n%s", plain, named)
	}
}

// Bare, explain prints its own help and writes nothing else.
func TestExplainBarePrintsItsHelp(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	code, out, errOut := explainRun(t)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	sentence, _ := surface.SentenceFor("abcd explain")
	if !strings.HasPrefix(out, sentence) || !strings.Contains(out, "Usage:") {
		t.Errorf("bare explain is not its help:\n%s", out)
	}
}

// The second criterion: for every registered input, --input --json names the
// owning package's schema-version constant, and the example it prints passes
// the owning package's own decoder and validator. The text form says the same
// version and carries the same example.
func TestExplainInputGivesTheShapeAtTheVersionTheBinaryReads(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if len(inputRegistry) == 0 {
		t.Fatal("the input registry is empty")
	}
	for _, e := range inputRegistry {
		t.Run(e.path+" --"+e.flag, func(t *testing.T) {
			args := append(explainWords(e.path), "--input", e.flag, "--json")
			code, out, errOut := explainRun(t, args...)
			if code != 0 {
				t.Fatalf("exit %d, stderr %q", code, errOut)
			}
			var shape surface.InputShape
			if err := json.Unmarshal([]byte(out), &shape); err != nil {
				t.Fatalf("--input --json does not decode: %v\n%s", err, out)
			}
			if shape.Path != e.path || shape.Flag != e.flag {
				t.Errorf("the shape is %s --%s", shape.Path, shape.Flag)
			}
			if shape.SchemaVersion != e.version || shape.VersionIn != e.versionIn {
				t.Errorf("schema version %d in %q; the owning package's constant is %d in %q",
					shape.SchemaVersion, shape.VersionIn, e.version, e.versionIn)
			}
			if len(shape.Fields) == 0 {
				t.Error("the shape lists no field")
			}
			if err := e.check(t.TempDir(), []byte(shape.Example)); err != nil {
				t.Errorf("the example explain prints does not pass the command's own check: %v\n%s", err, shape.Example)
			}

			code, text, errOut := explainRun(t, append(explainWords(e.path), "--input", "--"+e.flag)...)
			if code != 0 {
				t.Fatalf("text: exit %d, stderr %q", code, errOut)
			}
			first, _, _ := strings.Cut(text, "\n")
			if !strings.Contains(first, explainVersionText(e.version, e.versionIn)) {
				t.Errorf("the first line does not state the version: %q", first)
			}
			for _, line := range strings.Split(strings.TrimRight(e.example, "\n"), "\n") {
				if !strings.Contains(text, "\n  "+line+"\n") {
					t.Errorf("the text form does not carry the example's line %q", line)
					break
				}
			}
			for _, fd := range shape.Fields {
				if !strings.Contains(text, "\n  "+fd.Name+" ("+fd.Type) {
					t.Errorf("the text form does not list the field %q", fd.Name)
				}
			}
		})
	}
}

// --input is refused with exit 2 for a flag the command reads no input
// through, for an interview's answers file, and with no command named.
func TestExplainInputRefusals(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"ideate", "record", "--input", "findings-json"}, "the inputs it reads are --verdict-json"},
		{[]string{"decide", "--input", "verdict-json"}, "`abcd decide` reads no filled-in input"},
		{[]string{"reflect", "write", "--input", "answers"}, "an interview's answers file"},
		{[]string{"--input", "verdict-json"}, "name the command first"},
	} {
		code, out, errOut := explainRun(t, tc.args...)
		if code != 2 || out != "" || !strings.Contains(errOut, tc.want) {
			t.Errorf("%v: exit %d, stdout %q, stderr %q; want exit 2 saying %q", tc.args, code, out, errOut, tc.want)
		}
	}
}
