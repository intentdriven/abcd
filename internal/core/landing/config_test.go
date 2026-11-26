package landing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// configRoots lays out a repository layer and a machine layer of
// .abcd/config.json holding repo and machine ("" leaves a layer absent).
func configRoots(t *testing.T, repo, machine string) layered.Roots {
	t.Helper()
	r := layered.Roots{Repo: t.TempDir(), Home: t.TempDir()}
	// The machine layer is read for the mode only where it is the account's
	// own home (personmode.go), which the test's HOME stands for.
	t.Setenv("HOME", r.Home)
	write := func(p, body string) {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if repo != "" {
		write(filepath.Join(r.Repo, ".abcd", "config.json"), repo)
	}
	if machine != "" {
		write(abcdhome.Path(r.Home, "config.json"), machine)
	}
	return r
}

func TestBothSwitchesAreOnByDefault(t *testing.T) {
	push, enqueue, err := readSwitches(configRoots(t, "", ""), "", false, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []Switch{push, enqueue} {
		if !s.On || s.Layer != "bundled" {
			t.Errorf("%s = %+v, want on from the bundled default", s.Key, s)
		}
	}
	if push.Key != KeyPush || enqueue.Key != KeyEnqueue {
		t.Errorf("keys %q, %q", push.Key, enqueue.Key)
	}
}

// The machine layer turns an outward act off even where the repository's
// file, which ordinarily wins, says on: off wins from any layer.
func TestOffInTheMachineLayerWinsOverOnInTheRepo(t *testing.T) {
	r := configRoots(t, `{"land":{"push":true,"enqueue":true}}`, `{"land":{"push":false}}`)
	push, enqueue, err := readSwitches(r, "", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if push.On || push.Layer != "machine" || !strings.Contains(push.Origin, "config.json") {
		t.Errorf("push = %+v, want off from the machine layer", push)
	}
	if !enqueue.On {
		t.Errorf("enqueue = %+v, want on", enqueue)
	}
}

func TestOffInTheRepoWinsOverOnInTheMachineLayer(t *testing.T) {
	r := configRoots(t, `{"land":{"enqueue":false}}`, `{"land":{"enqueue":true,"push":true}}`)
	push, enqueue, err := readSwitches(r, "", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if enqueue.On || enqueue.Layer != "repo" || enqueue.Origin != ".abcd/config.json" {
		t.Errorf("enqueue = %+v, want off from the repo layer", enqueue)
	}
	if !push.On {
		t.Errorf("push = %+v, want on", push)
	}
}

// A flag turns a key off for one invocation; no flag turns one on against a
// layer's off.
func TestTheNoFlagsTurnTheKeysOffForOneRun(t *testing.T) {
	push, enqueue, err := readSwitches(configRoots(t, "", ""), "", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if push.On || push.Layer != "flag" || push.Origin != "--no-push" {
		t.Errorf("push = %+v, want off from --no-push", push)
	}
	if enqueue.On || enqueue.Layer != "flag" || enqueue.Origin != "--no-enqueue" {
		t.Errorf("enqueue = %+v, want off from --no-enqueue", enqueue)
	}
}

func TestAnUnknownKeyUnderLandIsRefused(t *testing.T) {
	_, _, err := readSwitches(configRoots(t, `{"land":{"psuh":false}}`, ""), "", false, false)
	if err == nil || !strings.Contains(err.Error(), "land.psuh") || !strings.Contains(err.Error(), ".abcd/config.json") {
		t.Fatalf("err = %v, want the misspelt key refused naming its file", err)
	}
}

func TestANonBooleanValueIsRefusedNamingItsLayer(t *testing.T) {
	_, _, err := readSwitches(configRoots(t, "", `{"land":{"push":"no"}}`), "", false, false)
	if err == nil || !strings.Contains(err.Error(), "land.push") || !strings.Contains(err.Error(), "machine") {
		t.Fatalf("err = %v, want the value refused naming the machine layer", err)
	}
}

// m4: a home RootsFor refused hides the machine layer, and with it any
// machine-wide off, so both acts read as off and say why.
func TestARefusedHomeTurnsBothActsOff(t *testing.T) {
	r := configRoots(t, `{"land":{"push":true}}`, "")
	r.Home, r.HomeRefused = "", true
	push, enqueue, err := readSwitches(r, "", false, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []Switch{push, enqueue} {
		if s.On || s.Layer != "machine" || !strings.Contains(s.Origin, "home") {
			t.Errorf("%s = %+v, want off from the refused machine layer, saying why", s.Key, s)
		}
	}
}

// With no layer declaring facilitator.mode, the facilitator is a person: a
// fresh install changes nothing about who merges.
func TestTheFacilitatorIsAPersonByDefault(t *testing.T) {
	f, err := readMode(configRoots(t, "", ""))
	if err != nil {
		t.Fatal(err)
	}
	if f.Key != KeyFacilitatorMode || f.Mode != ModeHuman || f.Layer != "bundled" || f.Origin != "bundled" {
		t.Errorf("mode = %+v, want human from the bundled default", f)
	}
}

func TestAutoInTheRepoMakesAbcdTheFacilitator(t *testing.T) {
	f, err := readMode(configRoots(t, `{"facilitator":{"mode":"auto"}}`, ""))
	if err != nil {
		t.Fatal(err)
	}
	if f.Mode != ModeAuto || f.Layer != "repo" || f.Origin != ".abcd/config.json" {
		t.Errorf("mode = %+v, want auto from .abcd/config.json", f)
	}
}

func TestAutoInTheMachineLayerAloneMakesAbcdTheFacilitator(t *testing.T) {
	f, err := readMode(configRoots(t, "", `{"facilitator":{"mode":"auto"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if f.Mode != ModeAuto || f.Layer != "machine" || !strings.Contains(f.Origin, "abcd.noindex/config.json") {
		t.Errorf("mode = %+v, want auto from the machine file", f)
	}
}

// The project's committed value is above the computer's (dcn-2611252132144530):
// a project that commits auto is auto under the computer's human, and says it
// enforces auto over it.
func TestTheProjectsAutoStandsAboveTheComputersHuman(t *testing.T) {
	f, err := readMode(configRoots(t, `{"facilitator":{"mode":"auto"}}`, `{"facilitator":{"mode":"human"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if f.Mode != ModeAuto || f.Layer != "repo" || f.Origin != ".abcd/config.json" || f.Over != ModeHuman {
		t.Errorf("mode = %+v, want auto from .abcd/config.json over the computer's human", f)
	}
	if got, want := f.Says(), "the project enforces auto over this computer's human"; got != want {
		t.Errorf("Says() = %q, want %q", got, want)
	}
}

func TestHumanInTheRepoWinsOverAutoInTheMachineLayer(t *testing.T) {
	f, err := readMode(configRoots(t, `{"facilitator":{"mode":"human"}}`, `{"facilitator":{"mode":"auto"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if f.Mode != ModeHuman || f.Layer != "repo" || f.Origin != ".abcd/config.json" {
		t.Errorf("mode = %+v, want human from .abcd/config.json", f)
	}
}

// Any value but human or auto is refused, naming the file, in every layer: a
// human above it does not excuse a bad value below.
func TestAModeOtherThanHumanOrAutoIsRefusedNamingItsFile(t *testing.T) {
	for _, c := range []struct {
		name, repo, machine, file, layer string
	}{
		{"unknown word in the repo", `{"facilitator":{"mode":"robot"}}`, "", ".abcd/config.json", "repo"},
		{"wrong case in the machine file", "", `{"facilitator":{"mode":"Auto"}}`, "abcd.noindex/config.json", "machine"},
		{"a boolean in the repo", `{"facilitator":{"mode":true}}`, "", ".abcd/config.json", "repo"},
		{"null in the machine file", "", `{"facilitator":{"mode":null}}`, "abcd.noindex/config.json", "machine"},
		{"empty in the repo", `{"facilitator":{"mode":""}}`, "", ".abcd/config.json", "repo"},
		{"bad below a winning human", `{"facilitator":{"mode":"human"}}`, `{"facilitator":{"mode":"robot"}}`, "abcd.noindex/config.json", "machine"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, err := readMode(configRoots(t, c.repo, c.machine))
			if err == nil {
				t.Fatalf("mode = %+v, want the value refused", f)
			}
			for _, want := range []string{KeyFacilitatorMode, c.file, c.layer + " layer", "human or auto"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("err = %v, want it to name %q", err, want)
				}
			}
		})
	}
}

// facilitator is a claimed namespace with the one key mode, so a misspelt key
// is refused rather than leaving the default to apply unannounced.
func TestAnUnknownKeyUnderFacilitatorIsRefused(t *testing.T) {
	for _, c := range []struct {
		name, repo, machine, file string
	}{
		{"misspelt in the repo", `{"facilitator":{"moed":"auto"}}`, "", ".abcd/config.json"},
		{"extra key in the machine file", "", `{"facilitator":{"mode":"human","merges":"me"}}`, "abcd.noindex/config.json"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, err := readMode(configRoots(t, c.repo, c.machine))
			if err == nil || !strings.Contains(err.Error(), "facilitator.") || !strings.Contains(err.Error(), c.file) {
				t.Fatalf("mode = %+v, err = %v, want the unknown key refused naming %s", f, err, c.file)
			}
		})
	}
}

func TestAFacilitatorThatIsNotAnObjectIsRefused(t *testing.T) {
	f, err := readMode(configRoots(t, `{"facilitator":"auto"}`, ""))
	if err == nil || !strings.Contains(err.Error(), "facilitator") || !strings.Contains(err.Error(), ".abcd/config.json") {
		t.Fatalf("mode = %+v, err = %v, want the non-object refused naming its file", f, err)
	}
}

// A refused home no longer decides where the project has a value: a project
// that commits auto is auto under it (criterion 3's first exception). Where
// the project has none, the hidden computer's file reads as human and says why.
func TestARefusedHomeDecidesOnlyWhereTheProjectHasNoValue(t *testing.T) {
	r := configRoots(t, `{"facilitator":{"mode":"auto"}}`, "")
	r.Home, r.HomeRefused = "", true
	f, err := readMode(r)
	if err != nil {
		t.Fatal(err)
	}
	if f.Mode != ModeAuto || f.Layer != "repo" || f.HomeRefused {
		t.Errorf("mode = %+v, want auto from the repository file", f)
	}
	r = configRoots(t, "", "")
	r.Home, r.HomeRefused = "", true
	if f, err = readMode(r); err != nil {
		t.Fatal(err)
	}
	if f.Mode != ModeHuman || f.Layer != "machine" || !f.HomeRefused || !strings.Contains(f.Origin, "home") {
		t.Errorf("mode = %+v, want human from the refused machine layer, saying why", f)
	}
}

// TestTheProjectsValueDecidesFirst is the resolver's whole table without a
// person's entry (itd-2611252049031692 criteria 1 and 3): the project's
// committed value decides, then the computer's file, then the bundled human;
// a refused home reads human only where the project has no value. Each cell
// names the mode, the layer that decided it, the computer's value the
// project's stands above where they differ, and whose it is in the words
// enforced, prevented or left to each contributor.
func TestTheProjectsValueDecidesFirst(t *testing.T) {
	const auto, human = `{"facilitator":{"mode":"auto"}}`, `{"facilitator":{"mode":"human"}}`
	const refused = "refused home"
	for _, c := range []struct {
		project, computer       string
		mode, layer, over, says string
	}{
		{auto, auto, ModeAuto, "repo", "", "the project enforces auto"},
		{auto, human, ModeAuto, "repo", ModeHuman, "the project enforces auto over this computer's human"},
		{auto, "", ModeAuto, "repo", "", "the project enforces auto"},
		{auto, refused, ModeAuto, "repo", "", "the project enforces auto"},
		{human, auto, ModeHuman, "repo", ModeAuto, "the project prevents auto over this computer's auto"},
		{human, human, ModeHuman, "repo", "", "the project prevents auto"},
		{human, "", ModeHuman, "repo", "", "the project prevents auto"},
		{human, refused, ModeHuman, "repo", "", "the project prevents auto"},
		{"", auto, ModeAuto, "machine", "", "left to each contributor, and this computer's file sets auto"},
		{"", human, ModeHuman, "machine", "", "left to each contributor, and this computer's file sets human"},
		{"", "", ModeHuman, "bundled", "", "left to each contributor, and set nowhere"},
		{"", refused, ModeHuman, "machine", "", "left to each contributor, and this computer's file could not be read"},
	} {
		name := "project " + c.project + ", computer " + c.computer
		t.Run(name, func(t *testing.T) {
			machine := c.computer
			if machine == refused {
				machine = ""
			}
			r := configRoots(t, c.project, machine)
			if c.computer == refused {
				r.Home, r.HomeRefused = "", true
			}
			f, err := readMode(r)
			if err != nil {
				t.Fatal(err)
			}
			if f.Mode != c.mode || f.Layer != c.layer || f.Over != c.over {
				t.Errorf("mode = %+v, want %s from the %s layer over %q", f, c.mode, c.layer, c.over)
			}
			if got := f.Says(); got != c.says {
				t.Errorf("Says() = %q, want %q", got, c.says)
			}
		})
	}
}

// A repository's human under a refused home is still the repository's: the
// human answer is reported from the layer that set it, not the refused one.
func TestHumanInTheRepoUnderARefusedHomeIsReportedFromTheRepo(t *testing.T) {
	r := configRoots(t, `{"facilitator":{"mode":"human"}}`, "")
	r.Home, r.HomeRefused = "", true
	f, err := readMode(r)
	if err != nil {
		t.Fatal(err)
	}
	if f.Mode != ModeHuman || f.Layer != "repo" || !strings.Contains(f.Origin, ".abcd/config.json") {
		t.Errorf("mode = %+v, want human from the repository file", f)
	}
}

// The records write the setting as facilitator.mode, so a file may hold it as
// one dotted key. The reader takes only the nested spelling, and silently
// ignoring the dotted one would let a value its file meant to set go unread,
// so it is refused in whichever layer it sits, naming the file and the nested
// spelling a reader takes.
func TestADottedFacilitatorModeKeyIsRefusedNamingItsFile(t *testing.T) {
	for _, c := range []struct {
		name, repo, machine, file, layer, key string
	}{
		{"dotted in the machine file over auto in the repo", `{"facilitator":{"mode":"auto"}}`, `{"facilitator.mode":"human"}`, "abcd.noindex/config.json", "machine", "facilitator.mode"},
		{"dotted in the repo", `{"facilitator.mode":"auto"}`, "", ".abcd/config.json", "repo", "facilitator.mode"},
		{"dotted and miscased beside the nested spelling", `{"facilitator":{"mode":"human"},"Facilitator.Mode":"auto"}`, "", ".abcd/config.json", "repo", "Facilitator.Mode"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, err := readMode(configRoots(t, c.repo, c.machine))
			if err == nil {
				t.Fatalf("mode = %+v, want the dotted key refused", f)
			}
			for _, want := range []string{`"` + c.key + `"`, `"facilitator"`, c.file, c.layer + " layer"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("err = %v, want it to name %s", err, want)
				}
			}
		})
	}
}

// A namespace spelt in another case is not the facilitator namespace, and
// silently ignoring it would let a value its file meant to set go unread, so
// it is refused in whichever layer it sits, naming the file and the spelling a
// reader takes.
func TestAMiscasedFacilitatorNamespaceIsRefusedNamingItsFile(t *testing.T) {
	for _, c := range []struct {
		name, repo, machine, file, layer, key string
	}{
		{"capitalised in the machine file over auto in the repo", `{"facilitator":{"mode":"auto"}}`, `{"Facilitator":{"mode":"human"}}`, "abcd.noindex/config.json", "machine", "Facilitator"},
		{"upper case in the repo", `{"FACILITATOR":{"mode":"auto"}}`, "", ".abcd/config.json", "repo", "FACILITATOR"},
		{"mixed case below a winning human", `{"facilitator":{"mode":"human"}}`, `{"faciLitator":{"mode":"auto"}}`, "abcd.noindex/config.json", "machine", "faciLitator"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, err := readMode(configRoots(t, c.repo, c.machine))
			if err == nil {
				t.Fatalf("mode = %+v, want the miscased namespace refused", f)
			}
			for _, want := range []string{`"` + c.key + `"`, `"facilitator"`, c.file, c.layer + " layer"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("err = %v, want it to name %s", err, want)
				}
			}
		})
	}
}
