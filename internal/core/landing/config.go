package landing

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/core/layered"
)

// The landing's configuration keys, in .abcd/config.json and
// ~/.abcd.noindex/config.json, each a bool whose bundled default is true.
const (
	KeyPush    = "land.push"
	KeyEnqueue = "land.enqueue"
)

// Switch is one of the landing's two outward acts as configured: whether it is
// on, and the layer and origin that decided it — the bundled default when no
// layer turned it off, else the highest layer that did.
type Switch struct {
	Key    string `json:"key"`
	On     bool   `json:"on"`
	Layer  string `json:"layer"`
	Origin string `json:"origin"`
}

// readSwitches resolves land.push and land.enqueue through the layered
// configuration reader, which claims the land namespace so a misspelt key is
// refused naming its file.
//
// The precedence is deliberately not the reader's ordinary one. Ordinarily the
// repository's file beats the machine's; here OFF wins from any layer, so a
// person can switch an outward act off for every repository on the machine and
// no repository's file switches it back on (the rules loader's kill switch is
// the precedent). Every layer that holds a key is decoded, strictly, and the
// values are ANDed. noPush and noEnqueue are the --no-push and --no-enqueue
// flags, which add a flag-layer false; there is no flag that turns a key on
// against a layer's off.
func readSwitches(r layered.Roots, noPush, noEnqueue bool) (push, enqueue Switch, err error) {
	s, err := layered.Load(layered.Config, r)
	if err != nil {
		return Switch{}, Switch{}, err
	}
	if noPush {
		if err := s.SetFlag(KeyPush, false, "--no-push"); err != nil {
			return Switch{}, Switch{}, err
		}
	}
	if noEnqueue {
		if err := s.SetFlag(KeyEnqueue, false, "--no-enqueue"); err != nil {
			return Switch{}, Switch{}, err
		}
	}
	if err := s.Claim("land", "push", "enqueue"); err != nil {
		return Switch{}, Switch{}, err
	}
	if push, err = lookupSwitch(s, KeyPush); err != nil {
		return Switch{}, Switch{}, err
	}
	if enqueue, err = lookupSwitch(s, KeyEnqueue); err != nil {
		return Switch{}, Switch{}, err
	}
	return push, enqueue, nil
}

// lookupSwitch ANDs key's value across every layer that holds it. Lookup lists
// the layers highest first, so the first off it meets is the one reported.
func lookupSwitch(s *layered.Stack, key string) (Switch, error) {
	found, err := s.Lookup(key)
	if err != nil {
		return Switch{}, err
	}
	out := Switch{Key: key, On: true, Layer: layered.Bundled.String(), Origin: layered.Bundled.String()}
	for _, f := range found {
		on, err := layered.Decode[bool](f.Raw)
		if err != nil {
			return Switch{}, fmt.Errorf("%s (%s layer): %s: %v; it takes true or false", f.Origin, f.Layer, key, err)
		}
		if !on && out.On {
			out = Switch{Key: key, On: false, Layer: f.Layer.String(), Origin: f.Origin}
		}
	}
	return out, nil
}

// offText says which layer turned a switch off, as every step names it.
func (s Switch) offText() string {
	return fmt.Sprintf("%s is off (%s layer, %s)", s.Key, s.Layer, s.Origin)
}
