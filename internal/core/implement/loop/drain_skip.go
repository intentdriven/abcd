package loop

// drain_skip.go is the drain's skip list (spc-2609212015054359 step 4;
// iss-2610090642385898): a person holds back one issue, or every issue whose
// found_at falls in one path, for the drain in progress, each with a reason.
// A held-back issue is passed over before anything is pushed or asked, named
// in the move's passed entries with the reason, and the list clears when the
// drain ends, so a hold for now never outlives the drain it was given to.

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// CheckSkipped is the pass-over of an issue the drain's skip list holds back.
const CheckSkipped = "skipped"

// maxDrainSkips caps the skip list, and maxSkipReasonBytes one entry's reason.
const (
	maxDrainSkips      = 64
	maxSkipReasonBytes = 512
)

// DrainSkip is one entry of the drain's skip list: an issue id, or a
// repository-relative path that holds back every issue whose found_at path
// lies in it or contains it, with the person's reason and when it was given.
type DrainSkip struct {
	Target string    `json:"target"`
	Reason string    `json:"reason"`
	At     time.Time `json:"at"`
}

// skipTarget normalises a --skip or --unskip target: an issue id as given, or
// a clean repository-relative path (a trailing slash dropped). False for
// anything else, and for a target that is not valid UTF-8 or carries a rune
// termsafe masks (a C1, bidi or zero-width control): the drain's reader
// refuses such text in its state, so storing one would wedge every later move.
func skipTarget(t string) (string, bool) {
	t = strings.TrimSpace(t)
	if !utf8.ValidString(t) || termsafe.Sanitize(t) != t {
		return "", false
	}
	if validIssueKey(t) {
		return t, true
	}
	p := strings.TrimSuffix(t, "/")
	if p == "" || strings.ContainsRune(p, '\\') {
		return "", false
	}
	if c := path.Clean(p); fsutil.ValidRelPath(c) && c == p {
		return c, true
	}
	return "", false
}

// checkSkips validates the drain's skip flags before anything is read or
// written: every target an issue id or a clean relative path, a reason with
// every --skip and none without one.
func checkSkips(d DrainOptions) error {
	if len(d.Skip) == 0 && strings.TrimSpace(d.SkipReason) != "" {
		return refuse(StageDrain, "", "", "--reason names why an issue is held back, and no --skip names one",
			"give --skip <iss-N|path> with --reason, or leave --reason out")
	}
	for _, list := range [][]string{d.Skip, d.Unskip} {
		for _, t := range list {
			if _, ok := skipTarget(t); !ok {
				return refuse(StageDrain, "", "", fmt.Sprintf("%q is neither an issue id nor a repository-relative path", termsafe.Sanitize(t)),
					"name an issue (iss-<N>) or a path inside the checkout, such as internal/core/capture")
			}
		}
	}
	if len(d.Skip) == 0 {
		return nil
	}
	reason := strings.TrimSpace(d.SkipReason)
	if reason == "" {
		return refuse(StageDrain, "", "", "--skip holds an issue back for now, and each hold carries its reason; no --reason was given",
			"give --reason \"<why it is held back>\" with --skip")
	}
	if !utf8.ValidString(reason) || len(termsafe.Sanitize(reason)) > maxSkipReasonBytes {
		return refuse(StageDrain, "", "", fmt.Sprintf("--reason is not UTF-8 text within %d bytes", maxSkipReasonBytes),
			"give a shorter reason")
	}
	return nil
}

// applySkips adds this move's --skip entries to the drain's list (a target
// already there takes the new reason) and takes its --unskip entries off,
// returning what it did in words, "" when nothing was asked. A list that would
// pass the cap is refused before anything is written.
func applySkips(st *DrainState, d DrainOptions, now time.Time) (string, error) {
	if len(d.Skip) == 0 && len(d.Unskip) == 0 {
		return "", nil
	}
	var clauses []string
	for _, raw := range d.Unskip {
		t, _ := skipTarget(raw)
		if i := slices.IndexFunc(st.Skips, func(s DrainSkip) bool { return s.Target == t }); i >= 0 {
			st.Skips = slices.Delete(st.Skips, i, i+1)
			clauses = append(clauses, t+" is no longer held back")
		} else {
			clauses = append(clauses, t+" was not held back, so --unskip changed nothing for it")
		}
	}
	reason := termsafe.Sanitize(strings.TrimSpace(d.SkipReason))
	var added []string
	for _, raw := range d.Skip {
		t, _ := skipTarget(raw)
		if i := slices.IndexFunc(st.Skips, func(s DrainSkip) bool { return s.Target == t }); i >= 0 {
			st.Skips[i] = DrainSkip{Target: t, Reason: reason, At: now}
		} else {
			st.Skips = append(st.Skips, DrainSkip{Target: t, Reason: reason, At: now})
		}
		if !slices.Contains(added, t) {
			added = append(added, t)
		}
	}
	if len(st.Skips) > maxDrainSkips {
		return "", refuse(StageDrain, "", "", fmt.Sprintf("the skip list would hold %d entries; a drain holds back at most %d", len(st.Skips), maxDrainSkips),
			"hold back a path that covers several issues, or --unskip entries no longer needed")
	}
	if len(added) > 0 {
		clauses = append(clauses, "held back for now: "+strings.Join(added, ", ")+" ("+reason+"); the skip list clears when this drain ends")
	}
	return strings.Join(clauses, "; "), nil
}

// skipOf is the skip-list entry that holds iss back: one naming its id, or a
// path its found_at path lies in or contains. False when none does.
func skipOf(st DrainState, iss capture.Issue) (DrainSkip, bool) {
	var fp []string
	if p, ok := issueFootprint(iss); ok {
		fp = p
	}
	for _, s := range st.Skips {
		if s.Target == iss.ID {
			return s, true
		}
		if validIssueKey(s.Target) || len(fp) == 0 {
			continue
		}
		if covers([]string{s.Target}, fp) || covers(fp, []string{s.Target}) {
			return s, true
		}
	}
	return DrainSkip{}, false
}

// skipPassed is the pass-over an issue the skip list holds back is named with.
func skipPassed(id string, s DrainSkip) Excluded {
	by := "the issue"
	if s.Target != id {
		by = "its found_at path, in " + s.Target + ","
	}
	return Excluded{ID: id, Check: CheckSkipped, Reason: fmt.Sprintf("held back for now: %s is on the drain's skip list since %s: %s; passed over with nothing pushed and no judgement asked, until `abcd drain --unskip %s` or the drain ends",
		by, stamp(s.At), s.Reason, s.Target)}
}

// skipsWritten reports whether every skip-list entry is one the drain writes.
func (st DrainState) skipsWritten() bool {
	if len(st.Skips) > maxDrainSkips {
		return false
	}
	for _, s := range st.Skips {
		if t, ok := skipTarget(s.Target); !ok || t != s.Target || strings.TrimSpace(s.Reason) == "" || len(s.Reason) > maxSkipReasonBytes {
			return false
		}
	}
	return true
}
