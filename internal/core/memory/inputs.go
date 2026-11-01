package memory

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// inputs.go — the two memory inputs as the input registry in the front door
// names them (spc-2610301431075964 step 1): `library ingest --pages-json`, a
// bare array of distilled pages, and `library ask --page-json`, the one answer
// page a file-back writes. Each has a Go type for its field list, a schema
// version, a worked example held here, and a check that runs the page
// validator the writing command runs, writing nothing.
//
// The pages are validated as key/value maps (validateDistilledPage), not
// decoded into a struct, so the wire shape below is declared for the field
// list alone, and a test holds its member names and requiredness to the
// validator's own key sets.

// PagesSchemaVersion is the version of `library ingest --pages-json`. The
// input is a bare JSON array, which has nowhere to carry a version, so the
// version is the registry's alone.
const PagesSchemaVersion = 1

// AnswerPageSchemaVersion is the version of `library ask --page-json`. The
// page carried none before it existed: an absent schema_version reads as 1,
// and any other version is refused.
const AnswerPageSchemaVersion = 1

// DistilledPageJSON is one distilled page as a caller writes it.
type DistilledPageJSON struct {
	Type   string `json:"type" doc:"the page type, a filename-safe token"`
	Domain string `json:"domain" doc:"the page's domain, a filename-safe token"`
	Slug   string `json:"slug" doc:"the page's slug, a filename-safe token"`
	Body   string `json:"body" doc:"the page text, non-empty; its first line is the subject the topic hash reads"`
	// Source is left out by a caller: the binary supplies the source block it
	// computed from the ingested source, or from the matched pages a file-back
	// cites, and validates the page with it.
	Source      map[string]any `json:"source,omitempty" doc:"left out, the binary supplies the source block it computed"`
	Contradicts []string       `json:"contradicts,omitempty" doc:"page ids this page contradicts"`
	Recall      []string       `json:"recall,omitempty" doc:"recall keywords"`
}

// AnswerPageJSON is the answer page `library ask --file-back` files.
type AnswerPageJSON struct {
	SchemaVersion int `json:"schema_version,omitempty"`
	DistilledPageJSON
}

// The Go types the two inputs decode into, for the registry's field lists.
var (
	PagesType      = reflect.TypeOf([]DistilledPageJSON{})
	AnswerPageType = reflect.TypeOf(AnswerPageJSON{})
)

// PagesExample is one `library ingest --pages-json` payload that passes
// CheckPages.
const PagesExample = `[
  {"type": "concept", "domain": "release", "slug": "derived-changelog",
   "body": "# The changelog is derived\n\nA release section is composed from the records that reached a terminal folder.",
   "recall": ["changelog", "release cut"]}
]`

// AnswerPageExample is one `library ask --page-json` page at
// AnswerPageSchemaVersion that passes CheckAnswerPage.
const AnswerPageExample = `{
  "schema_version": 1,
  "type": "answer",
  "domain": "release",
  "slug": "who-writes-the-changelog",
  "body": "# Who writes the changelog\n\nNobody by hand: the cut composes it from the shipped records.",
  "recall": ["changelog"]
}`

// DecodePages reads `--pages-json`: one JSON array of page objects, each
// still to be validated by the ingest.
func DecodePages(raw []byte) ([]map[string]any, error) {
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, err
	}
	return arr, nil
}

// DecodeAnswerPage reads `--page-json`: one JSON object, whose
// schema_version, when present, must be AnswerPageSchemaVersion. The version
// is taken off the page it returns, because the page validator closes its
// keys to the page's own.
func DecodeAnswerPage(raw []byte) (map[string]any, error) {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, newSchemaError("the answer page must be one JSON object")
	}
	if v, ok := obj["schema_version"]; ok {
		n, isNum := v.(float64)
		if !isNum || n != AnswerPageSchemaVersion {
			return nil, newSchemaError("the answer page's schema_version is %s, which is not a version this abcd reads; compose it at version %d (an absent schema_version reads as %d)",
				describeVersion(v), AnswerPageSchemaVersion, AnswerPageSchemaVersion)
		}
		delete(obj, "schema_version")
	}
	return obj, nil
}

func describeVersion(v any) string {
	if n, ok := v.(float64); ok {
		return fmt.Sprintf("%v", n)
	}
	return "not a number"
}

// checkSource is the stand-in source block a check validates a page with when
// the page carries none, as the writing command would supply one: a work-notes
// block, which asks for nothing more.
func checkSource() map[string]any { return map[string]any{"class": "work_notes"} }

// checkPage runs the page validator the writing command runs.
func checkPage(repoRoot string, page map[string]any) error {
	if _, ok := page["source"]; !ok && page != nil {
		merged := make(map[string]any, len(page)+1)
		for k, v := range page {
			merged[k] = v
		}
		merged["source"] = checkSource()
		page = merged
	}
	_, err := validateDistilledPage(repoRoot, page)
	return err
}

// CheckPages decodes a `--pages-json` payload and runs the ingest's page
// validator over every page, writing nothing. A page with no source block is
// validated with a stand-in, because the ingest supplies the block it computes
// from the source it read; the dedup against the pages already in the store
// is the ingest's.
func CheckPages(repoRoot string, raw []byte) error {
	pages, err := DecodePages(raw)
	if err != nil {
		return err
	}
	if len(pages) == 0 {
		return newSchemaError("the pages payload carries no page")
	}
	for i, p := range pages {
		if err := checkPage(repoRoot, p); err != nil {
			return fmt.Errorf("page %d: %w", i+1, err)
		}
	}
	return nil
}

// CheckAnswerPage decodes a `--page-json` page and runs the file-back's page
// validator over it, writing nothing; a page with no source block is
// validated with a stand-in, as CheckPages does.
func CheckAnswerPage(repoRoot string, raw []byte) error {
	page, err := DecodeAnswerPage(raw)
	if err != nil {
		return err
	}
	return checkPage(repoRoot, page)
}
