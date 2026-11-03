package interview

// redraft.go is how an AI-written interview treats a receipt abcd's check
// refuses (iss-2611011402496418; the product thinker's rulings
// dcn-2611011402510346 and dcn-2611012123240576). The interview never stops
// on such a receipt and the person never sees it: the question, or the done
// outcome, is handed back to the interviewer with the refusal in its next
// brief, up to the repository's interview.redrafts bound in a row. At the
// bound a question is shown as it stands, repaired mechanically where the
// refusing rules allow, and a done outcome ends the interview with a plain
// message naming what happened and what to run next. The rows limit refuses
// nothing on its own (question.RefusingFinding), as on the host's route: an
// over-tall question is drawn, and the next brief tells the role to keep the
// next within it.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// askCheck is an ask a role wrote, held to what abcd draws: safe is the ask
// sanitised (question.Safe), so what is measured is what is drawn; refusing
// are the findings that refuse it, and warned those that do not (the rows
// limit).
type askCheck struct {
	safe     question.Ask
	refusing []question.Finding
	warned   []question.Finding
}

// checkAsk sanitises a, then holds it to the structural check and the asking
// limits.
func checkAsk(a question.Ask, l question.Limits, verbs []string) askCheck {
	c := askCheck{safe: question.Safe(a)}
	fs := question.Check(c.safe)
	fs = append(fs, question.CheckLimits(c.safe.Fields(), l, question.Addressee{Verbs: verbs})...)
	for _, f := range fs {
		if question.RefusingFinding(f) {
			c.refusing = append(c.refusing, f)
		} else {
			c.warned = append(c.warned, f)
		}
	}
	return c
}

// err is every refusing finding as one error, or nil when the ask is drawn.
func (c askCheck) err() error {
	if len(c.refusing) == 0 {
		return nil
	}
	return fmt.Errorf("the question breaks %d rule(s): %s", len(c.refusing), findingList(c.refusing))
}

// all is every finding, the refusing ones first.
func (c askCheck) all() []question.Finding {
	return append(slices.Clone(c.refusing), c.warned...)
}

func findingList(fs []question.Finding) string {
	parts := make([]string, len(fs))
	for i, f := range fs {
		parts[i] = f.String()
	}
	return strings.Join(parts, "; ")
}

// RedraftsSpentError is the end of an interview whose interviewer, at the
// redrafts bound, still wrote a receipt abcd's check refuses that abcd cannot
// show: a done outcome, or a question no mechanical repair makes drawable. Its
// message is for the person, in plain words: what happened and what to run
// next, never the check's finding, which the turn's brief and receipt keep.
type RedraftsSpentError struct {
	// Interview is the interview's name, Command the verb and target to run
	// again.
	Interview, Command string
	// Outcome is true for a done outcome, false for a question.
	Outcome bool
	// Redrafts is the bound reached; Answered the answers given before it.
	Redrafts, Answered int
}

func (e *RedraftsSpentError) Error() string {
	even := ""
	if e.Redrafts > 0 {
		even = fmt.Sprintf(", even after %d more %s", e.Redrafts, plural(e.Redrafts, "attempt", "attempts"))
	}
	what := "the AI writing its questions could not write its next question in a form abcd can show" + even + "."
	if e.Outcome {
		what = "abcd could not accept the closing summary the AI writing its questions wrote" + even + ", so nothing was written from it."
	}
	kept := "No answer had been given yet."
	if e.Answered > 0 {
		kept = fmt.Sprintf("The %d %s you gave %s kept.", e.Answered, plural(e.Answered, "answer", "answers"), plural(e.Answered, "is", "are"))
	}
	return fmt.Sprintf("the %s interview has ended: %s %s Run `%s` again to start the interview afresh.", e.Interview, what, kept, e.Command)
}

// spent is the interview's end at the redrafts bound, after redrafts
// redrafts and answered answers.
func (w *Written) spent(outcome bool, redrafts, answered int) *RedraftsSpentError {
	return &RedraftsSpentError{Interview: w.Name, Command: w.Verb + " " + termsafe.Sanitize(w.Target),
		Outcome: outcome, Redrafts: redrafts, Answered: answered}
}

// carried is what a turn's brief tells the role of its last receipt.
type carried struct {
	// bound is the redrafts bound in force.
	bound int
	// again is the writer's refusal of the last outcome, kept until a
	// question is drawn.
	again string
	// refused is why abcd's check refused the last receipt, handed back as
	// redraft number redraft.
	refused string
	redraft int
	// warned is what the last question drawn broke without being refused:
	// the rows limit.
	warned string
	// shown marks the last question shown at the bound after redraft
	// redrafts, repaired; left is what it still broke as shown.
	shown bool
	left  string
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// repair mends a refused ask where the refusing rules allow it mechanically,
// for showing at the redrafts bound: the bold markers are removed from every
// part (rule 7), the recommendation marks from every label (rule 8), every
// label is cut to l.LabelWords words (rule 4), an option labelled as the way
// to decide later is dropped where another answer stays (rule 6), and the
// decide-later option is given a value no answer has, a decide-later label
// and a meaning where it lacks them (rule 6 and the structural check). It
// rewrites nothing else, and its result passes question.Safe again, so what
// is shown stays sanitised. a is left unchanged.
func repair(a question.Ask, l question.Limits) question.Ask {
	out := question.Ask{Questions: make([]question.Question, len(a.Questions))}
	for i, q := range a.Questions {
		q.Chip, q.Ask, q.Now, q.ChangeLater, q.Typed = unbold(q.Chip), unbold(q.Ask), unbold(q.Now), unbold(q.ChangeLater), unbold(q.Typed)
		if q.Material != nil {
			blocks := make([]question.Block, len(q.Material))
			for j, b := range q.Material {
				b.Text = unbold(b.Text)
				if b.Items != nil {
					items := make([]string, len(b.Items))
					for k, it := range b.Items {
						items[k] = unbold(it)
					}
					b.Items = items
				}
				blocks[j] = b
			}
			q.Material = blocks
		}
		q.Options = repairOptions(q.Options, l)
		if q.List != nil {
			q.List = &question.List{Choices: repairOptions(q.List.Choices, l)}
		}
		q.Later = repairLater(q, l)
		out.Questions[i] = q
	}
	return question.Safe(out)
}

// repairOptions is opts with each label and meaning mended, an option
// labelled as the way to decide later dropped where another stays.
func repairOptions(opts []question.Option, l question.Limits) []question.Option {
	if opts == nil {
		return nil
	}
	out := make([]question.Option, 0, len(opts))
	for _, o := range opts {
		o.Label = repairLabel(o.Label, o.Value, l)
		o.Meaning = unbold(o.Meaning)
		out = append(out, o)
	}
	kept := slices.DeleteFunc(slices.Clone(out), func(o question.Option) bool { return isLaterLabel(o.Label, l) })
	if len(kept) > 0 {
		return kept
	}
	return out
}

// laterMeaning is the decide-later option's meaning where the role gave it
// none.
const laterMeaning = "Nothing is decided now. The question is asked again later."

// repairLater is q's decide-later option mended: a value none of q's answers
// has, a decide-later label, and a meaning.
func repairLater(q question.Question, l question.Limits) question.Option {
	o := q.Later
	o.Label = repairLabel(o.Label, "", l)
	if !isLaterLabel(o.Label, l) && len(l.LaterLabels) > 0 {
		o.Label = l.LaterLabels[0]
	}
	if o.Meaning = unbold(o.Meaning); strings.TrimSpace(o.Meaning) == "" {
		o.Meaning = laterMeaning
	}
	taken := map[string]bool{}
	for _, x := range append(slices.Clone(q.Options), listChoices(q)...) {
		taken[strings.TrimSpace(x.Value)] = true
	}
	if v := strings.TrimSpace(o.Value); v == "" || taken[v] {
		o.Value = "later"
		for n := 2; taken[o.Value]; n++ {
			o.Value = fmt.Sprintf("later-%d", n)
		}
	}
	return o
}

// repairLabel is a label with its bold markers and recommendation marks
// removed and cut to l.LabelWords words; a label left with no words is the
// answer's value, when it has one.
func repairLabel(label, value string, l question.Limits) string {
	s := cutRecommended(unbold(label))
	s = strings.TrimFunc(s, func(r rune) bool {
		switch r {
		case ' ', '\t', '️', '*', '★', '☆', '⭐':
			return true
		}
		return false
	})
	words := strings.Fields(s)
	if l.LabelWords > 0 && len(words) > l.LabelWords {
		words = words[:l.LabelWords]
	}
	if len(words) == 0 {
		return termsafe.Sanitize(value)
	}
	return strings.Join(words, " ")
}

// recommendedMark is the mark rule 8 finds in a label, in any letter case.
const recommendedMark = "(recommended)"

// cutRecommended is s with every recommendedMark removed, matched without
// regard to ASCII letter case. The match is made on s itself, never on a
// lowercased copy: strings.ToLower changes the length in bytes of some runes
// (U+023A grows, U+0130 shrinks), so an index into the copy is not an index
// into s. The mark is ASCII and no byte of a multi-byte UTF-8 sequence is,
// and no rune but an ASCII one lowercases to one of the mark's letters, so
// this removes exactly what the check's strings.ToLower search detects.
// A removal that joins two halves into a new mark is removed in turn.
func cutRecommended(s string) string {
	for {
		i := indexASCIIFold(s, recommendedMark)
		if i < 0 {
			return s
		}
		s = s[:i] + s[i+len(recommendedMark):]
	}
}

// indexASCIIFold is the byte index in s of the first match of the lowercase
// ASCII pattern p, ASCII letters compared without case, or -1.
func indexASCIIFold(s, p string) int {
	for i := 0; i+len(p) <= len(s); i++ {
		j := 0
		for ; j < len(p); j++ {
			c := s[i+j]
			if 'A' <= c && c <= 'Z' {
				c += 'a' - 'A'
			}
			if c != p[j] {
				break
			}
		}
		if j == len(p) {
			return i
		}
	}
	return -1
}

// unbold is s with the markers rule 7 refuses removed.
func unbold(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "**", ""), "__", "")
}

// isLaterLabel reports whether label is one of l's decide-later labels, as
// rule 6 reads it.
func isLaterLabel(label string, l question.Limits) bool {
	label = strings.TrimSpace(label)
	return slices.ContainsFunc(l.LaterLabels, func(ll string) bool { return strings.EqualFold(label, ll) })
}
