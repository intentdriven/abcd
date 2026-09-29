package credential

// walk.go is the walkthrough (itd-2609221017023290 criterion 2): the one
// sequence every setup of a credential runs, whichever adapter asks. It is a
// function the front doors call, in itd-63's explain-then-install shape:
// first what the credential unlocks and what works without it, then the three
// homes with the keychain recommended in the prose above them, then the
// adapter's own verification call with the value, and only when that call
// succeeds, the write through Set. The front door asks the person for the
// home (the CLI on its flags, the plugin page through the host's question
// tool); a value never passes through a question, because an answer is echoed
// into a transcript or an agent's context.

import (
	"context"
	"errors"
	"fmt"
)

// HomesProse is the prose above the choice of home: the keychain is
// recommended here, and never as a marked option.
const HomesProse = "Where the credential lives is your choice of three, made once. The platform keychain is " +
	"the home abcd recommends, because the secret stays in the operating system's own store rather than in a " +
	"file. A setup outside abcd keeps it with a tool you already use (an environment variable, or a field of " +
	"that tool's configuration file), and abcd stores only where to find it. The abcd-only home keeps it in " +
	"~/.abcd/credentials.json, readable by you alone. The value never enters the harness's settings or a repository."

// Service is one credential's walkthrough, supplied by the adapter that reads
// it.
type Service struct {
	// Name is the credential's name.
	Name string
	// Unlocks is what the credential unlocks.
	Unlocks string
	// WithoutIt is what works without it.
	WithoutIt string
	// Verify is the adapter's own call, made with the value before it is
	// stored. It must not carry the value into its error.
	Verify func(ctx context.Context, value string) error
}

// Explain is the walkthrough's explanation, in the order it is given.
func (s Service) Explain() []string {
	lines := []string{
		"credential " + s.Name + " unlocks " + s.Unlocks + ".",
		"Without it: " + s.WithoutIt + ".",
		HomesProse,
		"The homes: " + HomeExternal + " (a setup outside abcd), " + HomeABCD + " (abcd-only, on this machine), " +
			HomeKeychain + " (the platform keychain).",
	}
	return lines
}

// WalkResult is what a walkthrough did. It never carries the value.
type WalkResult struct {
	Name     string `json:"name"`
	Home     string `json:"home"`
	Verified bool   `json:"verified"`
	// Changed is false when the same credential was already stored there.
	Changed bool `json:"changed"`
	// Wrote names what the write touched, in the tilde form: the abcd home's
	// file, or the index and the keychain. Empty when nothing changed.
	Wrote []string `json:"wrote"`
}

// KeychainItem names the keychain's item for name as a surface shows it.
func KeychainItem(name string) string {
	return "the platform keychain (service " + keychainService + ", account " + name + ")"
}

// Walk runs the walkthrough for s with the chosen home: the value (or, for the
// external home, what the pointer resolves to) is verified with the adapter's
// own call, and only then stored. A name another home holds, or a different
// value in the same home, is refused before the call, so a setup that cannot
// store its credential is never verified for nothing.
func Walk(ctx context.Context, home string, s Service, c Choice) (WalkResult, error) {
	if s.Verify == nil {
		return WalkResult{}, fmt.Errorf("credential: the walkthrough for %s has no verification call, so nothing was written", s.Name)
	}
	if !nameRe.MatchString(s.Name) {
		return WalkResult{}, errors.New("credential: the name is not a plain credential name")
	}
	held, err := Where(home, s.Name)
	if err != nil {
		return WalkResult{}, err
	}
	if held != "" && held != c.Home {
		return WalkResult{}, fmt.Errorf("credential: %s is already held in the %s home, and abcd never replaces a stored secret; remove it there by hand to choose another home", s.Name, held)
	}
	value := c.Value
	if c.Home == HomeExternal {
		if value != "" {
			return WalkResult{}, errors.New("credential: the external home keeps a pointer, never a value, and a value was given")
		}
		if value, err = resolvePointer(home, s.Name, c.Pointer); err != nil {
			return WalkResult{}, err
		}
	} else if err := CheckValue(value); err != nil {
		return WalkResult{}, err
	}
	if held == HomeExternal && c.Home == HomeExternal {
		if err := samePointer(home, s.Name, c.Pointer); err != nil {
			return WalkResult{}, err
		}
	} else if held == c.Home {
		stored, err := Store(home).Resolve(s.Name)
		if err != nil {
			return WalkResult{}, err
		}
		if stored != value {
			return WalkResult{}, fmt.Errorf("credential: the %s home already holds a different value for %s, and abcd never replaces a stored secret; remove it there by hand to store a new one", held, s.Name)
		}
	}
	if err := s.Verify(ctx, value); err != nil {
		return WalkResult{}, fmt.Errorf("%w; the verification call failed, so nothing was written", err)
	}
	changed, err := Set(home, s.Name, c)
	if err != nil {
		return WalkResult{}, fmt.Errorf("%w; the credential verified, and nothing was written", err)
	}
	res := WalkResult{Name: s.Name, Home: c.Home, Verified: true, Changed: changed, Wrote: []string{}}
	if changed {
		switch c.Home {
		case HomeABCD:
			res.Wrote = []string{StorePath}
		case HomeKeychain:
			res.Wrote = []string{KeychainItem(s.Name), IndexPath}
		default:
			res.Wrote = []string{IndexPath}
		}
	}
	return res, nil
}
