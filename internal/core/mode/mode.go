// Package mode is abcd's waiting-on state: which of three states the agent loop
// is parked in. Managed and nobody waiting; parked on the facilitator; parked on
// the product thinker. The agent writes it when it stops for a verdict, naming
// whom it is addressing, and the human writes it by hand to say which hat they
// wear; the status render and the bare board read the same state, so no surface
// invents its own answer (spc-70, itd-200).
//
// The state lives in the repository's local-ephemeral tier, at
// `.abcd/.work.local/mode`: one line, one of three words. Three properties are
// load-bearing.
//
// It is PER-REPOSITORY, because the question it answers is about one checkout's
// loop, so the root is resolved through gitutil.CheckoutRoot — git's toplevel or
// one of two refusals, never a guess. A repository-shaped tree git cannot answer
// for and a directory that is no repository at all are both refusals here; the
// alternative, falling back to the working directory, is the defect this project
// has already fixed across six front doors (iss-2609090951291524,
// iss-2609091707224329).
//
// It is MANAGED-ONLY BY CONSTRUCTION, not by a check. Only a managed repository
// has the local-ephemeral tier, so the writer requires the tier and creates
// nothing: a repository abcd does not manage has nowhere for the state to go and
// the write refuses, without this package ever asking "is this managed?" and
// without minting an abcd namespace in a tree that never asked for one. A bare
// `.abcd/` is deliberately not accepted in its place — it is not a managed
// signal on its own (iss-88), so gating on it would be a second, weaker answer
// to a question the tier already answers.
//
// ABSENT MEANS MANAGED, which is what lets the reader hold the same property
// with no check at all: an unmanaged repository has no tier, so it has no file,
// so it reads as managed — the same answer a managed repository with no parked
// stop gives, which is the only answer either could sensibly carry.
//
// This package is a library: it returns values and errors, and never prints.
package mode

import (
	"errors"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// State is the waiting-on state. The vocabulary is CLOSED: these three words are
// the whole of it, on the way in and on the way out, because a badge that can
// render a fourth thing is a badge whose meaning is set by whoever last wrote the
// file.
type State string

const (
	// Managed is abcd present and nobody waiting. It is what an absent store
	// reads as.
	Managed State = "managed"
	// Facilitator is the loop parked on the facilitator — the person at the
	// terminal running the agents.
	Facilitator State = "facilitator"
	// ProductThinker is the loop parked on the product thinker, who answers on a
	// surface of their own and is exactly the addressee a terminal-only stop
	// leaves unnotified.
	ProductThinker State = "product-thinker"
)

// StoreName is the noun this store's root refusals are phrased with. It is the
// store's half of gitutil.CheckoutRoot's contract: the resolution and the refusal
// policy are shared there, and only the noun is this store's.
const StoreName = "the mode store"

// TierRelPath is the local-ephemeral tier that holds the store, repo-relative and
// slash-separated (an os.Root path). Its presence is the managed-only property:
// the writer requires it and never creates it.
const TierRelPath = ".abcd/.work.local"

// FileRelPath is the store itself, repo-relative and slash-separated. One line,
// one word, gitignored with the rest of the tier and per worktree — so two
// concurrent sessions on one repository each park their own loop and neither
// merge-conflicts nor overwrites the other's state.
const FileRelPath = TierRelPath + "/mode"

// maxStoreBytes caps the guarded read. The longest legal content is sixteen
// bytes; the cap is generous enough that a hand-mangled file is refused as
// malformed (which names the three words) rather than as oversize, and small
// enough that a device or a runaway file planted at the path cannot be read into
// memory.
const maxStoreBytes = 4096

// echoCap bounds how much of a rejected value an error quotes back. The value is
// untrusted content — a hand-edited file, or an argument from a front door — so it
// is cleaned through termsafe before it reaches a human's terminal.
const echoCap = 64

var (
	// ErrUnknownState is the closed vocabulary's refusal, raised both by a write
	// offered a word that is not one of the three and by a read that finds one in
	// the store. The message names all three, because a refusal that does not say
	// what would have been accepted makes the caller guess.
	ErrUnknownState = errors.New("mode: unknown state")

	// ErrNoLocalTier is the writer's refusal when the local-ephemeral tier is
	// absent. It is the managed-only property doing its work: there is nowhere in
	// this tree for the state to live, and creating the tier would mint an abcd
	// namespace in a repository abcd does not manage.
	ErrNoLocalTier = errors.New("mode: no local-ephemeral tier")
)

// States returns the three legal states in the order they are documented. It is
// the one enumeration; a surface that offers a choice or a completion reads it
// from here rather than restating the list.
func States() []State {
	return []State{Managed, Facilitator, ProductThinker}
}

// Valid reports whether s is one of the three.
func (s State) Valid() bool {
	switch s {
	case Managed, Facilitator, ProductThinker:
		return true
	}
	return false
}

// String renders the state as its stored word.
func (s State) String() string { return string(s) }

// Addressee names, in plain words, the person whose answer a parked loop is
// waiting on: "facilitator", "product thinker", or "" for Managed, which owes
// nobody an answer. It is the one place the vocabulary is turned into prose a
// front door can put in a sentence, so the notice a set prints where the host
// has no status surface (ac-7) and the words the badge carries cannot drift
// into naming two different people.
func (s State) Addressee() string {
	switch s {
	case Facilitator:
		return "facilitator"
	case ProductThinker:
		return "product thinker"
	}
	return ""
}

// ParseState reads one of the three words from raw, tolerating the surrounding
// whitespace a stored line or a shell argument carries — the trailing newline the
// writer itself emits is the common case. Anything else is refused, and the
// refusal names the three.
//
// Matching is exact beyond that trim: `Facilitator` and `PRODUCT-THINKER` are
// refused rather than folded, because the store holds the word the writer wrote
// and a reader that accepts variants makes the file's contents ambiguous for the
// next writer.
func ParseState(raw string) (State, error) {
	s := State(strings.TrimSpace(raw))
	if s.Valid() {
		return s, nil
	}
	return "", fmt.Errorf("%w: %q is not one of %s",
		ErrUnknownState, termsafe.CleanProseLine(raw, echoCap), stateList())
}

// stateList renders the vocabulary for a refusal message.
func stateList() string {
	words := make([]string, 0, len(States()))
	for _, s := range States() {
		words = append(words, string(s))
	}
	return strings.Join(words, ", ")
}
