package question

import "github.com/intentdriven/abcd/internal/termsafe"

// Safe returns a copy of a with every part sanitised for the screen
// (spc-2610030911534855, "Sanitising before drawing"). Every string in a
// question is runtime-read: a runner wrote it, a record supplied it, or a model
// service listed it. Each single-line part (the id, the chip, the ask, every
// option's and list choice's value, label and meaning, Now and Change later)
// passes termsafe.Sanitize, so a line break inside a label cannot forge a
// line; each material block passes termsafe.SanitizeBlock. An injected escape,
// a C1 control, a bidi override, a zero-width rune or a bare carriage return
// reaches the screen as a visible '?'. a itself is left unchanged. The
// drawing (internal/surface/cli/ask) and the AI-written interviews' check
// (internal/core/interview) both sanitise through it, so what is measured is
// what is drawn.
func Safe(a Ask) Ask {
	out := Ask{Questions: make([]Question, len(a.Questions))}
	for i, q := range a.Questions {
		s := Question{
			ID:          termsafe.Sanitize(q.ID),
			Chip:        termsafe.Sanitize(q.Chip),
			Ask:         termsafe.Sanitize(q.Ask),
			Later:       safeOption(q.Later),
			Now:         termsafe.Sanitize(q.Now),
			ChangeLater: termsafe.Sanitize(q.ChangeLater),
		}
		if q.Material != nil {
			s.Material = make([]Block, len(q.Material))
			for j, b := range q.Material {
				s.Material[j] = Block{Kind: termsafe.Sanitize(b.Kind), Text: termsafe.SanitizeBlock(b.Text)}
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
			s.List = &List{Choices: safeOptions(q.List.Choices)}
		}
		out.Questions[i] = s
	}
	return out
}

func safeOptions(in []Option) []Option {
	if in == nil {
		return nil
	}
	out := make([]Option, len(in))
	for i, o := range in {
		out[i] = safeOption(o)
	}
	return out
}

func safeOption(o Option) Option {
	return Option{
		Value:   termsafe.Sanitize(o.Value),
		Label:   termsafe.Sanitize(o.Label),
		Meaning: termsafe.Sanitize(o.Meaning),
	}
}
