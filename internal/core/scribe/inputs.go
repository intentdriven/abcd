package scribe

import (
	"fmt"
	"reflect"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// inputs.go — `scribe ingest --scribe-json` as the input registry in the
// front door names it (spc-2610301431075964 step 1): the Go type, the schema
// version, a worked example and the part of the ingest's validation that
// needs no parked context.

// OutputSchemaVersion is the version of the scribe output's shape. The output
// carries it as the suffix of its _type tag (OutputType), which a test holds
// equal to this constant. It is apart from SchemaVersion, the version of the
// context and the manifest an assembly writes.
const OutputSchemaVersion = 1

// OutputShape is the Go type the scribe output decodes into.
var OutputShape = reflect.TypeOf(Output{})

// OutputExample is one scribe output at OutputSchemaVersion that passes
// CheckOutput.
const OutputExample = `{
  "_type": "abcd.scribe.output/1",
  "run": "rdg-2609250000000001",
  "context_sha256": "5555555555555555555555555555555555555555555555555555555555555555",
  "dispositions": [
    {"item": "rdi-2609250000000002", "state": "accepted", "grounds": "pursued: the tension is real and the next reading will show it again",
     "exit_condition": "", "supersedes": "", "recurs": []}
  ],
  "admissions": [],
  "surprises": [],
  "fidelity_flags": [],
  "outstanding": ["rdi-2609250000000003"],
  "refusals": []
}`

// CheckOutput runs the ingest's closed-key walk and strict decode, its _type
// check and its run id's shape, and writes nothing. The context the payload
// cites, the run's committed items and every word carried against the
// researcher's dispositions are judged by the ingest against the context an
// assembly parked.
func CheckOutput(repoRoot string, raw []byte) error {
	out, err := decodeOutput(raw, repoRoot)
	if err != nil {
		return err
	}
	if out.Type != OutputType {
		return fmt.Errorf("scribe: the output's _type is %s, want %q", typeTag(out.Type, OutputType), OutputType)
	}
	if !recordid.ValidReadingRunID(out.Run) {
		return fmt.Errorf("scribe: the output's run is %s, which is not a reading run id (rdg-N)",
			termsafe.DescribeRefused(out.Run))
	}
	return nil
}
