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
