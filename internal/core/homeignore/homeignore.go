// Package homeignore keeps abcd's home out of a git repository in the
// person's home folder (itd-2610140858048004, adr-2610140859394097).
//
// abcd keeps machine-scoped records in ~/.abcd.noindex: redacted transcripts,
// private ideate verdicts, run state, labs, backups. A person whose home is a
// dotfiles repository would stage every one of them with an ordinary
// `git add -A`. Lay puts a .gitignore holding `*` inside abcd's own folder,
// which git applies to every path beneath it, so the folder keeps itself out
// without abcd reading, running or changing anything of the person's
// repository.
//
// It is laid ONCE. A marker beside it records the lay, and while the marker
// stands Lay does nothing, so a person who deletes the .gitignore to let their
// home repository track abcd's home has made a deliberate choice abcd never
// undoes. A .gitignore the person already keeps there is never overwritten.
//
// The credential store does not rest on this file: it refuses the abcd home
// inside any git working tree whatever the file says (adr-2610140859394097).
package homeignore

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

const (
	// IgnoreName is the ignore file laid in abcd's home.
	IgnoreName = ".gitignore"
	// MarkerName records that abcd laid IgnoreName, so a deleted ignore file
	// is never laid again.
	MarkerName = "gitignore-laid"
)

// ignoreBody is the ignore file: two comment lines a person who finds it can
// read, and the one pattern that matters. `*` matches the file itself and the
// marker too, so neither is ever staged.
const ignoreBody = "# Laid once by abcd so that a git repository in your home folder never stages abcd's records.\n" +
	"# Delete this file to let such a repository track this folder; abcd will not lay it again.\n" +
	"*\n"

// markerBody says what the marker is for, for the person who finds it.
const markerBody = "abcd laid .gitignore in this folder once. While this file stands, abcd will not lay it again,\n" +
	"so deleting .gitignore is how you let a git repository in your home folder track this folder.\n"

// Outcome is what one Lay did.
type Outcome int

const (
	// NoHome: abcd's home does not exist yet, so nothing was laid; the
	// command that creates it is followed by another lay.
	NoHome Outcome = iota
	// AlreadyLaid: the marker stands, so nothing was written.
	AlreadyLaid
	// Laid: the ignore file and the marker were written.
	Laid
	// KeptExisting: the person already keeps an ignore file there; it was
	// left as it is and the marker was written beside it.
	KeptExisting
)

// Lay makes sure the ignore file has been laid in abcd's home under home. It
// never creates the home, never overwrites a file, and never writes through a
// link: the home is opened through the home-scope walk, a symlinked
// ~/.abcd.noindex reads as NoHome, and both files are created exclusively,
// owner-only. An empty or relative home is NoHome: there is no one home it
// names.
func Lay(home string) (Outcome, error) {
	if home == "" || !filepath.IsAbs(home) {
		return NoHome, nil
	}
	root, err := fsutil.OpenHomeScope(home, abcdhome.Rel())
	if err != nil {
		// A symlinked ~/.abcd.noindex (a dotfiles layout) is read as absent,
		// as the configuration layer reads it: the walk refuses to write
		// through it, so no later run could lay the file either, and a fault
		// would repeat on every command.
		var link *fsutil.HomeScopeLinkError
		if os.IsNotExist(err) || errors.As(err, &link) {
			return NoHome, nil
		}
		return NoHome, err
	}
	defer root.Close()

	if _, err := root.Lstat(MarkerName); err == nil {
		return AlreadyLaid, nil
	} else if !os.IsNotExist(err) {
		return NoHome, err
	}
	out := Laid
	if err := fsutil.CreateExclusiveIn(root, IgnoreName, []byte(ignoreBody), abcdhome.FileMode); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return NoHome, err
		}
		out = KeptExisting
	}
	// The marker goes second, so a run that stops between the two leaves the
	// ignore file standing and the next run marks it (KeptExisting) rather than
	// a marker over no file. A marker another process wrote first is the same
	// lay.
	if err := fsutil.CreateExclusiveIn(root, MarkerName, []byte(markerBody), abcdhome.FileMode); err != nil && !errors.Is(err, os.ErrExist) {
		return NoHome, err
	}
	return out, nil
}
