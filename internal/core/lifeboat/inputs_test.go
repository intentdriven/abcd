package lifeboat

import (
	"strings"
	"testing"
)

func TestSynthesisExamplesPassTheirChecks(t *testing.T) {
	cases := []struct {
		name, example string
		check         func(string, []byte) error
	}{
		{"lessons", LessonsExample, CheckLessons},
		{"principles", PrinciplesExample, CheckPrinciples},
		{"press release", PressReleaseExample, CheckPressRelease},
		{"review", ReviewExample, CheckReview},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.check(t.TempDir(), []byte(c.example)); err != nil {
				t.Fatalf("the example does not pass: %v", err)
			}
			if err := c.check(t.TempDir(), []byte(strings.Replace(c.example, `"schema_version"`, `"schema"`, 1))); err == nil {
				t.Error("the check accepted a payload with an undeclared member and no schema_version")
			}
		})
	}
}
