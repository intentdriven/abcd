package lifeboat

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// principles_contract_test.go pins agents/principle-distiller.md to the payload
// this package decodes (spc-2609020626042471). The definition is what the
// host-delegated distiller is told to emit and validateDelegatedPrinciples is
// what refuses it, so the two drifting apart is silent until a real payload is
// rejected wholesale by DisallowUnknownFields or drops every entry for a key it
// was never told to carry.
//
// It lives here rather than beside the agent_contract lint rule because
// core/lifeboat imports core/lint: a lint test importing this package back
// would be a cycle, and the lockstep assertion is worthless against a
// re-declared copy of the struct.

var distillerPromptVersionRe = regexp.MustCompile(`(?m)^prompt_version:\s*(\S+)\s*$`)

func TestDistillerDefinitionMatchesThePrinciplesPayload(t *testing.T) {
	path := filepath.Join("..", "..", "..", "agents", "principle-distiller.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	fence := regexp.MustCompile("(?s)```json\n(.*?)\n```").FindAllStringSubmatch(doc, -1)
	if len(fence) != 1 {
		t.Fatalf("the definition carries %d json example(s), want exactly one", len(fence))
	}
	body := []byte(fence[0][1])

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	var pf PrinciplesFile
	if err := dec.Decode(&pf); err != nil {
		t.Fatalf("the definition's example does not decode strictly: %v", err)
	}
	if pf.SchemaVersion != PrinciplesSchemaVersion {
		t.Errorf("the example's schema_version is %d, the payload's is %d", pf.SchemaVersion, PrinciplesSchemaVersion)
	}
	m := distillerPromptVersionRe.FindStringSubmatch(doc)
	if m == nil || pf.PromptVersion != m[1] {
		t.Errorf("the example's prompt_version %q is not the definition's own %v", pf.PromptVersion, m)
	}

	// Every entry documents exactly the fields the struct carries: a key the
	// struct lacks is a whole-payload refusal, and one it carries that the
	// example omits is a key the distiller is never shown.
	var raw struct {
		Principles []map[string]json.RawMessage `json:"principles"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || len(raw.Principles) == 0 {
		t.Fatalf("the example carries no principle entry: %v", err)
	}
	want := jsonTags(reflect.TypeOf(Principle{}))
	for i, e := range raw.Principles {
		got := make([]string, 0, len(e))
		for k := range e {
			got = append(got, k)
		}
		sort.Strings(got)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("example entry %d carries %v, the payload's entry is %v", i, got, want)
		}
	}
	// And the example must survive the validator's own rules.
	for _, e := range raw.Principles {
		var p Principle
		b, _ := json.Marshal(e)
		_ = json.Unmarshal(b, &p)
		if _, reason := principleClaims(e, p); reason != "" {
			t.Errorf("the example entry %s would be dropped: %s", p.ID, reason)
		}
	}
	// The field rules name each claim key, so the distiller is told what it is.
	for _, k := range []string{"`claim_type`", "`reference`", "`comparison`", "`evidence`"} {
		if !strings.Contains(doc, k) {
			t.Errorf("the definition's field rules never name %s", k)
		}
	}
}

func jsonTags(rt reflect.Type) []string {
	out := make([]string, 0, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		name, _, _ := strings.Cut(rt.Field(i).Tag.Get("json"), ",")
		if name != "" && name != "-" {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}
