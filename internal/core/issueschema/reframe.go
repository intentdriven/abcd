package issueschema

// The reframe record (itd-2609020625402518, spc-2609020626048705).
//
// A reframe occasioned by a reading is recorded as a reframe: one record naming
// what occasioned it, the content fingerprint of each of the frame's three
// committed surfaces before and after the rewrite, which of them changed, and
// the grounds. It carries NO text of any surface. adr-55 keeps the construal as
// it presently stands in the record and its history on the local side, and
// adr-2609021016288378 adds this record as a committed pointer to an event
// whose content stays there: the fingerprints identify which frame a reading
// saw and which replaced it, and nothing the record holds can reproduce the
// framing it abandoned.
//
// The record is written in two halves when it precedes the rewrite's commit:
// the before fingerprints and the grounds first, the after fingerprints and
// `changed` once the rewrite is committed. So the required set is the first
// half, and the allow-list is that set plus the after half.

import (
	"regexp"
	"strings"
)

const (
	// ReframeFamily identifies one reframe record (rfm-N). It mints through
	// recordid.Minter.Mint like every family in this workstream (adr-45).
	ReframeFamily = "rfm"
	// ReframesDir is FLAT, like SurprisesDir: reframes/rfm-<N>.md. A reframe is
	// keyed by what it carries — its occasion and its fingerprints — never by a
	// directory, and like the other reading-chain families it is deliberately
	// not in StatusDirs.
	ReframesDir = "reframes"
)

// ReframeRequired is every property a reframe record carries from its first
// write: the occasion, the three before fingerprints and the grounds.
//
// It is the ONE list: core/lint's rfm store and core/capture's writer both
// read it, so the gate and the writer cannot disagree about what a well-formed
// reframe carries.
var ReframeRequired = []string{
	"schema_version", "id",
	"occasioned_by",
	"construal_before", "glossary_before", "scope_before",
	"grounds",
}

// ReframeAfter is the half a reframe record gains when it is complete: the
// three after fingerprints and `changed`, the surfaces whose fingerprints
// differ. All four are absent while the record is open and present once it is
// complete, never some of them.
var ReframeAfter = []string{"construal_after", "glossary_after", "scope_after", "changed"}

// ReframeKnown is the reframe record's allow-list: the required set plus the
// after half, and nothing else — no key that could hold a surface's text.
var ReframeKnown = func() map[string]bool {
	known := knownSet(ReframeRequired)
	for _, f := range ReframeAfter {
		known[f] = true
	}
	return known
}()

// ReframeOccasionFamilies is the CLOSED set of families a reframe's
// `occasioned_by` may name: a reading item, a disposition or a surprise. It is
// the one list the verb resolves the occasion over and the record gate holds a
// committed record to.
var ReframeOccasionFamilies = []string{ReadingItemFamily, DispositionFamily, SurpriseFamily}

// FrameSurfaceNames are the three committed surfaces the frame is, in the
// order the record carries them (adr-55's enumeration, as adr-2609021016288378
// adopted it): the framing chapter's construal section, the committed glossary
// terms and the committed scope. `changed` is drawn from this vocabulary, and
// each name N is carried as the pair N_before / N_after.
var FrameSurfaceNames = []string{"construal", "glossary", "scope"}

// fingerprintRe is the shape every fingerprint takes: a SHA-256 in lower-case
// hex, and nothing else.
var fingerprintRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ValidFingerprint reports whether v is verbatim a lower-case 64-hex SHA-256.
func ValidFingerprint(v string) bool { return fingerprintRe.MatchString(v) }

// ValidFrameSurface reports whether v names one of the three frame surfaces.
func ValidFrameSurface(v string) bool {
	for _, n := range FrameSurfaceNames {
		if v == n {
			return true
		}
	}
	return false
}

// ValidReframeOccasion reports whether v is VERBATIM a handle of one of
// ReframeOccasionFamilies: the family's own prefix, one hyphen and digits, with
// nothing around it.
func ValidReframeOccasion(v string) bool { return validHandleOf(ReframeOccasionFamilies, v) }

// validHandleOf reports whether v is verbatim a handle of one of families.
func validHandleOf(families []string, v string) bool {
	for _, f := range families {
		rest, ok := strings.CutPrefix(v, f+"-")
		if !ok || rest == "" {
			continue
		}
		digits := true
		for i := 0; i < len(rest); i++ {
			if rest[i] < '0' || rest[i] > '9' {
				digits = false
				break
			}
		}
		if digits {
			return true
		}
	}
	return false
}
