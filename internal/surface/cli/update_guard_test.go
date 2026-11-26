package cli

import (
	"regexp"
	"testing"

	"github.com/spf13/pflag"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/guard"
)

// TestTheGuardReadsTheUpdateOperandAsTheCommandDoes: a word the update command
// takes as a release tag is one the guard leaves to the person, and a word it
// refuses as a mistyped sub-verb, or as no tag, is left alone.
func TestTheGuardReadsTheUpdateOperandAsTheCommandDoes(t *testing.T) {
	asJSON := false
	cmd := newUpdateCommand(&asJSON)
	reg := guard.Defaults()
	taken := 0
	for _, word := range []string{
		"v0.7.0", "v1", "v0.7.0-rc.1", "v0.7.0+build.2", "V0.7.0", "0.7.0", "1", "v0", "v0-x", "vx", "v", "v.1",
		"latest", "nightly", "news", "new", "check", "x-1", "Latest", "x_1", "x.y", "README.md",
		"./v0.7.0", "a/b", ".hidden", "+x", "_x", "v1,2", "v1:2", "v1@2", "v1=2", "v1%2",
		"v000000000000000000000000000000000000000000000000000000000000000",
		"v0000000000000000000000000000000000000000000000000000000000000000",
	} {
		takes := updateArgs(cmd, []string{word}) == nil && ahoy.ReleaseTagShape.MatchString(word)
		if takes {
			taken++
		}
		d, err := reg.Check("abcd update " + word)
		if err != nil {
			t.Fatalf("%q: %v", word, err)
		}
		if refused := d.Verdict == guard.VerdictBlock && d.EntryID == "abcd-update-tag"; refused != takes {
			t.Errorf("%q: the command takes it as a tag %v, the guard refuses it %v (%q via %q)", word, takes, refused, d.Verdict, d.EntryID)
		}
	}
	if taken == 0 {
		t.Error("the command took no word as a tag, so nothing was compared")
	}
}

// TestTheGuardsTagPatternsAreTheirSources: the patterns the guard reads the update
// operand by are, letter for letter, the ones the command reads it by.
func TestTheGuardsTagPatternsAreTheirSources(t *testing.T) {
	for name, p := range map[string][2]*regexp.Regexp{
		"the release tag's alphabet": {guard.ReleaseTagWord, ahoy.ReleaseTagShape},
		"a release tag's start":      {guard.ReleaseTagStart, releaseTagStart},
		"a verb's shape":             {guard.VerbWord, verbShape},
	} {
		if p[0].String() != p[1].String() {
			t.Errorf("%s: the guard reads %s, its source %s", name, p[0], p[1])
		}
	}
}

// TestNoUpdateFlagTakesTheWordAfterIt: the guard reads the word after one of the
// update command's options as the operand, and its one sub-verb as no tag.
func TestNoUpdateFlagTakesTheWordAfterIt(t *testing.T) {
	update, _, err := NewRootCommand().Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, flags := range []*pflag.FlagSet{update.LocalFlags(), update.InheritedFlags()} {
		flags.VisitAll(func(f *pflag.Flag) {
			seen++
			if f.NoOptDefVal == "" {
				t.Errorf("--%s takes the word after it", f.Name)
			}
		})
	}
	if seen == 0 {
		t.Error("the update command has no flags to read")
	}
	for _, sub := range update.Commands() {
		if d, err := guard.Defaults().Check("abcd update " + sub.Name() + " v0.7.0"); err != nil || d.Verdict == guard.VerdictBlock {
			t.Errorf("the sub-verb %s is read as a tag: %q via %q, %v", sub.Name(), d.Verdict, d.EntryID, err)
		}
	}
}
