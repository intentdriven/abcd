package launch

// target.go — an intent's `target_release` (itd-2609212103572513): the one
// forward-looking line a derived release keeps (adr-2609212115255771,
// decision 3). A planned intent may name the release it must land by; the
// preview and the cut report every such intent still unshipped, and neither
// refuses on one.
//
// The value's shape and the listing's shape live here, in the lowest package
// that reads releases, because three readers judge the same value — the intent
// verbs that write it, the record lint that refuses it on a shipped or
// superseded record, and the cut that reports it — and the lint cannot import
// the intent store. One predicate, so the writer and the gate cannot drift.

import (
	"fmt"
	"strings"
)

// TargetReleaseKey is the intent frontmatter key the target lives under,
// spelled once.
const TargetReleaseKey = "target_release"

// TargetNext is the symbolic target: the next release, whatever version it
// derives. A derived release cannot be numbered before it is cut, so `next` is
// the one spelling that names it without guessing.
const TargetNext = "next"

// ValidTargetRelease reports whether value is a legal target: `next`, or a
// release tag `vX.Y.Z` (the strict SemVer core, with a leading `v`, no
// pre-release and no build metadata, because a release this repository cuts is
// always a bare core version). The error names the accepted shapes.
func ValidTargetRelease(value string) error {
	if value == TargetNext {
		return nil
	}
	core, ok := strings.CutPrefix(value, "v")
	if ok {
		if v, err := ParseSemver(core); err == nil && v.Prerelease == "" && v.Build == "" {
			return nil
		}
	}
	return fmt.Errorf("a target release is `next` or a release tag vX.Y.Z (e.g. v0.11.0), not %q", value)
}

// TargetedIntent is one planned intent that names a release it must land by:
// the row the preview, the cut and their pre-flight reports list. Every field
// comes out of a record, so a front door sanitises each before it reaches a
// terminal.
type TargetedIntent struct {
	// ID is the intent's id (itd-N).
	ID string `json:"id"`
	// Path is the record's repo-relative path.
	Path string `json:"path"`
	// Target is the `target_release` value as the record carries it.
	Target string `json:"target_release"`
	// Invalid is why Target is not a legal target, empty when it is. The row is
	// listed either way: the report never drops a record, and the record lint is
	// what refuses the value.
	Invalid string `json:"invalid,omitempty"`
}

// TargetMove is one targeted intent a cut passes without shipping it: the row
// the cut rewrites to `next` in the change that rolls the changelog, and the
// changelog's move note names (criterion 3, ruling BS1 of 2026-09-29).
type TargetMove struct {
	// ID is the intent's id (itd-N).
	ID string `json:"id"`
	// Path is the record's repo-relative path.
	Path string `json:"path"`
	// From is the target the record carried before the cut: `next`, which
	// named the release being cut, or a tag at or below it.
	From string `json:"from"`
}

// MissedTargets returns every target the cut to nextTag passes, in the order
// given: `next`, which names the release being cut, and a tag at or below
// nextTag, which names this release or one that will now never be cut. Each
// becomes `next` — the following release, whatever version it derives — so a
// missed target never goes stale and is never renumbered by guess (the product
// thinker's ruling BS1 of 2026-09-29). A tag above nextTag is still ahead and
// is not listed; neither is a value that is not a target (the record lint
// names it), nor anything when nextTag is empty, because a refused cut
// derives no version and writes nothing.
func MissedTargets(targets []TargetedIntent, nextTag string) []TargetMove {
	cut, err := ParseSemver(strings.TrimPrefix(nextTag, "v"))
	if err != nil || nextTag == "" {
		return nil
	}
	var out []TargetMove
	for _, tg := range targets {
		if tg.Invalid != "" || ValidTargetRelease(tg.Target) != nil {
			continue
		}
		if tg.Target != TargetNext {
			v, err := ParseSemver(strings.TrimPrefix(tg.Target, "v"))
			if err != nil || CoreGreater(v, cut) {
				continue
			}
		}
		out = append(out, TargetMove{ID: tg.ID, Path: tg.Path, From: tg.Target})
	}
	return out
}
