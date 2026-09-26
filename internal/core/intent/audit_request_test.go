package intent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// audit_request_test.go — what the fidelity review request hands the auditor.
// The request is the auditor's whole
// brief: every set the ingest checks a verdict against has to be IN it, stated
// the way the ingest will read it, or a reviewer working from the request alone
// learns the set from a refusal.

// readRequest returns the request file the emit wrote for rcp.
func readRequest(t *testing.T, root, rcp string) string {
	t.Helper()
	rb, err := os.ReadFile(filepath.Join(root, reviewsDir, rcp+".request.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(rb)
}

// requestSection returns the body of one `## ` section of a request, up to the
// next `## ` heading, and the heading line itself.
func requestSection(t *testing.T, req, prefix string) (heading, body string) {
	t.Helper()
	_, after, ok := strings.Cut(req, "\n## "+prefix)
	if !ok {
		t.Fatalf("the request carries no `## %s` section:\n%s", prefix, req)
	}
	heading, body, _ = strings.Cut(after, "\n")
	if i := strings.Index(body, "\n## "); i >= 0 {
		body = body[:i]
	}
	return "## " + prefix + heading, body
}

// firstCodeBlock is the first indented code block in a section body: the
// contiguous four-space lines up to the line that ends them, unindented.
func firstCodeBlock(body string) string {
	var lines []string
	for _, ln := range strings.Split(body, "\n") {
		s, ok := strings.CutPrefix(ln, "    ")
		if !ok {
			if len(lines) > 0 {
				break
			}
			continue
		}
		lines = append(lines, s)
	}
	return strings.Join(lines, "\n")
}

// TestAuditRequestListsTheScopeConditionIdentities is iss-2609181121301638: the
// verdict must dispose every cond-… identity the intent carries, exactly once,
// and the request used to name none of them — the auditor scraped HTML comments
// out of the record by hand, and a miscount quarantined the verdict. The
// request now lists each identity with its text, and prints the counts the
// ingest checks: the criteria's K and the conditions' total.
func TestAuditRequestListsTheScopeConditionIdentities(t *testing.T) {
	root := t.TempDir()
	rcp := shipWithConditions(t, root,
		stampedCondition(condOne, "holds on POSIX"),
		stampedCondition(condTwo, "holds below 10k records"),
	)
	req := readRequest(t, root, rcp)

	heading, body := requestSection(t, req, "Scope Conditions")
	if !strings.Contains(heading, "2 conditions") {
		t.Errorf("the Scope Conditions heading does not state the count the ingest checks: %q", heading)
	}
	for _, want := range []string{
		"- " + condOne + " — holds on POSIX",
		"- " + condTwo + " — holds below 10k records",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the Scope Conditions block lacks %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "<!--") {
		t.Errorf("the block quotes the identity marker rather than the identity:\n%s", body)
	}

	acHeading, _ := requestSection(t, req, "Acceptance Criteria")
	if !strings.Contains(acHeading, "1 criterion") || !strings.Contains(acHeading, "ac-1..ac-1") {
		t.Errorf("the Acceptance Criteria heading does not print K: %q", acHeading)
	}
}

// TestAuditRequestStatesAConditionlessIntentTakesAnEmptyList: an intent that
// records no condition takes an empty scope_conditions list, and a non-empty
// one is refused — so the request says so rather than omitting the block.
func TestAuditRequestStatesAConditionlessIntentTakesAnEmptyList(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	heading, body := requestSection(t, readRequest(t, root, rcp), "Scope Conditions")
	if !strings.Contains(heading+body, "empty list") {
		t.Fatalf("a conditionless intent's request does not say scope_conditions is empty:\n%s\n%s", heading, body)
	}
}

// TestAuditRequestCarriesTheVerdictShape is iss-2609181121305984: the request
// named no schema, and the shape lived only in the bundled agent definition, so
// a reviewer working from the request learned it from ingest refusals. The
// request now carries it, rendered from the very struct the ingest decodes into
// — so it is that schema, not a copy of it: it decodes under the ingest's own
// DisallowUnknownFields, names every field the struct declares, and echoes this
// receipt.
func TestAuditRequestCarriesTheVerdictShape(t *testing.T) {
	root := t.TempDir()
	rcp := shipWithConditions(t, root, stampedCondition(condOne, "holds on POSIX"))
	_, body := requestSection(t, readRequest(t, root, rcp), "Verdict shape")

	shape := firstCodeBlock(body)
	dec := json.NewDecoder(strings.NewReader(shape))
	dec.DisallowUnknownFields()
	var v verdict
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("the stated shape does not decode as the ingest decodes: %v\n%s", err, shape)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(shape), &top); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(keysOf(top), ","), strings.Join(jsonTagsOf(verdict{}), ","); got != want {
		t.Fatalf("the stated shape names %s, the struct the ingest decodes declares %s", got, want)
	}
	if v.Type != VerdictType || v.ReceiptID != rcp {
		t.Fatalf("the stated shape carries _type %q and receipt_id %q, want %q and %q", v.Type, v.ReceiptID, VerdictType, rcp)
	}
	if len(v.Criteria) != 1 || len(v.Criteria[0].Evidence) != 1 || len(v.ScopeConditions) != 1 ||
		len(v.GapAudit.Honoured) != 1 || len(v.InputAttestations) != 1 {
		t.Fatalf("every list in the stated shape shows one element, so its members are named: %+v", v)
	}
	for k := range verdictEnum {
		if _, ok := v.AcceptanceRollup[k]; !ok {
			t.Errorf("the stated acceptance_rollup lacks the key %q", k)
		}
	}
}

// TestShapeHintsNameRealFields: a placeholder keyed to a field name the payload
// no longer carries would sit in the table unused, so every hint names a json
// field somewhere in the struct its ingest decodes.
func TestShapeHintsNameRealFields(t *testing.T) {
	for _, c := range []struct {
		name  string
		shape reflect.Type
		hints map[string]string
	}{
		{"verdict", reflect.TypeOf(verdict{}), verdictShapeHints("rcp-000000000000")},
		{"consistency findings", reflect.TypeOf(consistencyPayload{}), consistencyShapeHints("rcp-000000000000")},
	} {
		fields := map[string]bool{}
		var walk func(rt reflect.Type)
		walk = func(rt reflect.Type) {
			switch rt.Kind() {
			case reflect.Struct:
				for i := 0; i < rt.NumField(); i++ {
					name, _, _ := strings.Cut(rt.Field(i).Tag.Get("json"), ",")
					fields[name] = true
					walk(rt.Field(i).Type)
				}
			case reflect.Slice, reflect.Map:
				walk(rt.Elem())
			}
		}
		walk(c.shape)
		for name := range c.hints {
			if !fields[name] {
				t.Errorf("%s shape hint %q names no field of the payload", c.name, name)
			}
		}
	}
}

// TestConsistencyRequestCarriesTheFindingsShape is iss-2609262011046013, the
// Role 2 sibling of iss-2609181121305984: the consistency request stated the
// classes and the rubric but no findings shape. It now carries one, rendered
// from the struct the consistency ingest decodes, with both ends of a finding
// shown because the ingest requires exactly two.
func TestConsistencyRequestCarriesTheFindingsShape(t *testing.T) {
	root := consistencyRepo(t).Root()
	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	rb, err := os.ReadFile(filepath.Join(root, em.RequestPath))
	if err != nil {
		t.Fatal(err)
	}
	_, body := requestSection(t, string(rb), "Findings shape")
	shape := firstCodeBlock(body)
	dec := json.NewDecoder(strings.NewReader(shape))
	dec.DisallowUnknownFields()
	var p consistencyPayload
	if err := dec.Decode(&p); err != nil {
		t.Fatalf("the stated shape does not decode as the ingest decodes: %v\n%s", err, shape)
	}
	if p.Type != ConsistencyType || p.ReceiptID != em.ReceiptID {
		t.Fatalf("the stated shape carries _type %q and receipt_id %q, want %q and %q", p.Type, p.ReceiptID, ConsistencyType, em.ReceiptID)
	}
	if len(p.Findings) != 1 || len(p.Findings[0].Ends) != 2 {
		t.Fatalf("the stated shape shows %+v, want one finding with its two ends", p.Findings)
	}
}

// TestRenderShapeSkipsFieldsTheDecoderCannotSet: fillShape writes through
// reflection, and an unexported field is read-only there, so one added to a
// decode struct would panic every emit. encoding/json never decodes such a
// field, so it is no part of the shape and is skipped, while an exported field
// promoted from an unexported embedded struct IS decoded and still shows.
func TestRenderShapeSkipsFieldsTheDecoderCannotSet(t *testing.T) {
	type inner struct {
		Promoted string `json:"promoted"`
	}
	type shape struct {
		inner
		Named  string            `json:"named"`
		note   string            // unexported: read-only through reflection
		marks  []string          // unexported: read-only through reflection
		byName map[string]string // unexported: read-only through reflection
	}
	var out string
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("renderShape panicked on an unexported field: %v", r)
			}
		}()
		out = renderShape(reflect.TypeOf(shape{}), shapeSpec{hints: map[string]string{"named": "<named>"}})
	}()
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("rendered shape is not one JSON object: %v\n%s", err, out)
	}
	if got["named"] != "<named>" || got["promoted"] != "<string>" || len(got) != 2 {
		t.Fatalf("rendered shape = %v, want exactly named and the promoted field", got)
	}
}
