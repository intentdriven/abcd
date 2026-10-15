package landing

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
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
//
// branchRepo, when not "", is the landed branch's own checkout, whose
// repository layer is read beside r's and ANDed too: an off committed on the
// branch counts wherever land is run from. A home RootsFor refused hides the
// machine layer, and with it any machine-wide off, so both acts read as off.
func readSwitches(r layered.Roots, branchRepo string, noPush, noEnqueue bool) (push, enqueue Switch, err error) {
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
	stacks := []stack{{s: s}}
	if branchRepo != "" {
		// The branch's checkout is read for its repository layer alone: the
		// machine layer is r's and is already read.
		b, err := layered.Load(layered.Config, layered.Roots{Repo: branchRepo, HomeRefused: true})
		if err != nil {
			return Switch{}, Switch{}, err
		}
		stacks = append(stacks, stack{s: b, where: fsutil.RedactHome(branchRepo)})
	}
	for _, st := range stacks {
		if err := st.s.Claim("land", "push", "enqueue"); err != nil {
			return Switch{}, Switch{}, err
		}
	}
	if push, err = lookupSwitch(stacks, KeyPush, r.HomeRefused); err != nil {
		return Switch{}, Switch{}, err
	}
	if enqueue, err = lookupSwitch(stacks, KeyEnqueue, r.HomeRefused); err != nil {
		return Switch{}, Switch{}, err
	}
	return push, enqueue, nil
}

// stack is one loaded configuration, and the checkout it was read in when it
// is the landed branch's ("" for the working directory's own).
type stack struct {
	s     *layered.Stack
	where string
}

// homeRefusedOrigin says why a refused home turns both acts off.
const homeRefusedOrigin = "the home directory was refused (HOME lies inside a git working tree that is not this account's), so a machine-wide off cannot be read; run abcd with your own HOME"

// lookupSwitch ANDs key's value across every layer of every stack that holds
// it, and a refused home's hidden machine layer. Lookup lists the layers
// highest first, so the first off it meets is the one reported.
func lookupSwitch(stacks []stack, key string, homeRefused bool) (Switch, error) {
	out := Switch{Key: key, On: true, Layer: layered.Bundled.String(), Origin: layered.Bundled.String()}
	for _, st := range stacks {
		found, err := st.s.Lookup(key)
		if err != nil {
			return Switch{}, err
		}
		for _, f := range found {
			origin := f.Origin
			if st.where != "" {
				origin += " in the branch's checkout " + st.where
			}
			on, err := layered.Decode[bool](f.Raw)
			if err != nil {
				return Switch{}, fmt.Errorf("%s (%s layer): %s: %v; it takes true or false", origin, f.Layer, key, err)
			}
			if !on && out.On {
				out = Switch{Key: key, On: false, Layer: f.Layer.String(), Origin: origin}
			}
		}
	}
	if homeRefused && out.On {
		out = Switch{Key: key, On: false, Layer: layered.Machine.String(), Origin: homeRefusedOrigin}
	}
	return out, nil
}

// offText says which layer turned a switch off, as every step names it.
func (s Switch) offText() string {
	return fmt.Sprintf("%s is off (%s layer, %s)", s.Key, s.Layer, s.Origin)
}
