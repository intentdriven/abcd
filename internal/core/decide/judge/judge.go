// Package judge is the decision adapter's one interface (itd-2609221009495079,
// spc-2609221011151661 scope 1): a typed judgement, (state, question) →
// (value, probability), with three question kinds — choose one of N, score on
// a scale, yes or no. The host's own judgement implements it (Host, at
// probability 1), and so does the decision model reached through the
// OpenAI-compatible API adapter (Adapter), so the two are compared like for
// like and a judgement type is switched between them by configuration alone
// (router.go), never by its caller.
//
// It sits apart from internal/core/decide, the ADR verb's package: the name is
// shared, the concern is not.
//
// Like the rest of the core it never prints. It writes only through the
// recorder its Router is given, which is the lab's pairs file in the
// machine-scoped store (internal/core/lab), never the repository.
package judge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Kind is a question's kind.
type Kind string

// The three question kinds.
const (
	// KindChoose asks for one of the question's options.
	KindChoose Kind = "choose"
	// KindScore asks for an integer on the question's scale, Min to Max.
	KindScore Kind = "score"
	// KindYesNo asks yes or no.
	KindYesNo Kind = "yes-no"
)

// Type is a judgement type: one judgement-shaped step abcd may route through
// the adapter. The registry is closed: a type outside it is refused.
type Type string

// The judgement types the intent's scope lists. Embark's lesson ranking is
// one: it is to run through the adapter in place of the term-overlap ordering
// the embark verb uses today (iss-2610262033055327), and like every other
// type it is decided by the host until its step calls the router.
const (
	RemedyRead           Type = "remedy-read"
	DuplicateMatch       Type = "duplicate-match"
	UnmeetableCriteria   Type = "unmeetable-criteria"
	EligibilityResidual  Type = "eligibility-residual"
	SourceClassification Type = "source-classification"
	SourceSelection      Type = "source-selection"
	MemoryCheck          Type = "memory-check"
	LedgerCategory       Type = "ledger-category"
	LedgerSeverity       Type = "ledger-severity"
	PrepassOverlap       Type = "prepass-overlap"
	PresentTense         Type = "present-tense-advisory"
	EmbarkLessonRanking  Type = "embark-lesson-ranking"
)

var registry = []Type{RemedyRead, DuplicateMatch, UnmeetableCriteria, EligibilityResidual, SourceClassification,
	SourceSelection, MemoryCheck, LedgerCategory, LedgerSeverity, PrepassOverlap, PresentTense, EmbarkLessonRanking}

// types returns the registry, in its order.
func types() []Type { return slices.Clone(registry) }

// known reports whether name is a type in the registry.
func known(name string) bool { return slices.Contains(registry, Type(name)) }

// Bounds on a question.
const (
	maxOptions  = 64
	maxOption   = 200
	maxAsk      = 2000
	maxScaleLen = 100
)

// Question is one typed question.
type Question struct {
	Type Type
	Kind Kind
	// Ask is the question in words.
	Ask string
	// Options are a choose question's options.
	Options []string
	// Min and Max are a score question's scale, inclusive.
	Min, Max int
}

// Choices are the answers the question admits, as an answer spells them.
func (q Question) Choices() []string {
	switch q.Kind {
	case KindYesNo:
		return []string{"yes", "no"}
	case KindScore:
		out := make([]string, 0, q.Max-q.Min+1)
		for v := q.Min; v <= q.Max; v++ {
			out = append(out, strconv.Itoa(v))
		}
		return out
	default:
		return slices.Clone(q.Options)
	}
}

// Check refuses a question that does not hold together: a type outside the
// registry, an unknown kind, no words, or options or a scale out of shape.
func (q Question) Check() error {
	if !known(string(q.Type)) {
		return fmt.Errorf("judge: %q is not a judgement type (the registry is %s)", clip(string(q.Type)), typeNames())
	}
	if strings.TrimSpace(q.Ask) == "" || len(q.Ask) > maxAsk {
		return fmt.Errorf("judge: a %s question needs its words (at most %d bytes)", q.Type, maxAsk)
	}
	switch q.Kind {
	case KindYesNo:
	case KindScore:
		if q.Max <= q.Min || q.Max-q.Min >= maxScaleLen {
			return fmt.Errorf("judge: a %s score question needs a scale of 2 to %d points, not %d to %d", q.Type, maxScaleLen, q.Min, q.Max)
		}
	case KindChoose:
		if len(q.Options) < 1 || len(q.Options) > maxOptions {
			return fmt.Errorf("judge: a %s choose question needs 1 to %d options, not %d", q.Type, maxOptions, len(q.Options))
		}
		seen := map[string]bool{}
		for _, o := range q.Options {
			k := strings.ToLower(strings.TrimSpace(o))
			if k == "" || len(o) > maxOption || strings.ContainsAny(o, "\n\r") || seen[k] {
				return fmt.Errorf("judge: a %s choose question's options are distinct one-line words of at most %d bytes", q.Type, maxOption)
			}
			seen[k] = true
		}
	default:
		return fmt.Errorf("judge: %q is not a question kind (choose, score, yes-no)", clip(string(q.Kind)))
	}
	return nil
}

// match returns the choice text names, matched without regard to case or
// surrounding space, and whether it is one.
func (q Question) match(text string) (string, bool) {
	t := strings.TrimSpace(text)
	for _, c := range q.Choices() {
		if strings.EqualFold(c, t) {
			return c, true
		}
	}
	return "", false
}

// State is what a judgement is made over, after retrieval: the text both
// implementations are shown.
type State struct {
	Text string
}

// SHA256 is the state's hash, the identity a pair records instead of the text.
func (s State) SHA256() string {
	sum := sha256.Sum256([]byte(s.Text))
	return hex.EncodeToString(sum[:])
}

// Answer is a judgement's answer: one of the question's choices, the
// probability the answering side gives it, and, from the adapter, the model
// the provider reported.
type Answer struct {
	Value       string
	Probability float64
	Model       string
}

// Judge is the one interface: the host and the adapter both implement it.
type Judge interface {
	Judge(ctx context.Context, s State, q Question) (Answer, error)
}

// render is the prompt a judgement is asked as, the same for the host and the
// adapter.
func render(s State, q Question) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Judgement: %s (%s)\n\nQuestion: %s\n\n", q.Type, q.Kind, strings.TrimSpace(q.Ask))
	switch q.Kind {
	case KindChoose:
		b.WriteString("Answer with exactly one of these options:\n")
		for _, o := range q.Options {
			fmt.Fprintf(&b, "- %s\n", o)
		}
	case KindScore:
		fmt.Fprintf(&b, "Answer with one whole number from %d to %d.\n", q.Min, q.Max)
	case KindYesNo:
		b.WriteString("Answer yes or no.\n")
	}
	fmt.Fprintf(&b, "\nState:\n\n%s\n", s.Text)
	return b.String()
}

// tokens estimates the tokens text takes, high rather than low: three bytes a
// token, where English prose runs nearer four, so a state that fits by this
// count fits the model's window.
func tokens(text string) int { return (len(text) + 2) / 3 }

func typeNames() string {
	out := make([]string, len(registry))
	for i, t := range registry {
		out[i] = string(t)
	}
	return strings.Join(out, ", ")
}

// clip bounds a caller-supplied string for a message.
func clip(s string) string {
	const n = 64
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
