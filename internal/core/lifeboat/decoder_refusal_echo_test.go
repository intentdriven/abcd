package lifeboat

import (
	"strings"
	"testing"
)

// TestSynthesisDecoderRefusalsRedactTheKey — iss-2609290218032954. The four
// host-composed lifeboat payloads decode strictly, and encoding/json's refusal
// names an undeclared field by the payload's own key. That message was returned
// raw, so a token or a home path in a key reached the terminal and the
// transcript. The key is still named, redacted through the canonical scanner.
func TestSynthesisDecoderRefusalsRedactTheKey(t *testing.T) {
	const key = `"reviewer_notes /Users/zzotherperson/notes":1` // abcd-lint:allow — a planted home path in a KEY
	cases := map[string]func(t *testing.T) error{
		"lessons": func(t *testing.T) error {
			_, err := IngestLessons(stdFixture(t), []byte(`{"schema_version":1,`+key+`,"lessons":[]}`))
			return err
		},
		"press release": func(t *testing.T) error {
			dir := synthLifeboat(t, map[string]string{briefPressReleasePath: briefPR})
			_, err := ComposePressRelease(dir, []byte(`{"schema_version":1,"mode":"delegated","prompt_version":"0.1.0",`+
				`"headline":"h","body":"b","evidence":["brief/x"],`+key+`}`))
			return err
		},
		"principles": func(t *testing.T) error {
			_, err := SynthesizePrinciples(adrLifeboat(t), []byte(`{"schema_version":1,"mode":"delegated",`+
				`"prompt_version":"0.1.0","principles":[],`+key+`}`))
			return err
		},
		"review": func(t *testing.T) error {
			dir := reviewFixture(t, "abc", &Summary{Grounded: 7, Blank: 3})
			_, err := ReviewLifeboat(dir, realSourceDir(t), []byte(`{"schema_version":1,"mode":"delegated",`+
				`"prompt_version":"0.1.0","verdict":"SHIP","findings":[],`+key+`}`))
			return err
		},
	}
	for name, run := range cases {
		t.Run(name, func(t *testing.T) {
			err := run(t)
			if err == nil {
				t.Fatal("a payload carrying an undeclared key was accepted")
			}
			if strings.Contains(err.Error(), "zzotherperson") {
				t.Errorf("the refusal echoes the payload's key: %v", err)
			}
			if !strings.Contains(err.Error(), "reviewer_notes") {
				t.Errorf("the refusal no longer names the undeclared field: %v", err)
			}
		})
	}
}
