// Package landings reads how each change reached the default branch
// (itd-2610301850454860, spc-2610301950480609).
//
// This file is the grammar of the one fact the implement loop writes before a
// change merges: the rule a lane was admitted under, carried as an
// `Admitted-by:` trailer on the landing's records commit, which the merge brings
// into the default branch's history. It is the one spelling: the loop composes
// the trailer through Admitted.Trailer, and the reader and the tests read it
// back through ParseAdmitted.
package landings

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// TrailerKey is the trailer's key.
const TrailerKey = "Admitted-by"

// The lane kinds a trailer names.
const (
	// KindDrain is a lane the drain opened on an issue its rule record lets it
	// take alone; the record is that rule's ADR id.
	KindDrain = "drain"
	// KindBuild is a lane `abcd build` opened: the record is the spec the run
	// builds against, or the issue a person's `abcd build <iss-N>` fixes.
	KindBuild = "build"
)

// Admitted is the rule a lane was admitted under.
type Admitted struct {
	// Kind is KindDrain or KindBuild.
	Kind string `json:"kind"`
	// Record is the drain rule's ADR id (adr-N), the spec id (spc-N), or the
	// issue id (iss-N).
	Record string `json:"record"`
	// Path is the drain rule record's repo-relative, slash-separated path, for
	// a drain's admission only. The trailer does not carry it: a reader finds
	// the record by its id.
	Path string `json:"path,omitempty"`
}

// issueIDRe is an issue id as the loop keys a run by one.
var issueIDRe = regexp.MustCompile(`^iss-[1-9][0-9]{0,19}$`)

// Check says why a is not an admission the loop writes, or "" when it is: a
// drain's names an ADR id in its canonical spelling and that record's path in
// the decision store, and a build's names a spec or an issue id and no path.
// The trailer is written from it into a commit message, so nothing else, a line
// break above all, ever reaches one.
func (a Admitted) Check() string {
	if why := a.checkValue(); why != "" {
		return why
	}
	switch a.Kind {
	case KindDrain:
		if a.Path == "" || path.Dir(a.Path) != drainrule.ADRsRelDir || recordid.ADRFileID(path.Base(a.Path)) != a.Record {
			return "a drain's admission names no path in " + drainrule.ADRsRelDir + "/ to its rule record " + a.Record
		}
	case KindBuild:
		if a.Path != "" {
			return "a build's admission names a path, which only a drain's carries"
		}
	}
	return ""
}

// checkValue is Check over what the trailer carries: the kind and the record.
func (a Admitted) checkValue() string {
	switch a.Kind {
	case KindDrain:
		if a.Record == "" || recordid.CanonADRID(a.Record) != a.Record {
			return "a drain's admission names no drain rule record by its id (adr-N)"
		}
	case KindBuild:
		if !recordid.ValidSpecID(a.Record) && !issueIDRe.MatchString(a.Record) {
			return "a build's admission names no spec (spc-N) or issue (iss-N)"
		}
	default:
		return "an admission's kind is neither " + KindDrain + " nor " + KindBuild
	}
	return ""
}

// Value is the trailer's value: the kind and the record id.
func (a Admitted) Value() string { return a.Kind + " " + a.Record }

// Trailer is the trailer line, `Admitted-by: <kind> <record>`.
func (a Admitted) Trailer() string { return TrailerKey + ": " + a.Value() }

// String is the run record's wording of the admission.
func (a Admitted) String() string {
	if a.Path != "" {
		return fmt.Sprintf("%s (%s)", a.Value(), a.Path)
	}
	return a.Value()
}

// trailerLineRe is one `Admitted-by:` line, its key in any case as git reads a
// trailer key.
var trailerLineRe = regexp.MustCompile(`(?i)^admitted-by\s*:\s*(\S+)\s+(\S+)\s*$`)

// ParseAdmitted reads the admission a commit message's trailer block (its last
// paragraph, as git reads trailers) carries. It reports false when the block
// carries none, carries two that differ, or carries one the loop never writes;
// the path is never in a trailer, so the one returned has none.
func ParseAdmitted(message string) (Admitted, bool) {
	paras := strings.Split(strings.TrimRight(strings.ReplaceAll(message, "\r\n", "\n"), "\n"), "\n\n")
	block := paras[len(paras)-1]
	var found *Admitted
	for line := range strings.SplitSeq(block, "\n") {
		m := trailerLineRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "admitted-by") {
				return Admitted{}, false
			}
			continue
		}
		a := Admitted{Kind: m[1], Record: m[2]}
		if a.checkValue() != "" {
			return Admitted{}, false
		}
		if found != nil && *found != a {
			return Admitted{}, false
		}
		found = &a
	}
	if found == nil {
		return Admitted{}, false
	}
	return *found, true
}
