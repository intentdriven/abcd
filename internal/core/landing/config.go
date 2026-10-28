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

// homeRefusedOrigin says why a refused home decides a key for its
// restrictive value: what names the machine-wide setting that cannot be read.
func homeRefusedOrigin(what string) string {
	return "the home directory was refused (HOME lies inside a git working tree that is not this account's), so a machine-wide " + what + " cannot be read; run abcd with your own HOME"
}

// layerValue is one layer's decoded value for a key, and where it came from.
type layerValue[T any] struct {
	v      T
	layer  string
	origin string
}

// decodeLayers decodes key from every layer of every stack that holds it,
// highest first within a stack, strictly, and through check when check is
// non-nil. Every layer is decoded, not only the one that decides, so a bad
// value is refused wherever it sits, naming its file and layer and what the
// key takes.
func decodeLayers[T any](stacks []stack, key string, check func(T) error, takes string) ([]layerValue[T], error) {
	var out []layerValue[T]
	for _, st := range stacks {
		found, err := st.s.Lookup(key)
		if err != nil {
			return nil, err
		}
		for _, f := range found {
			origin := f.Origin
			if st.where != "" {
				origin += " in the branch's checkout " + st.where
			}
			v, err := layered.Decode[T](f.Raw)
			if err == nil && check != nil {
				err = check(v)
			}
			if err != nil {
				return nil, fmt.Errorf("%s (%s layer): %s: %v; it takes %s", origin, f.Layer, key, err, takes)
			}
			out = append(out, layerValue[T]{v: v, layer: f.Layer.String(), origin: origin})
		}
	}
	return out, nil
}

// lookupSwitch ANDs key's value across every layer of every stack that holds
// it, and a refused home's hidden machine layer. Lookup lists the layers
// highest first, so the first off it meets is the one reported.
func lookupSwitch(stacks []stack, key string, homeRefused bool) (Switch, error) {
	vals, err := decodeLayers[bool](stacks, key, nil, "true or false")
	if err != nil {
		return Switch{}, err
	}
	for _, v := range vals {
		if !v.v {
			return Switch{Key: key, On: false, Layer: v.layer, Origin: v.origin}, nil
		}
	}
	if homeRefused {
		return Switch{Key: key, On: false, Layer: layered.Machine.String(), Origin: homeRefusedOrigin("off")}, nil
	}
	return Switch{Key: key, On: true, Layer: layered.Bundled.String(), Origin: layered.Bundled.String()}, nil
}

// offText says which layer turned a switch off, as every step names it.
func (s Switch) offText() string {
	return fmt.Sprintf("%s is off (%s layer, %s)", s.Key, s.Layer, s.Origin)
}

// KeyFacilitatorMode declares who merges: the repository's technical
// facilitator is a person (ModeHuman, the bundled default) or abcd (ModeAuto).
// It says nothing about abcd mode's facilitator state, which is whose answer
// the loop is waiting on right now.
const (
	KeyFacilitatorMode = "facilitator.mode"
	ModeHuman          = "human"
	ModeAuto           = "auto"
)

// Facilitator is facilitator.mode as resolved: the mode, and the layer and
// origin that decided it. Layer is "bundled" (Origin "bundled") when no layer
// declares the key; otherwise Origin names the file that set it
// (".abcd/config.json", or the machine file in its tilde form), or, for a
// refused home, why the machine layer could not be read.
type Facilitator struct {
	Key    string `json:"key"`
	Mode   string `json:"mode"`
	Layer  string `json:"layer"`
	Origin string `json:"origin"`
	// HomeRefused says a refused home decided the mode (human), so Origin is
	// the reason, not a file.
	HomeRefused bool `json:"home_refused,omitempty"`
}

// checkMode accepts the two modes, exactly as spelt.
func checkMode(m string) error {
	if m != ModeHuman && m != ModeAuto {
		return fmt.Errorf("%q is not a facilitator mode", layered.BoundKey(m))
	}
	return nil
}

// ReadFacilitator resolves facilitator.mode for the working directory cwd,
// as the merge gate reads it, with the notes RootsFor wrote on the way (a
// refused home among them) for the caller to print. The error is the reader's
// raw text: a front door masks it before printing it, since it can quote a key
// a configuration file wrote.
func ReadFacilitator(cwd string) (Facilitator, []string, error) {
	r, notes := layered.RootsFor(cwd)
	f, err := readMode(r)
	return f, notes, err
}

// readMode resolves facilitator.mode through the layered configuration
// reader, which claims the facilitator namespace with its one key mode, so a
// misspelt key is refused naming its file.
//
// The precedence is land's switches' (readSwitches), not the reader's
// ordinary one: human from any layer wins, so a person can take the role back
// for every repository on their machine and no repository's file takes it from
// them. Every layer that holds the key is decoded and checked, so a value other
// than human or auto is refused in whichever layer it sits. With no human
// anywhere, the highest layer that says auto decides; with no layer at all,
// the bundled default human does. A home RootsFor refused hides the machine
// layer, and with it any machine-wide human, so the mode reads as human.
func readMode(r layered.Roots) (Facilitator, error) {
	s, err := layered.Load(layered.Config, r)
	if err != nil {
		return Facilitator{}, err
	}
	if err := s.Claim("facilitator", "mode"); err != nil {
		return Facilitator{}, err
	}
	vals, err := decodeLayers([]stack{{s: s}}, KeyFacilitatorMode, checkMode, "human or auto")
	if err != nil {
		return Facilitator{}, err
	}
	for _, v := range vals {
		if v.v == ModeHuman {
			return Facilitator{Key: KeyFacilitatorMode, Mode: ModeHuman, Layer: v.layer, Origin: v.origin}, nil
		}
	}
	if r.HomeRefused {
		return Facilitator{Key: KeyFacilitatorMode, Mode: ModeHuman, Layer: layered.Machine.String(), Origin: homeRefusedOrigin(ModeHuman), HomeRefused: true}, nil
	}
	if len(vals) > 0 {
		return Facilitator{Key: KeyFacilitatorMode, Mode: ModeAuto, Layer: vals[0].layer, Origin: vals[0].origin}, nil
	}
	return Facilitator{Key: KeyFacilitatorMode, Mode: ModeHuman, Layer: layered.Bundled.String(), Origin: layered.Bundled.String()}, nil
}
