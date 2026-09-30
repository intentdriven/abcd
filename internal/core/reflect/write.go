package reflect

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// Answer is the person's answer to one asked section, and the answer to its
// follow-up question when the first was thin.
type Answer struct {
	Text     string `json:"answer"`
	FollowUp string `json:"follow_up,omitempty"`
}

// Answers are the four asked sections' answers, as a front door hands them to
// Write. The metrics section is computed, so it has no answer.
type Answers struct {
	WentWell     Answer `json:"went_well"`
	CouldImprove Answer `json:"could_improve"`
	Lessons      Answer `json:"lessons"`
	Decisions    Answer `json:"decisions"`
}

// Of returns the answer to an asked section.
func (a Answers) Of(s Section) Answer {
	switch s {
	case WentWell:
		return a.WentWell
	case CouldImprove:
		return a.CouldImprove
	case Lessons:
		return a.Lessons
	case Decisions:
		return a.Decisions
	}
	return Answer{}
}

// maxAnswersBytes caps an answers file: four sections of prose.
const maxAnswersBytes = 1 << 20

// ParseAnswers reads an answers document strictly: an unknown key or a repeated
// one is refused, because a mistyped section name would otherwise drop an
// answer without a word.
func ParseAnswers(data []byte) (Answers, error) {
	if len(data) > maxAnswersBytes {
		return Answers{}, fmt.Errorf("reflect: answers exceed %d bytes", maxAnswersBytes)
	}
	var a Answers
	if err := jsonstrict.Decode(data, &a); err != nil {
		return Answers{}, fmt.Errorf("reflect: answers: %w", err)
	}
	return a, nil
}

// WriteRequest is one retrospective write. ProceedDespiteUnshipped is the
// person's confirmation, asked for when intents targeted at the release are
// still unshipped (criterion 7). Now dates the retrospective.
type WriteRequest struct {
	Tag                     string
	Answers                 Answers
	ProceedDespiteUnshipped bool
	Now                     time.Time
}

// WriteResult names the file written and the seed it was written from.
type WriteResult struct {
	Path string `json:"path"`
	Seed Seed   `json:"seed"`
}

// Write writes the retrospective for req.Tag. It rebuilds the seed, so every
// refusal BuildSeed makes holds at the write too, then refuses, writing
// nothing: while unshipped targets are unconfirmed (UnshippedError), and while
// any asked answer is under the floor with its follow-up unanswered, or is
// blank after it (ThinAnswersError). Missing audit notes are not a refusal
// (criterion 2): the seed names them and the write goes ahead.
//
// The file is created exclusively inside the retrospectives tree, every level
// of which must be a real directory, so neither a second run nor a symlinked
// store can overwrite or escape.
func Write(root string, req WriteRequest) (WriteResult, error) {
	seed, err := BuildSeed(root, req.Tag)
	if err != nil {
		return WriteResult{}, err
	}
	if len(seed.Unshipped) > 0 && !req.ProceedDespiteUnshipped {
		return WriteResult{}, &UnshippedError{Tag: req.Tag, Intents: seed.Unshipped}
	}
	var thin []ThinAnswer
	for _, s := range AskedSections {
		a := req.Answers.Of(s)
		isThin, reason := underFloor(s, a.Text)
		if !isThin || strings.TrimSpace(a.FollowUp) != "" {
			continue
		}
		thin = append(thin, ThinAnswer{Section: s, Heading: s.Heading(), Reason: reason, Question: s.FollowUp()})
	}
	if len(thin) > 0 {
		return WriteResult{}, &ThinAnswersError{Thin: thin}
	}

	answers, err := redactAnswers(root, req.Answers)
	if err != nil {
		return WriteResult{}, err
	}
	doc := render(seed, answers, req.Now)
	dir := path.Dir(seed.Output)
	if err := fsutil.EnsureRealDirAll(root, dir, 0o755); err != nil {
		return WriteResult{}, fmt.Errorf("reflect: %w", err)
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return WriteResult{}, err
	}
	defer r.Close()
	if err := fsutil.CreateExclusiveIn(r, seed.Output, []byte(doc), 0o644); err != nil {
		if errors.Is(err, os.ErrExist) {
			return WriteResult{}, &ExistsError{Tag: req.Tag, Path: seed.Output}
		}
		return WriteResult{}, fmt.Errorf("reflect: %w", err)
	}
	return WriteResult{Path: seed.Output, Seed: seed}, nil
}

// redactAnswers passes every answer through the one canonical scanner before it
// reaches the committed record, the stance the decision store takes
// (internal/core/decide): it fails closed on an unavailable or degraded scanner,
// because a retrospective is durable committed prose and a broken detector must
// never let a pasted credential reach it under a false "clean" signal.
func redactAnswers(root string, a Answers) (Answers, error) {
	sc, err := scanner.New(root)
	if err != nil {
		return Answers{}, fmt.Errorf("reflect: refusing to write answers with an unavailable scanner: %w (nothing written)", err)
	}
	if unavail, reason := sc.Unavailable(); unavail {
		return Answers{}, fmt.Errorf("reflect: refusing to write answers with a degraded scanner: %s (nothing written)", reason)
	}
	clean := func(text string) string {
		findings := sc.ScanText(text, "reflect")
		if len(findings) == 0 {
			return text
		}
		out, _ := scanner.Redact(text, findings)
		return out
	}
	for _, p := range []*Answer{&a.WentWell, &a.CouldImprove, &a.Lessons, &a.Decisions} {
		p.Text = clean(p.Text)
		p.FollowUp = clean(p.FollowUp)
	}
	return a, nil
}

// render is the retrospective's text: frontmatter naming the release, the
// date, the intents and which audits fed the seed; a line linking the
// changelog section; a table linking each intent and its audit notes; the four
// answered sections; and the computed metrics. The audit notes are linked,
// never copied.
func render(s Seed, a Answers, now time.Time) string {
	var audited, unaudited, receipts, ids []string
	for _, it := range s.Intents {
		ids = append(ids, it.ID)
		if it.Audited {
			audited = append(audited, it.ID)
			if it.Receipt != "" {
				receipts = append(receipts, it.Receipt)
			}
		} else {
			unaudited = append(unaudited, it.ID)
		}
	}
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "release: %s\n", s.Tag)
	if s.Metrics.PreviousTag != "" {
		fmt.Fprintf(&b, "previous_release: %s\n", s.Metrics.PreviousTag)
	}
	fmt.Fprintf(&b, "date: %s\n", now.UTC().Format("2006-01-02"))
	fmt.Fprintf(&b, "intents: %s\n", flowList(ids))
	fmt.Fprintf(&b, "audited: %s\n", flowList(audited))
	fmt.Fprintf(&b, "unaudited: %s\n", flowList(unaudited))
	fmt.Fprintf(&b, "audit_receipts: %s\n", flowList(receipts))
	b.WriteString("---\n\n")
	fmt.Fprintf(&b, "# Retrospective for %s\n\n", s.Tag)

	if s.Changelog.Found {
		fmt.Fprintf(&b, "What %s shipped is in [its changelog section](%s#%s).", s.Tag, relFrom(s.Output, "CHANGELOG.md"), s.Changelog.Anchor)
	} else {
		fmt.Fprintf(&b, "The changelog carries no dated section for %s.", s.Tag)
	}
	b.WriteString(" Each intent's audit notes stay on the intent and are linked here, not repeated.\n\n")
	b.WriteString("| Intent | Impact | Audit notes |\n|---|---|---|\n")
	for _, it := range s.Intents {
		link := relFrom(s.Output, it.Path)
		notes := "none written"
		if it.Audited {
			notes = fmt.Sprintf("[audit notes](%s#audit-notes)", link)
		}
		impact := it.Impact
		if impact == "" {
			impact = "none declared"
		}
		fmt.Fprintf(&b, "| [%s](%s) %s | %s | %s |\n", it.ID, link, cell(it.Title), cell(impact), notes)
	}

	for _, sec := range AskedSections {
		ans := a.Of(sec)
		fmt.Fprintf(&b, "\n## %s\n\n%s\n", sec.Heading(), strings.TrimSpace(ans.Text))
		if f := strings.TrimSpace(ans.FollowUp); f != "" {
			fmt.Fprintf(&b, "\n%s\n", f)
		}
	}

	m := s.Metrics
	fmt.Fprintf(&b, "\n## %s\n\n", Metrics.Heading())
	fmt.Fprintf(&b, "- Intents shipped: %d (%d with audit notes, %d without)\n", m.IntentsShipped, m.Audited, m.Unaudited)
	fmt.Fprintf(&b, "- Audit verdicts: MET %d · MET_WITH_CONCERNS %d · NOT_MET %d · INCONCLUSIVE %d\n",
		m.Rollup.Met, m.Rollup.MetWithConcerns, m.Rollup.NotMet, m.Rollup.Inconclusive)
	fmt.Fprintf(&b, "- Gap audit: honoured %d · diverged %d · missing %d\n", m.Gaps.Honoured, m.Gaps.Diverged, m.Gaps.Missing)
	if m.PreviousTag != "" {
		fmt.Fprintf(&b, "- Released: %s on %s; the previous release, %s, on %s\n", s.Tag, orUnknown(m.TagDate), m.PreviousTag, orUnknown(m.PreviousTagDate))
	} else {
		fmt.Fprintf(&b, "- Released: %s on %s, the first release\n", s.Tag, orUnknown(m.TagDate))
	}
	return b.String()
}

func flowList(items []string) string { return "[" + strings.Join(items, ", ") + "]" }

func orUnknown(s string) string {
	if s == "" {
		return "an unrecorded date"
	}
	return s
}

// cell keeps a value inside one table cell.
func cell(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "|", `\|`), "\n", " ")
}

// relFrom is the relative link from the file at from to the file at to, both
// repo-relative and slash-separated.
func relFrom(from, to string) string {
	fromDir := strings.Split(path.Dir(from), "/")
	toParts := strings.Split(to, "/")
	i := 0
	for i < len(fromDir) && i < len(toParts)-1 && fromDir[i] == toParts[i] {
		i++
	}
	var parts []string
	for range fromDir[i:] {
		parts = append(parts, "..")
	}
	return strings.Join(append(parts, toParts[i:]...), "/")
}
