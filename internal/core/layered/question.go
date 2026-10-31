package layered

import (
	"errors"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/question"
)

// The question namespace (itd-2610301914350541): a repository sets how many
// rows one of abcd's questions may take, how many words an option's label may
// run to, and how many sentences its description may say, in
// .abcd/config.json, and a person's own ~/.abcd.noindex/config.json may set
// each lower, never higher. Every other limit stays the binary's
// (question.Default).

// questionNamespace is the namespace QuestionLimits claims; the keys it takes
// are question.Bounds' keys below it.
const questionNamespace = "question"

// LimitSource is where the figure in force for one key came from: the
// bundled default, the repository's file, or the machine's file where a
// personal value tightened it.
type LimitSource struct {
	Key    string
	Layer  Layer
	Origin string
}

// QuestionLimitsValue is the resolved limits: Value carries them, with the
// Layer and Origin of the loosest layer that set any key (the repository's
// over the machine's, since the machine may only tighten; bundled where no
// file set one), and Sources the provenance of each key, in question.Bounds'
// order, which is what the check's limits_source reports key by key.
type QuestionLimitsValue struct {
	Value[question.Limits]
	Sources []LimitSource
}

// SetByFile reports whether any key's figure came from a file.
func (v QuestionLimitsValue) SetByFile() bool {
	for _, s := range v.Sources {
		if s.Layer != Bundled {
			return true
		}
	}
	return false
}

// KeyRefusedError is a question value read and refused: outside its bounds,
// not a whole number, or, for a personal value, looser than the limit
// otherwise in force (LimitOrigin set). Nothing falls through to a lower
// layer or to the default (itd-2610301914350541 decision 7).
type KeyRefusedError struct {
	Key string
	// Value is the value as the file holds it, compacted and bounded.
	Value  string
	Bounds question.Bound
	// Layer and Origin name the file that holds Value.
	Layer  Layer
	Origin string
	// NotInteger marks a value that does not decode as a whole number.
	NotInteger bool
	// Limit and LimitOrigin are set for a personal value that loosens: the
	// limit otherwise in force and the file that set it ("bundled" for the
	// default).
	Limit       int
	LimitOrigin string
}

// Loosens reports whether the refusal is a personal value above the limit in
// force, rather than a value outside its bounds.
func (e *KeyRefusedError) Loosens() bool { return e.LimitOrigin != "" }

func (e *KeyRefusedError) Error() string {
	head := fmt.Sprintf("%s (%s layer): %s is %s: ", e.Origin, e.Layer, e.Key, e.Value)
	switch {
	case e.NotInteger:
		return head + "want a whole number from " + e.Bounds.String()
	case e.Loosens():
		return head + fmt.Sprintf("looser than %d (%s); a personal value may only tighten", e.Limit, e.LimitOrigin)
	}
	return head + "outside " + e.Bounds.String()
}

// QuestionLimits resolves the three question keys over question.Default. It
// claims the question namespace, so any other key under it (question.options,
// question.columns, a misspelt one) is refused by Claim naming the key and the
// file. Each key's limit otherwise in force is the repository's value where
// .abcd/config.json sets it, else the default; a machine value at or below
// that limit replaces it, and one above it is refused naming both figures and
// their files (itd-2610301914350541 decision 5). Every value found, in either
// file, is decoded strictly as an int and held to its question.Bound before
// any comparison, so a personal value below the floor is refused naming the
// bounds, never the limit.
//
// It returns one of three states: the defaults (no key set, Layer Bundled),
// the limits in force with the file that set each, or an error. An error is
// an *UnreadError where a layer could not be read, returned with the defaults,
// since the gate fails open and loud on it and the question runs at them; and
// a refusal otherwise (*KeyRefusedError, or Claim's unknown key), returned
// with the zero value, since the gate fails closed and nothing falls through.
func QuestionLimits(r Roots) (QuestionLimitsValue, error) {
	s, err := Load(Config, r)
	if err != nil {
		var u *UnreadError
		if errors.As(err, &u) {
			return defaultQuestionLimits(), err
		}
		return QuestionLimitsValue{}, err
	}
	keys := make([]string, len(question.Bounds))
	for i, b := range question.Bounds {
		keys[i] = strings.TrimPrefix(b.Key, questionNamespace+".")
	}
	if err := s.Claim(questionNamespace, keys...); err != nil {
		return QuestionLimitsValue{}, err
	}
	def := question.Default.Configurable()
	var got [3]int
	sources := make([]LimitSource, len(question.Bounds))
	for i, b := range question.Bounds {
		found, err := s.Lookup(b.Key)
		if err != nil {
			return QuestionLimitsValue{}, err
		}
		limit, in := def[i], LimitSource{Key: b.Key, Layer: Bundled, Origin: "bundled"}
		var repo, machine *Found
		for j := range found {
			switch found[j].Layer {
			case Repo:
				repo = &found[j]
			case Machine:
				machine = &found[j]
			}
		}
		if repo != nil {
			n, err := boundedInt(*repo, b)
			if err != nil {
				return QuestionLimitsValue{}, err
			}
			limit, in = n, LimitSource{Key: b.Key, Layer: Repo, Origin: repo.Origin}
		}
		if machine != nil {
			n, err := boundedInt(*machine, b)
			if err != nil {
				return QuestionLimitsValue{}, err
			}
			if n > limit {
				return QuestionLimitsValue{}, &KeyRefusedError{Key: b.Key, Value: compact(machine.Raw), Bounds: b,
					Layer: machine.Layer, Origin: machine.Origin, Limit: limit, LimitOrigin: in.Origin}
			}
			limit, in = n, LimitSource{Key: b.Key, Layer: Machine, Origin: machine.Origin}
		}
		got[i], sources[i] = limit, in
	}
	v := defaultQuestionLimits()
	v.Sources = sources
	for _, src := range sources {
		if src.Layer > v.Layer {
			v.Layer, v.Origin = src.Layer, src.Origin
		}
	}
	if v.Layer != Bundled {
		v.V = question.Bounded(question.Default, got[0], got[1], got[2])
	}
	return v, nil
}

// defaultQuestionLimits is question.Default with every key bundled.
func defaultQuestionLimits() QuestionLimitsValue {
	v := QuestionLimitsValue{Value: Value[question.Limits]{V: question.Default, Layer: Bundled, Origin: "bundled"}}
	for _, b := range question.Bounds {
		v.Sources = append(v.Sources, LimitSource{Key: b.Key, Layer: Bundled, Origin: "bundled"})
	}
	return v
}

// boundedInt decodes one layer's value for b's key as an int and holds it to
// b, refusing with a *KeyRefusedError naming the key, the value, the bounds
// and the file.
func boundedInt(f Found, b question.Bound) (int, error) {
	n, err := Decode[int](f.Raw)
	if err != nil {
		return 0, &KeyRefusedError{Key: b.Key, Value: compact(f.Raw), Bounds: b, Layer: f.Layer, Origin: f.Origin, NotInteger: true}
	}
	if !b.Admits(n) {
		return 0, &KeyRefusedError{Key: b.Key, Value: compact(f.Raw), Bounds: b, Layer: f.Layer, Origin: f.Origin}
	}
	return n, nil
}
