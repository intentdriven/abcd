package lifeboat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The principles payload at schema version 2 (spc-2609020626042471): every
// entry carries claim_type, reference and comparison beside its evidence, a
// declined claim is null, and an absent key drops the entry.

func strp(s string) *string { return &s }

// rawEntries decodes principles.json's entries as raw objects, so absent and
// null are told apart.
func rawEntries(t *testing.T, dir string) []map[string]json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "principles.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Principles []map[string]json.RawMessage `json:"principles"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	return f.Principles
}

// TestDeterministicPrinciplesCarryTheFourKeys is ac-8's deterministic half: the
// fallback carries what it can establish — the ADR handle it distilled from as
// the reference — and declines the claim type and the comparison with null.
func TestDeterministicPrinciplesCarryTheFourKeys(t *testing.T) {
	dir := adrLifeboat(t)
	if _, err := SynthesizePrinciples(dir, nil); err != nil {
		t.Fatal(err)
	}
	pf := readPrinciplesFile(t, dir)
	if pf.SchemaVersion != 2 {
		t.Errorf("schema_version = %d, want 2", pf.SchemaVersion)
	}
	entries := rawEntries(t, dir)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	for _, e := range entries {
		for _, k := range []string{"claim_type", "reference", "comparison", "evidence"} {
			if _, ok := e[k]; !ok {
				t.Errorf("entry %s carries no %q key", e["id"], k)
			}
		}
		if string(e["claim_type"]) != "null" || string(e["comparison"]) != "null" {
			t.Errorf("entry %s: claim_type %s, comparison %s; the fallback invents neither", e["id"], e["claim_type"], e["comparison"])
		}
	}
	if got := pf.Principles[0].Reference; got == nil || *got != "adr-24" {
		t.Errorf("the first principle's reference = %v, want the ADR it was distilled from", got)
	}
}

// TestDelegatedPrincipleWithoutAKeyIsDropped is ac-8's delegated half.
func TestDelegatedPrincipleWithoutAKeyIsDropped(t *testing.T) {
	dir := adrLifeboat(t)
	payload := `{"schema_version": 2, "mode": "delegated", "prompt_version": "0.2.0", "principles": [
	  {"id": "prn-whole", "principle": "The cascade is fixed.", "confidence": "high",
	   "claim_type": "causal", "reference": "adr-24", "comparison": null, "evidence": ["adr-24"]},
	  {"id": "prn-short", "principle": "One binary.", "confidence": "high",
	   "claim_type": "context", "reference": "adr-31", "evidence": ["adr-31"]}
	]}`
	res, err := SynthesizePrinciples(dir, []byte(payload))
	if err != nil {
		t.Fatal(err)
	}
	if res.Written != 1 || res.Dropped != 1 || res.Drops[0].ID != "prn-short" || !strings.Contains(res.Drops[0].Reason, "comparison") {
		t.Fatalf("result = %+v, want prn-short dropped naming the missing comparison", res)
	}
}

// TestSchemaVersionOneIsRefused: a consumer of principles.json sees the change
// by the version, and a payload at the previous one is refused.
func TestSchemaVersionOneIsRefused(t *testing.T) {
	dir := adrLifeboat(t)
	_, err := SynthesizePrinciples(dir, []byte(`{"schema_version": 1, "mode": "delegated", "prompt_version": "0.1.0", "principles": []}`))
	if err == nil || !strings.Contains(err.Error(), "unsupported principles schema_version 1") {
		t.Fatalf("a version 1 payload was not refused as unsupported: %v", err)
	}
}

// TestNullKeyIsCarriedNotDropped: a declined claim survives, as a null.
func TestNullKeyIsCarriedNotDropped(t *testing.T) {
	dir := adrLifeboat(t)
	payload := principlesPayload(t, "0.2.0", Principle{
		ID: "prn-declined", Principle: "The cascade is fixed.", Confidence: ConfidenceHigh,
		Evidence: []string{"adr-24"},
	})
	res, err := SynthesizePrinciples(dir, payload)
	if err != nil {
		t.Fatal(err)
	}
	if res.Written != 1 {
		t.Fatalf("a principle declining three claims was dropped: %+v", res)
	}
	e := rawEntries(t, dir)[0]
	for _, k := range []string{"claim_type", "reference", "comparison"} {
		if string(e[k]) != "null" {
			t.Errorf("%s = %s, want null carried", k, e[k])
		}
	}
}

// TestDelegatedMechanismIsWrittenAsCausal: the alias is read and never written.
func TestDelegatedMechanismIsWrittenAsCausal(t *testing.T) {
	dir := adrLifeboat(t)
	payload := principlesPayload(t, "0.2.0", Principle{
		ID: "prn-mech", Principle: "The cascade is fixed.", Confidence: ConfidenceHigh,
		ClaimType: strp("mechanism"), Reference: strp("adr-24"), Comparison: strp("Routing against a fixed cascade."),
		Evidence: []string{"adr-24"},
	}, Principle{
		ID: "prn-fourth", Principle: "One binary.", Confidence: ConfidenceHigh,
		ClaimType: strp("normative"), Evidence: []string{"adr-31"},
	})
	res, err := SynthesizePrinciples(dir, payload)
	if err != nil {
		t.Fatal(err)
	}
	pf := readPrinciplesFile(t, dir)
	if len(pf.Principles) != 1 || pf.Principles[0].ClaimType == nil || *pf.Principles[0].ClaimType != "causal" {
		t.Fatalf("principles = %+v, want prn-mech written as causal", pf.Principles)
	}
	if res.Dropped != 1 || res.Drops[0].ID != "prn-fourth" {
		t.Errorf("a fourth claim type was not dropped: %+v", res)
	}
}

// TestPrinciplesMarkdownRendersTheKeys: the human render carries the three
// claims beside the evidence, a declined one said as such.
func TestPrinciplesMarkdownRendersTheKeys(t *testing.T) {
	dir := adrLifeboat(t)
	payload := principlesPayload(t, "0.2.0", Principle{
		ID: "prn-whole", Principle: "The cascade is fixed.", Confidence: ConfidenceHigh,
		ClaimType: strp("causal"), Reference: strp("adr-24"), Evidence: []string{"adr-24"},
	})
	if _, err := SynthesizePrinciples(dir, payload); err != nil {
		t.Fatal(err)
	}
	md, err := os.ReadFile(filepath.Join(dir, "principles.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Claim type: causal", "Reference: adr-24", "Comparison: none stated", "Evidence: adr-24"} {
		if !strings.Contains(string(md), want) {
			t.Errorf("principles.md does not carry %q:\n%s", want, md)
		}
	}
}

// TestEvidenceCarriesPackedIDsOnly: the payload's evidence is the lifeboat's
// own grammar — packed record ids, finding ids and packed paths — so a reading
// item or a condition identity, which the lifeboat packs no store for, resolves
// to nothing and is filtered out.
func TestEvidenceCarriesPackedIDsOnly(t *testing.T) {
	dir := adrLifeboat(t)
	payload := principlesPayload(t, "0.2.0", Principle{
		ID: "prn-mixed", Principle: "The cascade is fixed.", Confidence: ConfidenceHigh,
		Evidence: []string{"adr-24", "rdi-2609020000000009", "cond-2608311949582375", "prn-other"},
	})
	if _, err := SynthesizePrinciples(dir, payload); err != nil {
		t.Fatal(err)
	}
	pf := readPrinciplesFile(t, dir)
	if len(pf.Principles) != 1 || strings.Join(pf.Principles[0].Evidence, ",") != "adr-24" {
		t.Errorf("evidence = %+v, want the packed adr-24 alone", pf.Principles)
	}
}
