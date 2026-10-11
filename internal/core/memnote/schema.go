// Package memnote is the memory-notes store (itd-2610091918433290,
// spc-2610111627213138 step 1): short lessons an agent keeps so a later session
// does not relearn them. Notes live in a memory.json beside each rules.json
// layer — the machine's ~/.abcd.noindex/memory.json holds personal notes, and a
// repository's .abcd/memory.json holds the notes a person promoted into the
// project — and are read with the trust checks the rules layers get.
//
// Agents write only the machine layer, and only through the write gate
// (gate.go): the shared scanner and the private-names banlist refuse a note
// that would hold a secret, a private name or a network address, and every
// note is stamped with who wrote it, when and in which session. Nothing in this
// package writes the repository layer, and nothing here promotes a note: that
// is a later dashboard's act (itd-2610111655362931). Nothing here writes to
// stdout or knows a transport; the `abcd memory` verb is the front door.
package memnote

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/assistedby"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// SchemaVersion is the one memory.json schema both layers carry.
const SchemaVersion = 1

// Family is the record-id family a note's id is minted in (recordid.Minter,
// adr-45): mem-<yymmddHHMMSS><rrrr>.
const Family = "mem"

// The note's figures, set in one place (the spec's starting points). A write
// past one is refused naming the limit.
const (
	// MaxTitleChars is a title's length in characters, on one line.
	MaxTitleChars = 120
	// MaxBodyBytes is a body's length in bytes.
	MaxBodyBytes = 2 << 10
	// MinRecall and MaxRecall bound the recall list's length.
	MinRecall = 1
	MaxRecall = 12
	// MaxRecallChars is one recall word or short phrase's length in characters.
	MaxRecallChars = 64
	// MaxSessionChars is a session id's length.
	MaxSessionChars = 128
)

// The two origins a note can carry.
const (
	// OriginAgent is a note an agent wrote of its own accord.
	OriginAgent = "agent"
	// OriginHostMemoryMove is a lesson the person pointed at in the host's own
	// memory files, moved by hand with `memory add --moved-from-host`.
	OriginHostMemoryMove = "host-memory-move"
)

// Note is one memory note. Root is nil for a note recalled in every
// repository (`--everywhere`) and for every note in the repository layer,
// which carries no root; Session is nil where the host supplied none.
type Note struct {
	ID        string
	Title     string
	Body      string
	Recall    []string
	Root      *string
	Origin    string
	WrittenBy string
	WrittenAt string
	Session   *string
}

// File is one memory.json.
type File struct {
	SchemaVersion int
	Notes         []Note
}

// wireNote is a note as the file spells it. Root is raw so the reader can tell
// a key that is absent (the repository layer's shape) from one that is null
// (a machine note recalled everywhere).
type wireNote struct {
	ID        string          `json:"id"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Recall    []string        `json:"recall"`
	Root      json.RawMessage `json:"root,omitempty"`
	Origin    string          `json:"origin"`
	WrittenBy string          `json:"written_by"`
	WrittenAt string          `json:"written_at"`
	Session   *string         `json:"session"`
}

type wireFile struct {
	SchemaVersion int        `json:"schema_version"`
	Notes         []wireNote `json:"notes"`
}

// idRe is a minted note id: the family, a 12-digit UTC second stamp and a
// 4-digit suffix.
var idRe = regexp.MustCompile(`^mem-[0-9]{16}$`)

// sessionRe is a host session id: letters, digits, '-' and '_' only, so no
// link, host name or path can enter a note as its session (no ':', '/' or
// '.'), and so the id carries no session URL.
var sessionRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

// validID reports whether s is a note id's shape.
func validID(s string) bool { return idRe.MatchString(s) }

// layerKind says which layer a note is judged as: the machine layer requires a
// root key (a full object name, or null), the repository layer forbids one.
type layerKind int

const (
	machineLayer layerKind = iota
	repoLayer
)

// FieldError is a refusal of one field of one note, naming the field and the
// limit or shape it broke, never echoing the value.
type FieldError struct {
	Field string
	Why   string
}

func (e *FieldError) Error() string { return e.Field + " " + e.Why }

// ErrMalformed wraps every refusal of a file's contents.
var ErrMalformed = errors.New("malformed memory file")

// checkTitle refuses a title that is empty, longer than MaxTitleChars, not one
// line, or carrying a control, bidirectional or zero-width character.
func checkTitle(t string) error {
	switch {
	case strings.TrimSpace(t) == "":
		return &FieldError{"title", "is empty"}
	case !utf8.ValidString(t):
		return &FieldError{"title", "is not valid UTF-8"}
	case utf8.RuneCountInString(t) > MaxTitleChars:
		return &FieldError{"title", fmt.Sprintf("is longer than %d characters", MaxTitleChars)}
	case strings.ContainsAny(t, "\r\n"):
		return &FieldError{"title", "is more than one line"}
	case termsafe.Sanitize(t) != t:
		return &FieldError{"title", "carries a control, bidirectional or zero-width character"}
	}
	return nil
}

// checkBody refuses a body that is empty, over MaxBodyBytes, or carrying a
// control, bidirectional or zero-width character other than a newline or tab.
func checkBody(b string) error {
	switch {
	case strings.TrimSpace(b) == "":
		return &FieldError{"body", "is empty"}
	case !utf8.ValidString(b):
		return &FieldError{"body", "is not valid UTF-8"}
	case len(b) > MaxBodyBytes:
		return &FieldError{"body", fmt.Sprintf("is longer than %d bytes", MaxBodyBytes)}
	}
	for _, r := range b {
		if r == '\n' || r == '\t' {
			continue
		}
		if termsafe.Sanitize(string(r)) != string(r) {
			return &FieldError{"body", "carries a control, bidirectional or zero-width character"}
		}
	}
	return nil
}

// checkRecall refuses a recall list outside MinRecall..MaxRecall, or holding a
// word that is empty, longer than MaxRecallChars, not one line, carrying a
// control character, holding no letter or digit (the loader's term rule
// matches only those), or repeating another.
func checkRecall(words []string) error {
	if len(words) < MinRecall || len(words) > MaxRecall {
		return &FieldError{"recall", fmt.Sprintf("must hold %d to %d words or short phrases, not %d", MinRecall, MaxRecall, len(words))}
	}
	seen := map[string]bool{}
	for i, w := range words {
		f := fmt.Sprintf("recall[%d]", i)
		switch {
		case strings.TrimSpace(w) == "" || strings.TrimSpace(w) != w:
			return &FieldError{f, "is empty or begins or ends with white space"}
		case !utf8.ValidString(w):
			return &FieldError{f, "is not valid UTF-8"}
		case utf8.RuneCountInString(w) > MaxRecallChars:
			return &FieldError{f, fmt.Sprintf("is longer than %d characters", MaxRecallChars)}
		case strings.ContainsAny(w, "\r\n") || termsafe.Sanitize(w) != w:
			return &FieldError{f, "is not one plain line"}
		case strings.IndexFunc(w, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) < 0:
			return &FieldError{f, "holds no letter or digit, so no prompt could match it"}
		}
		k := strings.ToLower(w)
		if seen[k] {
			return &FieldError{f, "repeats an earlier recall word"}
		}
		seen[k] = true
	}
	return nil
}

// checkWrittenBy refuses a writer stamp that is not the Assisted-by trailer's
// model form, <Vendor>:<model-version>.
func checkWrittenBy(v string) error {
	if !assistedby.IsModelValue(v) {
		return &FieldError{"written_by", "is not the <Vendor>:<model-version> form an Assisted-by trailer carries"}
	}
	return nil
}

// checkSession refuses a session id that is longer than MaxSessionChars or is
// not letters, digits, '-' and '_' — so a link, which carries ':', '/' or '.',
// is refused and no session URL can enter a note.
func checkSession(s string) error {
	switch {
	case len(s) > MaxSessionChars:
		return &FieldError{"session", fmt.Sprintf("is longer than %d characters", MaxSessionChars)}
	case !sessionRe.MatchString(s):
		return &FieldError{"session", "is not a host session id (letters, digits, '-' and '_'; a link is refused)"}
	}
	return nil
}

// checkNote validates one note as layer reads it.
func checkNote(n Note, layer layerKind, rootKeyPresent bool) error {
	if !validID(n.ID) {
		return &FieldError{"id", "is not a mem-<16 digits> note id"}
	}
	for _, err := range []error{checkTitle(n.Title), checkBody(n.Body), checkRecall(n.Recall), checkWrittenBy(n.WrittenBy)} {
		if err != nil {
			return err
		}
	}
	switch layer {
	case machineLayer:
		if !rootKeyPresent {
			return &FieldError{"root", "is missing (a full object name, or null for a note recalled everywhere)"}
		}
		if n.Root != nil && !gitutil.IsFullSHA(*n.Root) {
			return &FieldError{"root", "is not a full object name"}
		}
	case repoLayer:
		if rootKeyPresent {
			return &FieldError{"root", "is not carried in the project's file, which is already the project's"}
		}
	}
	switch n.Origin {
	case OriginAgent, OriginHostMemoryMove:
	default:
		return &FieldError{"origin", fmt.Sprintf("is neither %q nor %q", OriginAgent, OriginHostMemoryMove)}
	}
	if t, err := time.Parse(time.RFC3339, n.WrittenAt); err != nil || t.UTC().Format(time.RFC3339) != n.WrittenAt {
		return &FieldError{"written_at", "is not a UTC RFC 3339 time (2026-10-11T12:00:00Z)"}
	}
	if n.Session != nil {
		if err := checkSession(*n.Session); err != nil {
			return err
		}
	}
	return nil
}

// parseFile decodes and validates one memory.json as layer, refusing an
// unknown field, a duplicate key, trailing content, another schema version, a
// malformed note, or two notes sharing an id.
func parseFile(data []byte, layer layerKind) (File, error) {
	var w wireFile
	if err := jsonstrict.Decode(data, &w); err != nil {
		return File{}, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	if w.SchemaVersion != SchemaVersion {
		return File{}, fmt.Errorf("%w: schema_version must be %d, got %d", ErrMalformed, SchemaVersion, w.SchemaVersion)
	}
	f := File{SchemaVersion: w.SchemaVersion}
	ids := map[string]bool{}
	for i, wn := range w.Notes {
		n := Note{ID: wn.ID, Title: wn.Title, Body: wn.Body, Recall: wn.Recall, Origin: wn.Origin,
			WrittenBy: wn.WrittenBy, WrittenAt: wn.WrittenAt, Session: wn.Session}
		present := wn.Root != nil
		if present && string(wn.Root) != "null" {
			var s string
			if err := json.Unmarshal(wn.Root, &s); err != nil {
				return File{}, fmt.Errorf("%w: note %d: root is neither a string nor null", ErrMalformed, i)
			}
			n.Root = &s
		}
		if err := checkNote(n, layer, present); err != nil {
			return File{}, fmt.Errorf("%w: note %d: %v", ErrMalformed, i, err)
		}
		if ids[n.ID] {
			return File{}, fmt.Errorf("%w: note %d: id %s appears twice", ErrMalformed, i, n.ID)
		}
		ids[n.ID] = true
		f.Notes = append(f.Notes, n)
	}
	return f, nil
}

// encodeMachine renders a machine-layer file: every note carries its root key,
// a full object name or null.
func encodeMachine(f File) ([]byte, error) {
	w := wireFile{SchemaVersion: SchemaVersion, Notes: []wireNote{}}
	for _, n := range f.Notes {
		root := json.RawMessage("null")
		if n.Root != nil {
			b, err := json.Marshal(*n.Root)
			if err != nil {
				return nil, err
			}
			root = b
		}
		w.Notes = append(w.Notes, wireNote{ID: n.ID, Title: n.Title, Body: n.Body, Recall: n.Recall, Root: root,
			Origin: n.Origin, WrittenBy: n.WrittenBy, WrittenAt: n.WrittenAt, Session: n.Session})
	}
	b, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
