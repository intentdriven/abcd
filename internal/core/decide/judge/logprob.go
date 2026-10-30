package judge

// logprob.go is the decision adapter's second candidate
// (itd-2609221009495079 scope, "a small text model scoring options by
// log-probability ... behind the same interface"): a text model asked for a
// one-token answer, with the log-probabilities of its first token's
// alternatives, through the same API adapter client as the first candidate.
// The probability it gives each closed option's answer token, normalised over
// the options, is the answer's probability; the most probable option is its
// value.
//
// A route reaches it when the provider's block names the model in
// logprob_models (oracle.APIConfig.ScoresByLogprob); every other route is the
// first candidate, Adapter, as before. It is held to everything the first is:
// the call goes through oracle's Call, which admits the target against the
// provider's models list and oracle.denylist and resolves the key by name
// before any call, and the router's window check runs before it is asked.
// An answer whose first token is none of the options, or one with no
// log-probabilities at all, is no answer: an error, which the router records
// as a failed pair in shadow and falls back to the host on, never a guess.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/oracle"
)

// LogprobAdapter is the second candidate through the one interface.
type LogprobAdapter struct {
	Calls   Caller
	Creds   credential.Source
	Target  oracle.Target
	Options []openaiapi.Option
}

// logprobInstructions frame every question the second candidate is sent.
const logprobInstructions = "You answer one typed judgement with a single label and nothing else: " +
	"no words, no punctuation, no explanation. Read the question and the state, then reply with the one label " +
	"the question offers that is right."

// labelLetters label a choose question's options, one letter each: a letter
// is one token for any tokenizer, where a two-digit number is two for some.
const labelLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// labels are the one-token answer labels of q's choices, in the choices'
// order: a letter per option, each score on the scale as its digit, yes and
// no. A question whose choices cannot each be one such label (more options
// than letters, a score outside 0 to 9) is refused: no answer can be read
// from one token.
func labels(q Question) ([]string, error) {
	switch q.Kind {
	case KindChoose:
		if len(q.Options) > len(labelLetters) {
			return nil, fmt.Errorf("judge: the log-probability candidate labels at most %d options with a letter each, and %s has %d",
				len(labelLetters), q.Type, len(q.Options))
		}
		out := make([]string, len(q.Options))
		for i := range q.Options {
			out[i] = labelLetters[i : i+1]
		}
		return out, nil
	case KindScore:
		if q.Min < 0 || q.Max > 9 {
			return nil, fmt.Errorf("judge: the log-probability candidate reads a score as one digit, and %s's scale is %d to %d",
				q.Type, q.Min, q.Max)
		}
		return q.Choices(), nil
	default:
		return q.Choices(), nil
	}
}

// renderLabelled is the prompt the second candidate is asked: render's,
// with a choose question's options labelled by letter and the answer asked
// for as the label alone.
func renderLabelled(s State, q Question, ls []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Judgement: %s (%s)\n\nQuestion: %s\n\n", q.Type, q.Kind, strings.TrimSpace(q.Ask))
	switch q.Kind {
	case KindChoose:
		b.WriteString("Answer with the letter of exactly one of these options, and nothing else:\n")
		for i, o := range q.Options {
			fmt.Fprintf(&b, "%s. %s\n", ls[i], o)
		}
	case KindScore:
		fmt.Fprintf(&b, "Answer with one digit from %d to %d, and nothing else.\n", q.Min, q.Max)
	case KindYesNo:
		b.WriteString("Answer yes or no, and nothing else.\n")
	}
	fmt.Fprintf(&b, "\nState:\n\n%s\n", s.Text)
	return b.String()
}

// errNoLogprobs is an answer that carried no log-probabilities.
var errNoLogprobs = errors.New("judge: the model's answer carried no log-probabilities, so no option can be scored")

// Judge asks the question for a one-token answer with its alternatives'
// log-probabilities, and answers the most probable option with its
// probability normalised over the options.
func (a LogprobAdapter) Judge(ctx context.Context, s State, q Question) (Answer, error) {
	if err := q.Check(); err != nil {
		return Answer{}, err
	}
	if a.Calls == nil {
		return Answer{}, fmt.Errorf("judge: no adapter configured for %s", q.Type)
	}
	ls, err := labels(q)
	if err != nil {
		return Answer{}, err
	}
	var (
		top     []openaiapi.TokenLogprob
		carried bool
	)
	opts := append(slices.Clone(a.Options), openaiapi.WithTopLogprobs(openaiapi.MaxTopLogprobs, func(t []openaiapi.TokenLogprob, ok bool) {
		top, carried = t, ok
	}))
	brief := openaiapi.Brief{Instructions: logprobInstructions, Input: renderLabelled(s, q, ls)}
	req := oracle.CallRequest{Target: a.Target, Brief: brief,
		Settings: oracle.Settings{"max_tokens": json.RawMessage("1")},
		Contract: func(content []byte) error {
			if len(content) > maxAnswerBytes {
				return fmt.Errorf("judge: the answer is %d bytes, more than the %d a one-token answer takes", len(content), maxAnswerBytes)
			}
			return nil
		}}
	_, rec, err := a.Calls.Call(ctx, a.Creds, req, opts...)
	if err != nil {
		return Answer{}, err
	}
	if !carried {
		return Answer{}, errNoLogprobs
	}
	got, err := fromLogprobs(q, ls, top)
	if err != nil {
		return Answer{}, err
	}
	got.Model = rec.ModelReported
	return got, nil
}

// fromLogprobs reads the answer from the first token's alternatives: each
// alternative whose token, trimmed of space and read without regard to case,
// is an option's label adds its probability to that option's; the options'
// probabilities are normalised over the options, and the most probable is the
// answer (the first in the question's order on a tie). No alternative naming
// an option is no answer.
func fromLogprobs(q Question, ls []string, top []openaiapi.TokenLogprob) (Answer, error) {
	choices := q.Choices()
	mass := make([]float64, len(ls))
	total := 0.0
	for _, t := range top {
		tok := strings.TrimSpace(t.Token)
		for i, l := range ls {
			if strings.EqualFold(tok, l) {
				p := math.Exp(t.Logprob)
				mass[i] += p
				total += p
				break
			}
		}
	}
	if total <= 0 || math.IsNaN(total) || math.IsInf(total, 0) {
		seen := make([]string, 0, len(top))
		for _, t := range top {
			seen = append(seen, strconv.Quote(clip(t.Token)))
		}
		return Answer{}, fmt.Errorf("judge: none of the %d alternatives at the answer's first token (%s) is one of the labels %s, so no option is scored",
			len(top), detailList(seen), strings.Join(ls, ", "))
	}
	best := 0
	for i := range mass {
		if mass[i] > mass[best] {
			best = i
		}
	}
	return Answer{Value: choices[best], Probability: mass[best] / total}, nil
}

// detailList joins the alternatives an error quotes, bounded.
func detailList(s []string) string {
	const n = 8
	if len(s) == 0 {
		return "none"
	}
	if len(s) > n {
		return strings.Join(s[:n], ", ") + ", …"
	}
	return strings.Join(s, ", ")
}
