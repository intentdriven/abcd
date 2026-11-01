package reading

import "reflect"

// inputs.go — `reading ingest --reading-json` as the input registry in the
// front door names it (spc-2610301431075964 step 1): the Go type, the schema
// version, a worked example and the part of the ingest's validation that
// needs no parked run.

// OutputSchemaVersion is the version of the reading output's shape. The
// output carries it as the suffix of its _type tag (OutputType), which a test
// holds equal to this constant. It is apart from SchemaVersion, the version
// of the artefacts an assembly writes.
const OutputSchemaVersion = 1

// OutputShape is the Go type the reading output decodes into. Its items are
// position-specific and closed against the position's own body fields at
// ingest, so the field list names them as objects without fields.
var OutputShape = reflect.TypeOf(Output{})

// OutputExample is one reading output at OutputSchemaVersion that passes
// CheckOutput: a detection reading returning one tension.
const OutputExample = `{
  "_type": "abcd.reading.output/1",
  "run_id": "rdg-2609250000000001",
  "position": "detection",
  "regime": "registrative",
  "manifest_sha256": "3333333333333333333333333333333333333333333333333333333333333333",
  "instrument": {
    "model": "example-model-1",
    "definition_sha256": "4444444444444444444444444444444444444444444444444444444444444444",
    "assembler_version": "1"
  },
  "items": [
    {"pattern": "a claim the shipped tree no longer holds",
     "tension": "the brief says every verb has a page; the spec verb has none",
     "constraint_in_play": "wired or it isn't done",
     "why_a_tension": "a verb with no page is reachable from one surface only"}
  ]
}`

// CheckOutput runs the ingest's strict decode and its envelope check, and
// writes nothing: the closed shape, one document, the _type, the run id's
// shape, the position, the manifest hash's shape and a whole instrument
// claim. The run the payload names, its manifest, its definition and every
// item against the position's body fields are judged by the ingest against
// the run an assembly parked.
func CheckOutput(repoRoot string, raw []byte) error {
	out, err := decodeOutput(raw, repoRoot)
	if err != nil {
		return err
	}
	_, err = checkEnvelope(out)
	return err
}
