package intent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// audit_contract_test.go pins `agents/intent-auditor.md` to the struct this
// package decodes. The definition is what the host-delegated agent is told to
// emit and the ingest is what actually refuses it, so the two drifting apart is
// silent until a real verdict dead-letters: a field the ingest requires but the
// definition never documents can only be supplied by luck, and a field the
// definition documents but the struct does not carry is rejected wholesale by
// DisallowUnknownFields.
//
// The test lives here rather than beside the agent_contract lint rule because
// the verdict schema is unexported: only this package can decode into it, and
// the lockstep assertion is worthless against a re-declared copy of the struct.

// auditorDefinitionPath is the definition the ingest's contract is published in.
var auditorDefinitionPath = filepath.Join(repoRootFromPackage, "agents", "intent-auditor.md")

// jsonFences returns the body of every ```json fenced block in a markdown file.
func jsonFences(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.Base(path), err)
	}
	var out []string
	var cur []string
	inFence := false
	for _, ln := range strings.Split(string(data), "\n") {
		switch {
		case !inFence && strings.TrimSpace(ln) == "```json":
			inFence, cur = true, nil
		case inFence && strings.TrimSpace(ln) == "```":
			inFence = false
			out = append(out, strings.Join(cur, "\n"))
		case inFence:
			cur = append(cur, ln)
		}
	}
	return out
}

// jsonTagsOf returns the json field names a struct declares, sorted.
func jsonTagsOf(v any) []string {
	rt := reflect.TypeOf(v)
	names := make([]string, 0, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		name, _, _ := strings.Cut(rt.Field(i).Tag.Get("json"), ",")
		if name != "" && name != "-" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// keysOf returns a decoded object's top-level keys, sorted.
func keysOf(m map[string]json.RawMessage) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// TestAuditorDefinitionMatchesTheVerdictSchema is the lockstep assertion: every
// verdict-shaped block the definition publishes decodes into the struct the
// ingest decodes, and documents exactly the fields that struct carries — the
// scope-condition dispositions included.
func TestAuditorDefinitionMatchesTheVerdictSchema(t *testing.T) {
	fences := jsonFences(t, auditorDefinitionPath)
	if len(fences) == 0 {
		t.Fatal("the auditor definition publishes no ```json block; the output contract is undocumented")
	}
	wantTop := jsonTagsOf(verdict{})
	wantCondition := jsonTagsOf(verdictCondition{})

	checked := 0
	for i, body := range fences {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(body), &raw); err != nil {
			t.Fatalf("json block %d in the auditor definition is not an object: %v", i, err)
		}
		// The error verdict is deliberately a different, minimal shape.
		if _, isError := raw["error"]; isError {
			continue
		}
		checked++
		if got := keysOf(raw); !reflect.DeepEqual(got, wantTop) {
			t.Errorf("json block %d documents keys %v, but the ingest decodes %v", i, got, wantTop)
		}
		dec := json.NewDecoder(strings.NewReader(body))
		dec.DisallowUnknownFields()
		var v verdict
		if err := dec.Decode(&v); err != nil {
			t.Errorf("json block %d does not decode into the verdict the ingest accepts: %v", i, err)
			continue
		}
		if len(v.ScopeConditions) == 0 {
			t.Errorf("json block %d documents no scope_conditions entry; the disposition rubric is unreachable to the agent", i)
		}
		var conds []map[string]json.RawMessage
		if err := json.Unmarshal(raw["scope_conditions"], &conds); err != nil {
			t.Errorf("json block %d: scope_conditions is not a list of objects: %v", i, err)
			continue
		}
		for j, c := range conds {
			if got := keysOf(c); !reflect.DeepEqual(got, wantCondition) {
				t.Errorf("json block %d scope_conditions[%d] documents keys %v, want %v", i, j, got, wantCondition)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no verdict-shaped json block was checked; the lockstep assertion is vacuous")
	}
}

// TestAuditorDefinitionDocumentsEveryDisposition proves the definition publishes
// the whole closed set the ingest refuses outside of — an agent told about three
// of four values can only reach the fourth by accident.
func TestAuditorDefinitionDocumentsEveryDisposition(t *testing.T) {
	data, err := os.ReadFile(auditorDefinitionPath)
	if err != nil {
		t.Fatal(err)
	}
	for value := range dispositionEnum {
		if !strings.Contains(string(data), "`"+value+"`") {
			t.Errorf("the auditor definition never names the disposition %q", value)
		}
	}
}
