package release

import "reflect"

// inputs.go — `launch ship --changelog-json` as the input registry in the
// front door names it (spc-2610301431075964 step 1): the Go type, the schema
// version (ChangelogSchemaVersion), a worked example and the part of the
// ingest's validation that needs no cut.

// ChangelogPayloadType is the Go type the composed changelog decodes into.
var ChangelogPayloadType = reflect.TypeOf(ChangelogPayload{})

// ChangelogExample is one composed changelog at ChangelogSchemaVersion that
// passes CheckChangelog: a fixes-only cut, so its release page is null.
const ChangelogExample = `{
  "schema_version": 2,
  "prompt_version": "1.0.0",
  "next_tag": "v0.2.1",
  "entries": [
    {"section": "Fixed", "records": ["iss-2609010000000001"], "text": "A capture run from a subdirectory files into the checkout's ledger."}
  ],
  "press_release": null
}`

// CheckChangelog runs the ingest's decoder and every check of the document's
// own lines, and writes nothing: the size cap, the strict decode, the schema
// version, the prompt version, and each entry's section, citations and prose.
// What needs the cut (the next tag it derives, the bijection between the
// lines and the records that shipped, and the release page's set) is judged
// by `launch ship` against the cut it derives.
func CheckChangelog(repoRoot string, raw []byte) error {
	p, perr := decodeChangelogPayload(repoRoot, raw)
	if perr != nil {
		return perr
	}
	var rs reasons
	validateEntries(p.Entries, &rs)
	if len(rs) > 0 {
		return &PayloadRefusal{Reasons: rs}
	}
	return nil
}
