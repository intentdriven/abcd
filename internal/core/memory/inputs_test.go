package memory

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
)

// TestDistilledPageJSONIsTheValidatorsKeys holds the wire shape the field
// list is read from to the key sets the page validator closes a page to, so
// the two cannot drift: the same members, less topic_hash (which the boundary
// computes and refuses), and required exactly where the validator requires a
// key, less source (which the writing command supplies).
func TestDistilledPageJSONIsTheValidatorsKeys(t *testing.T) {
	var names, required []string
	shape, err := surface.NewInputShape("abcd library ingest", "pages-json", "distilled pages", PagesType,
		surface.RootArray, PagesSchemaVersion, surface.VersionNone, PagesExample)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range shape.Fields {
		names = append(names, f.Name)
		if f.Required {
			required = append(required, f.Name)
		}
	}
	var wantNames []string
	for k := range distilledPageKeys {
		if k != "topic_hash" {
			wantNames = append(wantNames, k)
		}
	}
	var wantRequired []string
	for _, k := range distilledPageRequired {
		if k != "source" {
			wantRequired = append(wantRequired, k)
		}
	}
	for _, s := range [][]string{names, required, wantNames, wantRequired} {
		sort.Strings(s)
	}
	if !reflect.DeepEqual(names, wantNames) {
		t.Errorf("DistilledPageJSON declares %v; the validator admits %v", names, wantNames)
	}
	if !reflect.DeepEqual(required, wantRequired) {
		t.Errorf("DistilledPageJSON requires %v; the validator requires %v", required, wantRequired)
	}
}

func TestMemoryInputExamplesPassTheirChecks(t *testing.T) {
	if err := CheckPages(t.TempDir(), []byte(PagesExample)); err != nil {
		t.Errorf("PagesExample: %v", err)
	}
	if err := CheckAnswerPage(t.TempDir(), []byte(AnswerPageExample)); err != nil {
		t.Errorf("AnswerPageExample: %v", err)
	}
	if err := CheckPages(t.TempDir(), []byte(`[{"type": "concept", "domain": "x", "slug": "y", "body": " "}]`)); err == nil {
		t.Error("CheckPages accepted a page with an empty body")
	}
}

// TestAnswerPageReadsAnAbsentVersionAsOne holds `--page-json` to the rule:
// absent reads as 1, 1 is read and taken off the page, and any other version
// is refused.
func TestAnswerPageReadsAnAbsentVersionAsOne(t *testing.T) {
	const body = `"type": "answer", "domain": "release", "slug": "s", "body": "# s"`
	for _, doc := range []string{`{` + body + `}`, `{"schema_version": 1, ` + body + `}`} {
		page, err := DecodeAnswerPage([]byte(doc))
		if err != nil {
			t.Errorf("%s refused: %v", doc, err)
			continue
		}
		if _, ok := page["schema_version"]; ok {
			t.Errorf("%s: the version was left on the page, where the validator refuses it", doc)
		}
	}
	for _, v := range []string{"2", `"1"`} {
		_, err := DecodeAnswerPage([]byte(`{"schema_version": ` + v + `, ` + body + `}`))
		if err == nil || !strings.Contains(err.Error(), "version 1") {
			t.Errorf("schema_version %s: err = %v, want a refusal naming version 1", v, err)
		}
	}
	if _, err := DecodeAnswerPage([]byte(`null`)); err == nil {
		t.Error("a null page was accepted")
	}
}
