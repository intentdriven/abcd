package matrix

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
)

// goodDeclaration is a matrix of all three task kinds, two builders and one
// reviewer, with a metered cell and so a cap.
const goodDeclaration = `{
  "schema_version": 1,
  "name": "routing check",
  "tasks": [
    {"id": "feature-a", "kind": "code", "intent": "itd-2610161733316518"},
    {"id": "doc-page", "kind": "writing", "source": ["docs/guide.md"], "criteria": "accurate against the source first"},
    {"id": "fidelity", "kind": "check", "material": ["docs/reference"]}
  ],
  "builders": [
    {"id": "host-big", "route": "host"},
    {"id": "or-cheap", "route": "builder=runner:opencode@openrouter/vendor/model", "settings": {"temperature": 0.6}}
  ],
  "reviewers": [
    {"id": "local-rev", "route": "reviewer=runner:opencode@local/qwen"}
  ],
  "repeats": {"pilot": 3, "max": 20},
  "spend_cap": {"amount": 25, "unit": "usd"},
  "load_cap": 4,
  "concurrency": 2,
  "judging": {"guests_may_judge": false}
}`

func TestADeclarationParsesIntoTasksCellsAndRepeats(t *testing.T) {
	d, err := parseDeclaration([]byte(goodDeclaration))
	if err != nil {
		t.Fatalf("parseDeclaration: %v", err)
	}
	if d.SchemaVersion != DeclarationSchema || d.Name != "routing check" {
		t.Fatalf("header = %d %q", d.SchemaVersion, d.Name)
	}
	if len(d.Tasks) != 3 || d.Tasks[0].Kind != KindCode || d.Tasks[0].Intent != "itd-2610161733316518" ||
		d.Tasks[1].Kind != KindWriting || d.Tasks[1].Criteria == "" || d.Tasks[2].Kind != KindCheck || d.Tasks[2].Material[0] != "docs/reference" {
		t.Fatalf("tasks = %+v", d.Tasks)
	}
	if len(d.Builders) != 2 || d.Builders[1].Route != "builder=runner:opencode@openrouter/vendor/model" ||
		string(d.Builders[1].Settings["temperature"]) != "0.6" || len(d.Reviewers) != 1 {
		t.Fatalf("cells = %+v %+v", d.Builders, d.Reviewers)
	}
	if d.Repeats != (Repeats{Pilot: 3, Max: 20}) || d.SpendCap == nil || d.SpendCap.Amount != 25 || d.SpendCap.Unit != "usd" ||
		d.LoadCap != 4 || d.Concurrency != 2 || d.Judging.GuestsMayJudge {
		t.Fatalf("run settings = %+v %+v %d %d %+v", d.Repeats, d.SpendCap, d.LoadCap, d.Concurrency, d.Judging)
	}
	if got := d.cells(); len(got) != 3 || got[2].role != RoleReviewer || got[0].role != RoleBuilder {
		t.Fatalf("cells() = %+v", got)
	}
}

func TestADeclarationWithAnUnknownKeyIsRefused(t *testing.T) {
	for name, doc := range map[string]string{
		"top level":  strings.Replace(goodDeclaration, `"load_cap": 4,`, `"load_cap": 4, "loadcap_typo": 1,`, 1),
		"in a cell":  strings.Replace(goodDeclaration, `{"id": "host-big", "route": "host"}`, `{"id": "host-big", "route": "host", "fallback": "host"}`, 1),
		"in a task":  strings.Replace(goodDeclaration, `"kind": "check",`, `"kind": "check", "seeded": true,`, 1),
		"in repeats": strings.Replace(goodDeclaration, `"max": 20`, `"max": 20, "min": 1`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseDeclaration([]byte(doc)); err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("parseDeclaration = %v; want an unknown-field refusal", err)
			}
		})
	}
}

func TestADeclarationThatRepeatsAKeyIsRefused(t *testing.T) {
	doc := strings.Replace(goodDeclaration, `"load_cap": 4,`, `"load_cap": 4, "Load_Cap": 64,`, 1)
	_, err := parseDeclaration([]byte(doc))
	var dk *jsonstrict.DuplicateKeyError
	if !errors.As(err, &dk) {
		t.Fatalf("parseDeclaration = %v; want a duplicate-key refusal", err)
	}
	if _, err := parseDeclaration([]byte(goodDeclaration + "{}")); !errors.Is(err, jsonstrict.ErrTrailing) {
		t.Fatalf("a trailing document = %v; want ErrTrailing", err)
	}
}

func TestADeclarationPastItsSizeBoundIsRefusedUnread(t *testing.T) {
	// Not JSON at all past the bound: the bound refuses before any parse.
	big := make([]byte, MaxDeclarationBytes+1)
	for i := range big {
		big[i] = '{'
	}
	_, err := parseDeclaration(big)
	if err == nil || !strings.Contains(err.Error(), "bytes") {
		t.Fatalf("parseDeclaration of %d bytes = %v; want the size refusal", len(big), err)
	}
}

func TestADeclarationWithAPilotOutsideThreeToFiveIsRefused(t *testing.T) {
	for _, pilot := range []string{`"pilot": 2`, `"pilot": 6`, `"pilot": 0`} {
		doc := strings.Replace(goodDeclaration, `"pilot": 3`, pilot, 1)
		if _, err := parseDeclaration([]byte(doc)); err == nil || !strings.Contains(err.Error(), "pilot") {
			t.Errorf("%s: parseDeclaration = %v; want a pilot refusal", pilot, err)
		}
	}
	doc := strings.Replace(goodDeclaration, `"max": 20`, `"max": 4`, 1)
	doc = strings.Replace(doc, `"pilot": 3`, `"pilot": 5`, 1)
	if _, err := parseDeclaration([]byte(doc)); err == nil || !strings.Contains(err.Error(), "max") {
		t.Errorf("max below the pilot: %v; want a refusal", err)
	}
}

// Each refusal names what is wrong; none admits a declaration the later steps
// would have to second-guess.
func TestADeclarationIsHeldToItsShape(t *testing.T) {
	cases := map[string]struct{ from, to, want string }{
		"schema":              {`"schema_version": 1`, `"schema_version": 2`, "schema_version"},
		"no name":             {`"name": "routing check"`, `"name": ""`, "name"},
		"unknown kind":        {`"kind": "check"`, `"kind": "essay"`, "kind"},
		"code without intent": {`, "intent": "itd-2610161733316518"`, ``, "intent"},
		"bad intent":          {`"intent": "itd-2610161733316518"`, `"intent": "iss-1"`, "intent"},
		"writing no criteria": {`, "criteria": "accurate against the source first"`, ``, "criteria"},
		"writing no source":   {`"source": ["docs/guide.md"], `, ``, "source"},
		"check no material":   {`, "material": ["docs/reference"]`, ``, "material"},
		"field of other kind": {`"kind": "check", "material"`, `"kind": "check", "criteria": "x", "material"`, "criteria"},
		"path escapes":        {`["docs/guide.md"]`, `["../secrets"]`, "path"},
		"absolute path":       {`["docs/guide.md"]`, `["/etc/passwd"]`, "path"},
		"duplicate task":      {`"id": "doc-page"`, `"id": "feature-a"`, "twice"},
		"bad task id":         {`"id": "doc-page"`, `"id": "Doc Page"`, "id"},
		"duplicate cell":      {`"id": "local-rev"`, `"id": "host-big"`, "twice"},
		"no builders": {`{"id": "host-big", "route": "host"},
    {"id": "or-cheap", "route": "builder=runner:opencode@openrouter/vendor/model", "settings": {"temperature": 0.6}}`, ``, "builder"},
		"empty route":      {`"route": "host"`, `"route": ""`, "route"},
		"control in route": {`"route": "host"`, `"route": "ho\u001bst"`, "route"},
		"zero load cap":    {`"load_cap": 4`, `"load_cap": 0`, "load_cap"},
		"zero concurrency": {`"concurrency": 2`, `"concurrency": 0`, "concurrency"},
		"cap unit":         {`"unit": "usd"`, `"unit": "eur"`, "unit"},
		"cap amount":       {`"amount": 25`, `"amount": 0`, "amount"},
		"too many repeats": {`"max": 20`, `"max": 100000`, "max"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(goodDeclaration, c.from) {
				t.Fatalf("fixture lacks %q", c.from)
			}
			doc := strings.Replace(goodDeclaration, c.from, c.to, 1)
			_, err := parseDeclaration([]byte(doc))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("parseDeclaration = %v; want a refusal naming %q", err, c.want)
			}
		})
	}
}
