package reflect

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/mdrecord"
)

var (
	auditHeadingRe = regexp.MustCompile(`^##\s+Audit Notes\s*$`)
	receiptRe      = regexp.MustCompile(`abcd-review:\s+INGESTED\s+receipt=(rcp-[0-9a-f]+)`)
	gapBucketRe    = regexp.MustCompile(`^- (honoured|diverged|missing):`)
	gapItemRe      = regexp.MustCompile(`^  - \S`)
)

// readAuditNotes reads the record's `## Audit Notes` section into it: whether
// it carries audit notes at all, the ingested receipt, and the counts. The
// section runs to the next heading of any depth, the one notion of a section
// every reader of these records shares (mdrecord.SectionLineRange), so a
// sub-heading parked under the placeholder is not an audit.
//
// An ingested review is recognised by its `Acceptance rollup:` line, read from
// the prose only (a fenced or commented rollup is an example, not a verdict).
// A section with other prose — a hand-written audit from before the reviewer
// existed — counts as audited with no counts. The placeholder the intent
// template carries and an owed review's marker are not audit notes.
func readAuditNotes(text string, it *SeedIntent) {
	_, body := frontmatter.Split(text)
	lines := strings.Split(body, "\n")
	mask := mdrecord.Mask(lines)
	start, end, ok := mdrecord.SectionLineRangeIn(lines, mask, auditHeadingRe)
	if !ok {
		return
	}
	prose := false
	bucket := ""
	for i := start; i < end; i++ {
		raw := strings.TrimRight(lines[i], "\r")
		if m := receiptRe.FindStringSubmatch(raw); m != nil && it.Receipt == "" {
			it.Receipt = m[1]
		}
		if mask[i] != 0 {
			continue
		}
		t := strings.TrimSpace(raw)
		// mdrecord masks a comment SPAN; a comment closed on its own line, the
		// review markers' shape, is skipped here.
		oneLineComment := strings.HasPrefix(t, "<!--") && strings.HasSuffix(t, "-->")
		if t == "" || oneLineComment || strings.HasPrefix(t, "_Empty.") || strings.HasPrefix(t, "Fidelity review OWED") {
			continue
		}
		prose = true
		if _, after, found := strings.Cut(t, "Acceptance rollup:"); found {
			it.Rollup = parseRollup(after)
			it.Audited = true
			continue
		}
		if m := gapBucketRe.FindStringSubmatch(raw); m != nil {
			bucket = m[1]
			continue
		}
		if !strings.HasPrefix(raw, " ") {
			bucket = ""
			continue
		}
		if bucket != "" && gapItemRe.MatchString(raw) {
			switch bucket {
			case "honoured":
				it.Gaps.Honoured++
			case "diverged":
				it.Gaps.Diverged++
			case "missing":
				it.Gaps.Missing++
			}
		}
	}
	if prose {
		it.Audited = true
	}
}

// parseRollup reads `MET n · MET_WITH_CONCERNS n · NOT_MET n · INCONCLUSIVE n`.
// A count that does not read as a non-negative number is taken as zero.
func parseRollup(s string) AuditRollup {
	var r AuditRollup
	for _, part := range strings.Split(s, "·") {
		f := strings.Fields(part)
		if len(f) != 2 {
			continue
		}
		n, err := strconv.Atoi(f[1])
		if err != nil || n < 0 {
			continue
		}
		switch f[0] {
		case "MET":
			r.Met = n
		case "MET_WITH_CONCERNS":
			r.MetWithConcerns = n
		case "NOT_MET":
			r.NotMet = n
		case "INCONCLUSIVE":
			r.Inconclusive = n
		}
	}
	return r
}
