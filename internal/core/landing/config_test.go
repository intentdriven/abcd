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

// A person takes the role back on their own machine, and no repository's file
// takes it from them: human wins from any layer, though the repository's file
// ordinarily beats the machine's.
func TestHumanInTheMachineLayerWinsOverAutoInTheRepo(t *testing.T) {
	f, err := readMode(configRoots(t, `{"facilitator":{"mode":"auto"}}`, `{"facilitator":{"mode":"human"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if f.Mode != ModeHuman || f.Layer != "machine" || !strings.Contains(f.Origin, "abcd.noindex/config.json") {
		t.Errorf("mode = %+v, want human from the machine file", f)
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

// A home RootsFor refused hides the machine layer, and with it any machine-wide
// human, so the facilitator reads as a person and says why.
func TestARefusedHomeMakesTheFacilitatorAPerson(t *testing.T) {
	r := configRoots(t, `{"facilitator":{"mode":"auto"}}`, "")
	r.Home, r.HomeRefused = "", true
	f, err := readMode(r)
	if err != nil {
		t.Fatal(err)
	}
	if f.Mode != ModeHuman || f.Layer != "machine" || !strings.Contains(f.Origin, "home") {
		t.Errorf("mode = %+v, want human from the refused machine layer, saying why", f)
	}
}
