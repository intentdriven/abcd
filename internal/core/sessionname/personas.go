// Package sessionname gives every session abcd's hooks run in a persona name,
// unique across the machine (itd-2610111707085098, spc-2610152057143798).
//
// A session that starts unnamed takes the next name of its repository's
// persona sequence that no running named session holds: Alice, Bob, Carol and
// on through the list, then Alice Two, Bob Two and so on. The names handed out
// are kept in one machine-scoped register, the sessions folder of abcd's home,
// beside an entry per running session naming the host process that ran its
// session-start hook. An entry whose process is gone, or whose process id now
// belongs to a process started at another time, has ended; when every entry
// has ended the handed-out set is emptied and naming starts again at Alice.
//
// The package decides and records; it never prints. Every outcome carries its
// diagnostics as notes for the front door to put on stderr, and naming never
// fails a hook: anything that goes wrong leaves the session unnamed with one
// note saying why (C11). The person's own name always wins: a session that
// arrives with a title is never given one (C4).
//
// Everything the register reads beyond its arguments, the home folder, the
// process table, the process the host walk starts from and the lock's wait,
// is a field of Machine, so a test drives a whole machine of sessions from a
// fixture table and a temporary home.
package sessionname

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// PersonasRel is the repository's persona list, repo-relative: the registry
// press-release composers attribute quotes from, and the source of the names
// sessions take.
const PersonasRel = ".abcd/development/personas.json"

// maxPersonasBytes caps the list's read. abcd's own list is under two
// kilobytes; the cap bounds a planted device or an endless file.
const maxPersonasBytes = 64 << 10

// bundled is the list a repository without a usable persona list names its
// sessions from. It is abcd's own registry, in its order, and
// TestBundledIsAbcdsOwnPersonaRegistry holds the two equal.
var bundled = []string{
	"Alice", "Bob", "Carol", "Dave", "Eve", "Frank", "Grace",
	"Henry", "Iris", "Jack", "Kira", "Liam", "Maya", "Nia",
}

// Bundled returns a copy of the bundled persona list.
func bundledList() []string { return append([]string(nil), bundled...) }

// rounds are the suffixes each pass through the list takes. The sequence ends
// after the ninth: a machine running more named sessions than nine passes of
// its list leaves the next one unnamed rather than inventing a tenth word.
var rounds = []string{"", " Two", " Three", " Four", " Five", " Six", " Seven", " Eight", " Nine"}

// Sequence is the order names are handed out in: names as listed, then each
// again with " Two", and so on to " Nine".
func sequenceOf(names []string) []string {
	seq := make([]string, 0, len(names)*len(rounds))
	for _, r := range rounds {
		for _, n := range names {
			seq = append(seq, n+r)
		}
	}
	return seq
}

// personaName is the one shape a repository's persona name may take before it
// is used (decision 8): one capitalised word of letters, at most sixteen
// characters. The name goes into the session list other sessions' models
// read, and the list is a tracked file a pull request can change, so a name
// carrying a space, a control byte, punctuation or a sentence is never used.
var personaName = regexp.MustCompile(`^[A-Z][a-z]{1,15}$`)

// Personas returns the persona names sessions in the repository at root take,
// and a note when the repository's list was not used.
//
// A repository with no list (or no root) takes the bundled list silently: that
// is the ordinary case. A list that is there and cannot be used takes the
// bundled list in its place, WHOLE, with one note naming the file and the
// reason. A list with one bad entry is not filtered down to its good ones:
// whoever wrote the bad one also chose the order of the rest. The note never
// quotes an entry, because the entry is what the check refused.
func personasOf(root string) ([]string, string) {
	if root == "" {
		return bundledList(), ""
	}
	names, err := readPersonas(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return bundledList(), ""
		}
		return bundledList(), fmt.Sprintf("abcd session name: %s is not used for session names (%s); the bundled names, Alice to Nia, are used instead", PersonasRel, err.Error())
	}
	return names, ""
}

// readPersonas reads and checks the list. Its error is the reason the list is
// not used, worded for the note; os.ErrNotExist means there is no list.
func readPersonas(root string) ([]string, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	// Read inside the repository's root, never following a link at the leaf
	// and never blocking on a device: the file is tracked, so it is whatever a
	// clone committed (fsutil.ReadGuardedInRoot).
	raw, err := fsutil.ReadGuardedInRoot(r, PersonasRel, maxPersonasBytes)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, err
	case errors.Is(err, fsutil.ErrTooBig):
		return nil, fmt.Errorf("it is larger than %d bytes", maxPersonasBytes)
	case err != nil:
		// The raw error can carry the absolute path; the reason names the
		// file's kind of failure only.
		return nil, errors.New("it is not a regular file that can be read")
	}
	var doc struct {
		Personas []struct {
			Name string `json:"name"`
		} `json:"personas"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		// A decoder error quotes the value it failed on, which is the entry
		// the note must not echo.
		return nil, errors.New("it is not a JSON object whose personas list holds names")
	}
	if len(doc.Personas) == 0 {
		return nil, errors.New("it lists no persona names")
	}
	names := make([]string, 0, len(doc.Personas))
	seen := make(map[string]bool, len(doc.Personas))
	for i, p := range doc.Personas {
		if !personaName.MatchString(p.Name) {
			return nil, fmt.Errorf("entry %d is not one capitalised word of letters of at most sixteen characters", i+1)
		}
		if seen[p.Name] {
			return nil, fmt.Errorf("entry %d repeats an earlier name", i+1)
		}
		seen[p.Name] = true
		names = append(names, p.Name)
	}
	return names, nil
}
