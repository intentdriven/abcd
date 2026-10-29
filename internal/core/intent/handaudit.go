package intent

import (
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/condition"
	"github.com/intentdriven/abcd/internal/core/mdrecord"
)

// handaudit.go — the one rule for when an Audit Notes section that carries no
// review marker counts as an audit (iss-2610241431221889, itd-24 criterion 2).
//
// The product thinker ruled on 29 October 2026 that such a section is audited
// only when it reads as a real audit, one that names the acceptance criteria
// and gives a verdict on them; a ruling's amendment line, or any other note,
// is not. The retrospective seed (reflect) and the list of reviews owed
// (ReviewOf) both read the section through HandAudited, so the two cannot
// disagree about a record again.

var (
	// auditSectionCloseRe is a heading that ends the Audit Notes section for this
	// rule: one of level one or two. A sub-heading is part of the section, as in
	// the one hand-written audit the record holds (itd-3, shipped on 2026-07-11),
	// whose per-criterion verdicts sit under `### Per-criterion verdicts`.
	auditSectionCloseRe = regexp.MustCompile(`^#{1,2}\s`)
	// auditHeadingLineRe is any ATX heading line.
	auditHeadingLineRe = regexp.MustCompile(`^#{1,6}\s`)
	// auditItemOpenRe opens a list item at any indent, bulleted or numbered.
	auditItemOpenRe = regexp.MustCompile(`^\s*(?:[-*+]|\d{1,3}[.)])\s+\S`)
	// auditNumberedItemRe opens a numbered list item: by its position it names the
	// criterion of that number, as itd-3's verdict list does.
	auditNumberedItemRe = regexp.MustCompile(`^\s*\d{1,3}[.)]\s+\S`)
	// auditCriterionRe names a criterion by its id or its number: `ac-2`, `AC 2`,
	// `ac2`, `criterion 2`, `criteria 1 and 2`. The character before `ac` must
	// not be a word character or a hyphen, so a receipt id such as
	// `rcp-ac12…` is not read as a criterion.
	auditCriterionRe = regexp.MustCompile(`(?i)(?:^|[^\w-])ac[- ]?\d+\b|\bcriteri(?:on|a)\s+\d+\b`)
	// auditVerdictRe is one of the auditor's four verdicts, in capitals as the
	// auditor and the hand-written audit both write them. `NOT MET` and
	// `MET WITH CONCERNS` read as verdicts through their `MET`.
	auditVerdictRe = regexp.MustCompile(`\b(?:MET_WITH_CONCERNS|NOT_MET|INCONCLUSIVE|MET)\b`)
	// auditInlineCommentRe is an HTML comment opened and closed on one line, which
	// the section mask leaves live.
	auditInlineCommentRe = regexp.MustCompile(`<!--.*?-->`)
)

// HandAudited reports whether the record's `## Audit Notes` section reads as a
// hand-written audit. It is the rule for a section with NO review marker; a
// record that carries one is judged by its marker (ReviewStateOf), and an OWED
// marker is never an audit whatever else the section holds.
//
// The section runs from its heading to the next heading of level one or two,
// so its sub-headings are part of it. Its live lines are read as markdown
// reads them (a fenced block, an HTML comment and an inline code span are not
// prose) and split into blocks: a paragraph, a list item with its
// continuation lines, or a heading line. The section is a hand-written audit
// when at least one block does both of these:
//
//   - names an acceptance criterion: by id (`ac-2`, `AC 2`), by number
//     (`criterion 2`, `criteria 1 and 2`), or by opening a numbered list item,
//     whose number names the criterion it judges;
//   - gives a verdict: one of the auditor's four, MET, MET_WITH_CONCERNS,
//     NOT_MET or INCONCLUSIVE, written in capitals.
//
// The rule is deliberately narrow. A placeholder, a ruling's amendment line, a
// note that a criterion "is met" in running prose, and a paragraph that says
// every criterion holds without naming one each name no criterion with a
// verdict, so none of them is an audit, and the intent stays owed its review.
func HandAudited(content string) bool {
	lines, mask, start, _, ok := condition.AuditNotes(content)
	if !ok {
		return false
	}
	var block strings.Builder
	judged := false
	flush := func() {
		if blockJudges(block.String()) {
			judged = true
		}
		block.Reset()
	}
	for i := start; i < len(lines) && !judged; i++ {
		if mask[i] != 0 {
			flush()
			continue
		}
		raw := strings.TrimRight(lines[i], "\r")
		if auditSectionCloseRe.MatchString(raw) {
			break
		}
		prose := proseOf(raw)
		switch {
		case strings.TrimSpace(prose) == "":
			flush()
		case auditHeadingLineRe.MatchString(raw):
			flush()
			block.WriteString(prose)
			flush()
		case auditItemOpenRe.MatchString(raw):
			flush()
			block.WriteString(prose)
		default:
			block.WriteString("\n" + prose)
		}
	}
	flush()
	return judged
}

// proseOf is one line with what is not prose taken out: each HTML comment
// closed on the line, and each inline code span, since a verdict word quoted
// as code is vocabulary under discussion, not a verdict given.
func proseOf(line string) string {
	line = auditInlineCommentRe.ReplaceAllString(line, "")
	spans := mdrecord.CodeSpanRanges(line)
	for k := len(spans) - 1; k >= 0; k-- {
		line = line[:spans[k][0]] + line[spans[k][1]:]
	}
	return line
}

// blockJudges reports whether one block names a criterion and gives a verdict.
func blockJudges(block string) bool {
	names := auditCriterionRe.MatchString(block) || auditNumberedItemRe.MatchString(block)
	return names && auditVerdictRe.MatchString(block)
}
