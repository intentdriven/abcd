// Package dryrun is the one vocabulary a practice run reports in
// (itd-2610301428390406, spc-2610301431131636): what a writing command that
// reads a filled-in input would write, and what it would do that is not a
// file, had the input been submitted for real.
//
// A writing command's core splits its check from its write: a Plan stage
// does everything up to the first write (decode, validate, resolve, redact,
// compose) and returns the writes and effects below, and the real call is
// that Plan followed by applying it. So a practice run and the real write
// share every check by construction. This package holds only the shapes the
// plans are reported in; it is a leaf, so every core package that plans can
// import it without importing another.
package dryrun

// Action is what a planned write does to its path.
type Action string

const (
	// Create writes a path that does not exist yet.
	Create Action = "create"
	// Replace rewrites a path that exists.
	Replace Action = "replace"
	// Move moves a path to another; From names where it moves from.
	Move Action = "move"
)

// Write is one file a real run would write. Path is repo-relative, or
// ~/-relative for a file in abcd's home, so no absolute home path is echoed.
// Bytes is the size the file would have. Pending, when set, says what the
// write itself settles that a plan cannot: a record id minted at the write,
// and the size of a record another writer composes.
type Write struct {
	Path    string `json:"path"`
	Action  Action `json:"action"`
	Bytes   int    `json:"bytes"`
	From    string `json:"from,omitempty"`
	Pending string `json:"pending,omitempty"`
	// Content is the bytes the write would lay, where the plan composed
	// them. It is the plan's own, for the apply stage and for a front door
	// that shows a single record's opening; it is never marshalled.
	Content []byte `json:"-"`
}

// Effect is one thing a real run would do that is not a file write: a tag,
// a push, a release, a record another store files or resolves.
type Effect struct {
	Effect string `json:"effect"`
	Target string `json:"target"`
}

// Plan is what a practice run reports: every planned write, then every
// planned effect. Both are empty for a run that would write nothing (an
// input already recorded).
type Plan struct {
	Writes  []Write  `json:"would_write"`
	Effects []Effect `json:"would_do"`
}

// MintedID is the placeholder a planned path carries where the real write
// mints a record id: the id is drawn from the clock and entropy at the
// moment of the write, so no plan can name it in advance.
const MintedID = "<id>"

// Normalise returns p with nil lists made empty, so a plan marshals as two
// lists and never as null.
func (p Plan) Normalise() Plan {
	if p.Writes == nil {
		p.Writes = []Write{}
	}
	if p.Effects == nil {
		p.Effects = []Effect{}
	}
	return p
}
