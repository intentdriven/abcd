package matrix

// reviewscore.go scores each reviewer model of a matrix run on its own, never
// by vote (adr-2610161733400551, spc-2610161949559050 step 5), against copies
// whose truth is known: copies the hidden tests pass (known good), copies
// they fail, and copies with a defect seeded on purpose. So recall and the
// true-negative rate both have a denominator, as itd-81 asks. A rate with no
// copy under it is not estimable, never zero.
//
// A model's family is its developer, the organisation that trained it.
// Verdicts a reviewer gives on a copy its own family wrote are scored apart
// from the rest, and so are verdicts where either family cannot be read: an
// unread name never dilutes either score. The reviewer's guess at each copy's
// author, asked after its verdict, is scored as the measured leak of what
// blinding by metadata leaves in the content.
//
// A reviewer's answer is a model's output and is read as untrusted input
// (parseReviewAnswer); a verdict set naming a copy or a reviewer the run does
// not hold, or two verdicts on one copy, is refused whole.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"unicode"
	"unicode/utf8"
)

// CopyTruth is what is known of a reviewed copy before any reviewer sees it.
type CopyTruth string

const (
	// TruthPassing: the hidden tests pass the copy, so it is known good.
	TruthPassing CopyTruth = "passing"
	// TruthFailing: the hidden tests fail the copy.
	TruthFailing CopyTruth = "failing"
	// TruthSeeded: a defect was seeded into the copy on purpose.
	TruthSeeded CopyTruth = "seeded"
)

func (t CopyTruth) valid() bool { return t == TruthPassing || t == TruthFailing || t == TruthSeeded }

// developers is each model family's developer (modelFamilies, samemodel.go).
var developers = map[string]string{
	"opus": "anthropic", "sonnet": "anthropic", "haiku": "anthropic", "fable": "anthropic", "mythos": "anthropic",
	"gpt": "openai", oSeries: "openai",
	"gemini": "google", "gemma": "google",
	"llama":   "meta",
	"mistral": "mistral", "mixtral": "mistral", "codestral": "mistral", "devstral": "mistral",
	"magistral": "mistral", "ministral": "mistral", "pixtral": "mistral",
	"grok": "xai", "deepseek": "deepseek", "qwen": "alibaba", "kimi": "moonshot", "glm": "zhipu", "phi": "microsoft",
}

// developerOf is the developer of the model s names, "" where the name is
// unknown.
func developerOf(s string) string {
	n, ok := readModel(s)
	if !ok {
		return ""
	}
	return developers[n.family]
}

// guessedModel reports whether a guess names the builder's model: both names
// read to one family and one version, each written. A name with no version
// is never shown to be the model.
func guessedModel(guess, builder string) bool {
	g, okG := readModel(guess)
	b, okB := readModel(builder)
	return okG && okB && g.family == b.family && g.version != "" && g.version == b.version
}

// ReviewedCopy is one copy a reviewer is asked about: the opaque label it
// went under, the trial it is (a seeded copy keeps the key of the copy it was
// seeded into), and what is known of it.
type ReviewedCopy struct {
	Label string    `json:"label"`
	Key   TrialKey  `json:"key"`
	Truth CopyTruth `json:"truth"`
}

// ReviewVerdict is one reviewer cell's verdict on one copy.
type ReviewVerdict struct {
	Reviewer    string `json:"reviewer"`
	Label       string `json:"label"`
	Defective   bool   `json:"defective"`
	AuthorGuess string `json:"author_guess"`
}

// Rate is a count of hits out of the copies it is over.
type Rate struct {
	Hits int `json:"hits"`
	Of   int `json:"of"`
}

// Estimable reports whether the rate has any copy under it.
func (r Rate) Estimable() bool { return r.Of > 0 }

func (r *Rate) add(hit bool) {
	r.Of++
	if hit {
		r.Hits++
	}
}

// ReviewScores is one reviewer's verdicts on one set of copies: recall on
// every defective copy and on each kind apart, and the true-negative rate on
// the known-good ones.
type ReviewScores struct {
	Recall        Rate `json:"recall"`
	RecallFailing Rate `json:"recall_failing"`
	RecallSeeded  Rate `json:"recall_seeded"`
	TrueNegative  Rate `json:"true_negative"`
}

func (s *ReviewScores) add(truth CopyTruth, defective bool) {
	switch truth {
	case TruthPassing:
		s.TrueNegative.add(!defective)
	case TruthFailing:
		s.Recall.add(defective)
		s.RecallFailing.add(defective)
	case TruthSeeded:
		s.Recall.add(defective)
		s.RecallSeeded.add(defective)
	}
}

// GuessScore is the author guess as the measured leak: of the verdicts
// given, how many guessed, and of those, how many named the builder's model
// and how many its developer.
type GuessScore struct {
	Asked     int  `json:"asked"`
	Guessed   int  `json:"guessed"`
	Model     Rate `json:"model"`
	Developer Rate `json:"developer"`
}

// ReviewerScore is one reviewer cell's score, on its own.
type ReviewerScore struct {
	Reviewer string `json:"reviewer"`
	Model    string `json:"model"`
	// OtherFamily is the reviewer's score: its verdicts on copies another
	// developer's model wrote. SameFamily and UnknownFamily are reported
	// apart from it.
	OtherFamily   ReviewScores `json:"other_family"`
	SameFamily    ReviewScores `json:"same_family"`
	UnknownFamily ReviewScores `json:"unknown_family"`
	// Unanswered counts the copies the reviewer gave no verdict on: no
	// evidence either way, so in no rate.
	Unanswered  int        `json:"unanswered"`
	AuthorGuess GuessScore `json:"author_guess"`
}

// maxGuessBytes bounds an author guess, as a model name is bounded.
const maxGuessBytes = 200

// scoreReviewers scores every reviewer cell of pins on its own, in cell
// order, from its verdicts on copies. Each copy's builder is the builder cell
// its key names.
func scoreReviewers(copies []ReviewedCopy, pins []CellPin, verdicts []ReviewVerdict) ([]ReviewerScore, error) {
	builders, reviewers := map[string]string{}, map[string]string{}
	var order []string
	for _, p := range pins {
		switch p.Role {
		case RoleBuilder:
			builders[p.Cell] = p.Model
		case RoleReviewer:
			if _, dup := reviewers[p.Cell]; !dup {
				order = append(order, p.Cell)
			}
			reviewers[p.Cell] = p.Model
		}
	}
	byLabel := map[string]ReviewedCopy{}
	for _, c := range copies {
		if !reviewLabelRe.MatchString(c.Label) {
			return nil, fmt.Errorf("%q is not an opaque copy label", clip(c.Label))
		}
		if _, dup := byLabel[c.Label]; dup {
			return nil, fmt.Errorf("two copies go under the label %s", c.Label)
		}
		if err := c.Key.validate(); err != nil {
			return nil, err
		}
		if _, ok := builders[c.Key.Cell]; !ok {
			return nil, fmt.Errorf("copy %s is %s, and no builder cell of the run is %q", c.Label, c.Key, clip(c.Key.Cell))
		}
		if !c.Truth.valid() {
			return nil, fmt.Errorf("copy %s has truth %q; a copy is passing, failing or seeded", c.Label, clip(string(c.Truth)))
		}
		byLabel[c.Label] = c
	}
	type seat struct{ reviewer, label string }
	given := map[seat]ReviewVerdict{}
	for _, v := range verdicts {
		if _, ok := reviewers[v.Reviewer]; !ok {
			return nil, fmt.Errorf("a verdict names %q, which is no reviewer cell of the run", clip(v.Reviewer))
		}
		if _, ok := byLabel[v.Label]; !ok {
			return nil, fmt.Errorf("a verdict of %s names %q, which is no copy the run asked about", v.Reviewer, clip(v.Label))
		}
		if err := checkGuess(v.AuthorGuess); err != nil {
			return nil, fmt.Errorf("the verdict of %s on %s: %w", v.Reviewer, v.Label, err)
		}
		s := seat{v.Reviewer, v.Label}
		if _, dup := given[s]; dup {
			return nil, fmt.Errorf("%s gave two verdicts on %s", v.Reviewer, v.Label)
		}
		given[s] = v
	}
	labels := make([]string, 0, len(byLabel))
	for l := range byLabel {
		labels = append(labels, l)
	}
	sort.Strings(labels)
	out := make([]ReviewerScore, 0, len(order))
	for _, r := range order {
		sc := ReviewerScore{Reviewer: r, Model: reviewers[r]}
		mine := developerOf(reviewers[r])
		for _, l := range labels {
			c := byLabel[l]
			v, ok := given[seat{r, l}]
			if !ok {
				sc.Unanswered++
				continue
			}
			builder := builders[c.Key.Cell]
			theirs := developerOf(builder)
			switch {
			case mine == "" || theirs == "":
				sc.UnknownFamily.add(c.Truth, v.Defective)
			case mine == theirs:
				sc.SameFamily.add(c.Truth, v.Defective)
			default:
				sc.OtherFamily.add(c.Truth, v.Defective)
			}
			sc.AuthorGuess.Asked++
			if v.AuthorGuess == "" {
				continue
			}
			sc.AuthorGuess.Guessed++
			sc.AuthorGuess.Model.add(guessedModel(v.AuthorGuess, builder))
			dev := developerOf(v.AuthorGuess)
			sc.AuthorGuess.Developer.add(dev != "" && dev == theirs)
		}
		out = append(out, sc)
	}
	return out, nil
}

// checkGuess holds an author guess to a model name's bounds: printable text
// of at most maxGuessBytes, or empty for a reviewer that declined.
func checkGuess(g string) error {
	if len(g) > maxGuessBytes {
		return fmt.Errorf("the author guess is %d bytes; a guess names a model in at most %d", len(g), maxGuessBytes)
	}
	if !utf8.ValidString(g) {
		return errors.New("the author guess is not valid UTF-8")
	}
	for _, r := range g {
		if unicode.IsControl(r) {
			return errors.New("the author guess holds a control character")
		}
	}
	return nil
}

// The bounds a reviewer's answer is read within.
const (
	maxReviewAnswerBytes = 256 << 10
	maxReviewFindings    = 200
	maxFindingBytes      = 4000
)

// reviewFinding is one finding a reviewer gives, kept for the adjudication a
// later step makes.
type reviewFinding struct {
	Item string `json:"item"`
	Text string `json:"text"`
}

// reviewAnswer is the answer reviewAsk asks for.
type reviewAnswer struct {
	Label       string          `json:"label"`
	Verdict     string          `json:"verdict"`
	Findings    []reviewFinding `json:"findings"`
	AuthorGuess string          `json:"author_guess"`
}

// parseReviewAnswer reads reviewer's answer about the copy under label as
// untrusted input: one JSON object within maxReviewAnswerBytes, holding the
// asked keys and no other, answering that label, with a verdict of defect or
// clean, bounded findings, and a guess held to checkGuess.
func parseReviewAnswer(data []byte, reviewer, label string) (ReviewVerdict, error) {
	if len(data) > maxReviewAnswerBytes {
		return ReviewVerdict{}, fmt.Errorf("the answer is %d bytes; a review answers in at most %d", len(data), maxReviewAnswerBytes)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var a reviewAnswer
	if err := dec.Decode(&a); err != nil {
		return ReviewVerdict{}, fmt.Errorf("the answer is not the JSON object the review asks for: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return ReviewVerdict{}, errors.New("the answer holds more than the one JSON object the review asks for")
	}
	if a.Label != label {
		return ReviewVerdict{}, fmt.Errorf("the answer is about %q, and the review asked about %s", clip(a.Label), label)
	}
	if a.Verdict != "defect" && a.Verdict != "clean" {
		return ReviewVerdict{}, fmt.Errorf("the answer's verdict is %q; a verdict is defect or clean", clip(a.Verdict))
	}
	if len(a.Findings) > maxReviewFindings {
		return ReviewVerdict{}, fmt.Errorf("the answer gives %d findings; a review gives at most %d", len(a.Findings), maxReviewFindings)
	}
	for _, f := range a.Findings {
		if len(f.Item) > maxIDBytes || len(f.Text) > maxFindingBytes || !utf8.ValidString(f.Text) {
			return ReviewVerdict{}, errors.New("a finding of the answer is past a finding's bounds")
		}
	}
	if err := checkGuess(a.AuthorGuess); err != nil {
		return ReviewVerdict{}, err
	}
	return ReviewVerdict{Reviewer: reviewer, Label: label, Defective: a.Verdict == "defect", AuthorGuess: a.AuthorGuess}, nil
}
