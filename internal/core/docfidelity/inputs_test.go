package docfidelity

import (
	"strings"
	"testing"
)

func TestVerdictExamplePassesCheck(t *testing.T) {
	if err := Check([]byte(VerdictExample)); err != nil {
		t.Fatalf("VerdictExample: %v", err)
	}
}

// TestVerdictReadsAnAbsentVersionAsOne: the verdict carried no version before
// VerdictSchemaVersion, so an absent one reads as 1, and any other is
// refused through the decoder Record runs.
func TestVerdictReadsAnAbsentVersionAsOne(t *testing.T) {
	const body = `"verificationResult": "PROMOTE", "judgeModel": "example-model-2026-10-01", "tier": "full", "failing": []`
	for _, doc := range []string{`{` + body + `}`, `{"schema_version": 1, ` + body + `}`} {
		if _, err := decodePayload([]byte(doc)); err != nil {
			t.Errorf("%s refused: %v", doc, err)
		}
	}
	for _, v := range []string{"2", "-1"} {
		_, err := decodePayload([]byte(`{"schema_version": ` + v + `, ` + body + `}`))
		if err == nil || !strings.Contains(err.Error(), "version 1") {
			t.Errorf("schema_version %s: err = %v, want a refusal naming version 1", v, err)
		}
	}
}
