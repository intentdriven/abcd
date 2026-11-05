package landing

// loopreview.go is the build loop's own review receipt. Where the facilitator mode
// is auto, a lane whose validators passed needs no person to record that they did:
// the loop records the pair of its latest round, in process, from the returns that
// round read. The landing adds a records commit after that round, so the head it
// pushes is not the head the pair read. The receipt for the pushed head rests on the
// pair at the judged head while the one commit between them holds only records, and
// the merge gate judges that again at arming.
//
// The delta rule (delta.go) is not this rule: a delta changes the wording of a file
// already there, and a records commit moves records and rewrites their frontmatter.

import (
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/layered"
)

// landingRecords are the directories the landing's records commit writes in: a
// spec's step marked and its close, an intent shipped, an issue resolved. Each path
// below them is a markdown record of mode 100644, added, changed or removed.
var landingRecords = []string{
	".abcd/development/specs/",
	".abcd/development/intents/",
	".abcd/work/issues/",
}

// MaxLandingRecords is how many paths a records commit may touch, counted once
// each, a moved record being two. Past it the lane waits for a person's review.
const MaxLandingRecords = 24

// RecordLoopReview writes the review receipts of the build loop's latest round: the
// full pair's for judged, and, when the landing's records commit moved the head, one
// for head resting on it. Anything short of a pair that passed and names judged, or a
// head that is more than the records on judged, is refused and head gets no receipt.
// A head whose receipt already stands against tip is left as it is.
func RecordLoopReview(repoRoot, tip, judged, head string, returns []ReviewReturn) (string, error) {
	if readReviewProof(repoRoot, head, tip).reviewed {
		return "", nil
	}
	if err := holdsCommit(repoRoot, judged); err != nil {
		return "", err
	}
	if err := holdsCommit(repoRoot, head); err != nil {
		return "", err
	}
	entries, err := judgeReturns(returns, func(text string) error { return statesFullPair(text, judged) })
	if err != nil {
		return "", err
	}
	var d deltaRecord
	if head != judged {
		if d, err = recordsChange(repoRoot, judged, head); err != nil {
			return "", err
		}
	}
	path, err := writeReviewRecord(repoRoot, newReviewRecord(judged, entries, nil))
	if err != nil || head == judged {
		return path, err
	}
	return writeReviewRecord(repoRoot, newReviewRecord(head, entries, &d))
}

// AutoAt reports whether the facilitator mode the merge gate reads at tip is auto.
func AutoAt(r layered.Roots, repoRoot, tip, name string) bool {
	m, err := readFacilitatorAt(r, repoRoot, tip, name)
	return err == nil && m.Mode == ModeAuto
}

// judgeRecords answers whether head may rest on the full pair at base as the
// landing's records commit: base has that pair, and head is one commit on it that
// holds only records. The error says what fails.
func judgeRecords(repoRoot, base, head string) (deltaRecord, error) {
	if err := hasFullPair(repoRoot, base); err != nil {
		return deltaRecord{}, err
	}
	return recordsChange(repoRoot, base, head)
}

// recordsChange reads the one commit from base to head, refusing anything but a
// single-parent commit on base that changes records within the bound. History is
// read as stored, whatever a replace ref or a graft says.
func recordsChange(repoRoot, base, head string) (deltaRecord, error) {
	out, err := storedBytes(repoRoot, maxDeltaGitOutput, "rev-list", "--parents", "-n", "1", head)
	if err != nil {
		return deltaRecord{}, fmt.Errorf("git cannot read %s's parents: %w", short(head), err)
	}
	if f := strings.Fields(string(out)); len(f) != 2 || f[1] != base {
		return deltaRecord{}, fmt.Errorf("%s does not stand on %s as one commit, so the pair that read %s did not read it", short(head), short(base), short(base))
	}
	raw, err := storedBytes(repoRoot, maxDeltaGitOutput, "diff-tree", "-r", "--no-renames", "--no-ext-diff", "--no-textconv", "--no-abbrev", "--ignore-submodules=none", "--raw", "-z", base, head)
	if err != nil {
		return deltaRecord{}, fmt.Errorf("git cannot read what %s changes: %w", short(head), err)
	}
	fields := strings.Split(strings.TrimSuffix(string(raw), "\x00"), "\x00")
	if len(raw) == 0 || len(fields)%2 != 0 {
		return deltaRecord{}, fmt.Errorf("git's list of what %s changes does not parse", short(head))
	}
	d := deltaRecord{Base: base, Records: true, Files: len(fields) / 2}
	if d.Files > MaxLandingRecords {
		return deltaRecord{}, fmt.Errorf("%s changes %d files, past the %d a landing's records may", short(head), d.Files, MaxLandingRecords)
	}
	for i := 0; i < len(fields); i += 2 {
		f, p := strings.Fields(fields[i]), fields[i+1]
		if len(f) < 4 {
			return deltaRecord{}, fmt.Errorf("git's list of what %s changes does not parse", short(head))
		}
		if !landingRecord(p) {
			return deltaRecord{}, fmt.Errorf("%s changes %s, outside the records a landing writes (%s)", short(head), clip(p), strings.Join(landingRecords, ", "))
		}
		for side, mode := range []string{strings.TrimPrefix(f[0], ":"), f[1]} {
			switch mode {
			case modeAbsent:
			case modeRegular:
				data, err := storedBytes(repoRoot, maxDeltaBlobBytes, "cat-file", "blob", f[2+side])
				if err != nil {
					return deltaRecord{}, fmt.Errorf("%s changes %s, which cannot be read whole: %w", short(head), clip(p), err)
				}
				if why := notPlainText(data); why != "" {
					return deltaRecord{}, fmt.Errorf("%s changes %s, which holds %s, and a record is plain text", short(head), clip(p), why)
				}
			default:
				return deltaRecord{}, fmt.Errorf("%s changes %s at mode %s, and a record is of mode %s", short(head), clip(p), mode, modeRegular)
			}
		}
	}
	return d, nil
}

// landingRecord reports whether p is a path the landing's records commit writes.
func landingRecord(p string) bool {
	if !strings.HasSuffix(p, deltaCoversKind) {
		return false
	}
	for _, dir := range landingRecords {
		if rest, ok := strings.CutPrefix(p, dir); ok {
			return !hostLoads(rest)
		}
	}
	return false
}
