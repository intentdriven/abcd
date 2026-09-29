package lint

// The record_schema family's target leg (itd-2609212103572513 criterion 1,
// spc-2609212138243443 scope 1).
//
// `target_release` names the release a planned intent must land by. The verbs
// write it onto a planned record only and every move out of planned/ drops it,
// so this leg refuses the two things no verb produces: the key on a shipped
// or superseded record, which has nothing left to land, and a value that is
// not a target at all. What a legal value IS is launch.ValidTargetRelease, the
// predicate the verbs and the cut read, because this package cannot import
// the intent store.
//
// A draft or discipline carrying one is not judged here: the criterion names
// the two shelves, and the report lists only planned records, so a target on
// either is inert rather than wrong.

import (
	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/launch"
)

// checkIntentTarget runs the target leg over one scanned record.
func checkIntentTarget(r schemaRecord, severity string) []Finding {
	if r.store.prefix != "itd" {
		return nil
	}
	f, ok := r.fields[launch.TargetReleaseKey]
	if !ok {
		return nil
	}
	line := f.line
	if line == 0 {
		line = 1
	}
	finding := func(msg string) []Finding {
		return []Finding{{File: r.rel, Line: line, RuleID: ruleRecordSchema, Severity: severity, Message: msg}}
	}
	switch r.bucket {
	case "shipped", "superseded":
		return finding(launch.TargetReleaseKey + " on a " + r.bucket + " intent; a target names the release unshipped work must land by, " +
			"and the moves out of planned/ drop it, so remove the line")
	}
	value, isScalar := frontmatter.ScalarString(f.value)
	if !isScalar {
		if frontmatter.IsNull(f.value) {
			return nil
		}
		return finding(launch.TargetReleaseKey + " is not a single-line value; a target is `next` or a release tag vX.Y.Z, " +
			"written by `abcd intent target`")
	}
	if err := launch.ValidTargetRelease(value); err != nil {
		return finding(launch.TargetReleaseKey + ": " + err.Error() + ", written by `abcd intent target`")
	}
	return nil
}
