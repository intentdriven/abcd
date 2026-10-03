package ask

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Safe returns a copy of a with every part sanitised for the screen
// (spc-2610030911534855, "Sanitising before drawing"). Every string in a
// question is runtime-read: a runner wrote it, a record supplied it, or a model
// service listed it. Each single-line part (the id, the chip, the ask, every
// option's and list choice's value, label and meaning, Now and Change later)
// passes termsafe.Sanitize, so a line break inside a label cannot forge a
// line; each material block passes termsafe.SanitizeBlock. An injected escape,
// a C1 control, a bidi override, a zero-width rune or a bare carriage return
// reaches the screen as a visible '?'. a itself is left unchanged.
func Safe(a question.Ask) question.Ask {
	out := question.Ask{Questions: make([]question.Question, len(a.Questions))}
	for i, q := range a.Questions {
		s := question.Question{
			ID:          termsafe.Sanitize(q.ID),
			Chip:        termsafe.Sanitize(q.Chip),
			Ask:         termsafe.Sanitize(q.Ask),
			Later:       safeOption(q.Later),
			Now:         termsafe.Sanitize(q.Now),
			ChangeLater: termsafe.Sanitize(q.ChangeLater),
		}
		if q.Material != nil {
			s.Material = make([]question.Block, len(q.Material))
			for j, b := range q.Material {
				s.Material[j] = question.Block{Kind: termsafe.Sanitize(b.Kind), Text: termsafe.SanitizeBlock(b.Text)}
				if b.Items != nil {
					s.Material[j].Items = make([]string, len(b.Items))
					for k, it := range b.Items {
						s.Material[j].Items[k] = termsafe.SanitizeBlock(it)
					}
				}
			}
		}
		s.Options = safeOptions(q.Options)
		if q.List != nil {
			s.List = &question.List{Choices: safeOptions(q.List.Choices)}
		}
		out.Questions[i] = s
	}
	return out
}

func safeOptions(in []question.Option) []question.Option {
	if in == nil {
		return nil
	}
	out := make([]question.Option, len(in))
	for i, o := range in {
		out[i] = safeOption(o)
	}
	return out
}

func safeOption(o question.Option) question.Option {
	return question.Option{
		Value:   termsafe.Sanitize(o.Value),
		Label:   termsafe.Sanitize(o.Label),
		Meaning: termsafe.Sanitize(o.Meaning),
	}
}

// Refusal is a question the structural check refuses: it is never drawn.
type Refusal struct {
	Findings []question.Finding
}

func (r *Refusal) Error() string {
	parts := make([]string, len(r.Findings))
	for i, f := range r.Findings {
		parts[i] = f.String()
	}
	return "the question cannot be drawn: " + strings.Join(parts, "; ")
}

// Prepare holds a to the structural check (question.Check) and returns it
// sanitised (Safe): the one door a question passes before it is drawn. Until
// the companion bundle's limits are run on this path, a question drawn in a
// plain Terminal is held to the structural check alone (spc-2610030911534855,
// Scope "Out"). A refused question is returned as a *Refusal naming every
// finding.
func Prepare(a question.Ask) (question.Ask, error) {
	if fs := question.Check(a); len(fs) > 0 {
		return question.Ask{}, &Refusal{Findings: fs}
	}
	return Safe(a), nil
}
