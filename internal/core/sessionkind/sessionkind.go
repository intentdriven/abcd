// Package sessionkind is the per-run context stamp: the one token a reading
// bundle and a scribe context each carry, naming the kind of session the
// context is for, the run it belongs to and a digest of what it holds
// (adr-2609021016275803, spc-2609020626045177).
//
// It is a LEAF, and that is the reason it exists as a package at all. Two
// packages read the stamp: core/reading writes it into the bundle, and
// core/history recognises it in a transcript at capture and checks separation
// over what it recorded. core/scribe writes it too. None of the three may import
// another for it — history importing reading would pull the whole assembler
// into the transcript store — so the grammar lives here, once, and each of them
// imports this.
//
// The stamp is PER RUN on purpose. A fixed token per kind would sit committed in
// the documentation and the code, so any session that read either would carry
// both kinds and be reported in breach, and a session that reached the ledger by
// another route would carry one and be reported clean; the ADR rejects that
// mechanism by name. A stamp names one run and a digest, and it is matched
// exactly, so a session that reads the docs carries none and a transcript
// carrying the reading stamp of one run and the scribe stamp of another is not
// a violation.
package sessionkind

import (
	"fmt"
	"regexp"

	"github.com/intentdriven/abcd/internal/core/recordid"
)

// Kind is the kind of session a context is assembled for. The set is closed.
type Kind string

// The two kinds.
const (
	Reading Kind = "reading"
	Scribe  Kind = "scribe"
)

// Prefix opens every stamp. It is carried in the documentation, which is why a
// prefix alone is never a stamp: StampRe requires a kind, a well-formed run and
// a full digest behind it.
const Prefix = "abcd.context-stamp"

// DigestLen is how many hex digits of the context's sha256 a stamp carries.
const DigestLen = 12

// StampRe is the ONE expression that recognises a stamp. Find adds the
// boundaries RE2 cannot express as lookarounds; every reader goes through Find
// or Parse rather than this expression bare.
var StampRe = regexp.MustCompile(`abcd\.context-stamp/(reading|scribe)/(rdg-[0-9]+)/([0-9a-f]{12})`)

// exactRe is StampRe anchored, for Parse.
var exactRe = regexp.MustCompile(`^` + StampRe.String() + `$`)

// Parsed is one stamp taken apart.
type Parsed struct {
	Kind   Kind
	Run    string
	Digest string
}

// Stamp renders the stamp for one context: kind, run, and the first DigestLen
// hex digits of the context's sha256. It refuses a kind outside the closed set,
// a run that is not a reading run id and a digest that is not hex or is too
// short to cut, so no caller can mint a stamp Parse would not read back.
func Stamp(kind Kind, run, sha256Hex string) (string, error) {
	if kind != Reading && kind != Scribe {
		return "", fmt.Errorf("sessionkind: kind %q is not one of %q, %q", kind, Reading, Scribe)
	}
	if !recordid.ValidReadingRunID(run) {
		return "", fmt.Errorf("sessionkind: run %q is not a reading run id (rdg-N)", run)
	}
	if len(sha256Hex) < DigestLen {
		return "", fmt.Errorf("sessionkind: digest %q is shorter than %d hex digits", sha256Hex, DigestLen)
	}
	s := Prefix + "/" + string(kind) + "/" + run + "/" + sha256Hex[:DigestLen]
	if !exactRe.MatchString(s) {
		return "", fmt.Errorf("sessionkind: digest %q is not lowercase hex", sha256Hex)
	}
	return s, nil
}

// Parse reads one stamp, exactly: the whole string must be a stamp and nothing
// else.
func Parse(s string) (Parsed, bool) {
	m := exactRe.FindStringSubmatch(s)
	if m == nil {
		return Parsed{}, false
	}
	return Parsed{Kind: Kind(m[1]), Run: m[2], Digest: m[3]}, true
}

// Find returns every distinct stamp in b, in first-seen order.
//
// A match counts only between boundaries: the character before it may not
// continue a name (so `xabcd.context-stamp/...` is not a stamp) and the
// character after it may not continue the digest or the run (so a thirteenth hex
// digit, or any letter, disqualifies it). Without the trailing boundary a longer
// digest would be read as a shorter stamp that is not the one written.
func Find(b []byte) []string {
	var out []string
	seen := map[string]bool{}
	for _, loc := range StampRe.FindAllIndex(b, -1) {
		start, end := loc[0], loc[1]
		if start > 0 && continuesName(b[start-1]) {
			continue
		}
		if end < len(b) && continuesName(b[end]) && b[end] != '.' {
			continue
		}
		s := string(b[start:end])
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// continuesName reports whether c would continue a token a stamp is part of. A
// full stop after the digest is a sentence ending rather than a continuation, so
// Find admits it there and only there.
func continuesName(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	case c == '_' || c == '-' || c == '.':
		return true
	}
	return false
}
