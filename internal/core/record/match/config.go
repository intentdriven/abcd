package match

import (
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/layered"
)

// The compared fields: which text of each candidate family the match reads.
// The set is closed; a name outside it is refused, never ignored.
const (
	// FieldIssueBody is an open or resolved issue's markdown body.
	FieldIssueBody = "issue.body"
	// FieldIntentTitle is an intent's H1.
	FieldIntentTitle = "intent.title"
	// FieldIntentPressRelease is an intent's `## Press Release` section.
	FieldIntentPressRelease = "intent.press_release"
)

// DefaultFields is the bundled match.fields: every field in the set.
var DefaultFields = []string{FieldIssueBody, FieldIntentTitle, FieldIntentPressRelease}

var knownFields = map[string]bool{FieldIssueBody: true, FieldIntentTitle: true, FieldIntentPressRelease: true}

// Config is the resolved match configuration, with the origin of each value
// (the layered resolver's: "bundled", a file, or a flag), so a surface can say
// where a threshold came from.
type Config struct {
	Threshold       float64  `json:"threshold"`
	ThresholdOrigin string   `json:"threshold_origin"`
	Fields          []string `json:"fields"`
	FieldsOrigin    string   `json:"fields_origin"`
}

// Bundled is the configuration when nothing is configured.
func Bundled() Config {
	return Config{
		Threshold: DefaultThreshold, ThresholdOrigin: "bundled",
		Fields: append([]string(nil), DefaultFields...), FieldsOrigin: "bundled",
	}
}

// Compares reports whether the configuration reads field.
func (c Config) Compares(field string) bool {
	for _, f := range c.Fields {
		if f == field {
			return true
		}
	}
	return false
}

// LoadConfig resolves match.threshold and match.fields through the one layered
// configuration reader (.abcd/config.json, then ~/.abcd/config.json, then the
// bundled defaults). It claims the `match` namespace, so a misspelt key is
// refused rather than letting a default apply unannounced, and a value outside
// its range or set is refused naming its file, never replaced by the default.
func LoadConfig(r layered.Roots) (Config, error) {
	s, err := layered.Load(layered.Config, r)
	if err != nil {
		return Config{}, err
	}
	if err := s.Claim("match", "threshold", "fields"); err != nil {
		return Config{}, err
	}
	th, err := layered.Get(s, "match.threshold", DefaultThreshold, func(v float64) error {
		if !(v > 0 && v <= 1) {
			return fmt.Errorf("want a share of the new text's terms greater than 0 and at most 1")
		}
		return nil
	})
	if err != nil {
		return Config{}, err
	}
	fields, err := layered.Get(s, "match.fields", append([]string(nil), DefaultFields...), checkFields)
	if err != nil {
		return Config{}, err
	}
	return Config{
		Threshold: th.V, ThresholdOrigin: th.Origin,
		Fields: fields.V, FieldsOrigin: fields.Origin,
	}, nil
}

func checkFields(fs []string) error {
	accepted := strings.Join(DefaultFields, ", ")
	if len(fs) == 0 {
		return fmt.Errorf("want at least one of %s", accepted)
	}
	seen := map[string]bool{}
	for _, f := range fs {
		if !knownFields[f] {
			return fmt.Errorf("%q is not a compared field; want one or more of %s", layered.BoundKey(f), accepted)
		}
		if seen[f] {
			return fmt.Errorf("%q is named twice", f)
		}
		seen[f] = true
	}
	return nil
}
