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

// A model-matrix check copy's verdict is validated by the recorder's own
// decoder and its findings handed back, with nothing recorded: the release
// gates read receipts, and a copy's verdict never becomes one (decision 30 of
// itd-2610161733316518).
func TestReportFindingsValidatesAVerdictAndHandsItsFindingsBack(t *testing.T) {
	verdict, findings, err := ReportFindings([]byte(strings.Replace(VerdictExample, `"17-guard.md"`, `"04-surfaces/17-guard.md"`, 1)))
	if err != nil {
		t.Fatalf("ReportFindings(VerdictExample): %v", err)
	}
	if verdict != "HOLD" || len(findings) != 1 {
		t.Fatalf("ReportFindings = %q, %+v; want HOLD and the one finding", verdict, findings)
	}
	f := findings[0]
	if f.Doc != DocBrief || f.Chapter != "17-guard.md" || f.Sentence != "The guard reads its registry from the plugin root." ||
		f.Replacement == "" || !strings.Contains(f.Evidence, "registry.go") {
		t.Fatalf("the finding = %+v; want the chapter read through the recorder's parse and every field kept", f)
	}
	for name, raw := range map[string]string{
		"not JSON":              `{`,
		"a HOLD naming nothing": strings.Replace(VerdictExample, `"doc": "brief"`, `"doc": "public"`, 1),
		"a traversal chapter":   strings.Replace(VerdictExample, `"17-guard.md"`, `"../17-guard.md"`, 1),
		"an unknown key":        strings.Replace(VerdictExample, `"tier"`, `"model": "x", "tier"`, 1),
	} {
		if _, _, err := ReportFindings([]byte(raw)); err == nil {
			t.Errorf("%s: ReportFindings accepted it", name)
		}
	}
}
