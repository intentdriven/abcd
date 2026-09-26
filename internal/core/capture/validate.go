package capture

import (
	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/issuerecord"
	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// knownFields is the additionalProperties:false allow-list from
// issue.schema.json — the ONE copy in core/issueschema, the same set the record
// lint reads, so the reader and the committed-ledger gate cannot disagree about
// which keys a well-formed record may carry.
var knownFields = issueschema.Known

// The record reader's judgement lives in core/issuerecord, where record-lint
// reads it too; these are this package's names for it.

// validateStrict validates a frontmatter map against the issue schema
// (issuerecord.ValidateStrict).
func validateStrict(fm map[string]any) error { return issuerecord.ValidateStrict(fm) }

// validateInvariants enforces the folder<->field invariants and the
// filename<->id agreement (issuerecord.ValidateInvariants).
func validateInvariants(fm map[string]any, status State, path string) error {
	return issuerecord.ValidateInvariants(fm, statusDirName[status], path)
}

// parseFrontmatterAndBody splits a record into its frontmatter map and body
// (issuerecord.Parse).
func parseFrontmatterAndBody(text string) (map[string]any, string, error) {
	return issuerecord.Parse(text)
}

// acceptedValues renders a closed enum's legal set for a refusal message
// (issuerecord.AcceptedValues).
func acceptedValues(vals []string) string { return issuerecord.AcceptedValues(vals) }

// issueFromFrontmatter builds a typed Issue from a validated frontmatter map.
func issueFromFrontmatter(fm map[string]any, status State, path, body string) Issue {
	iss := Issue{
		SchemaVersion: fm["schema_version"].(int),
		ID:            asString(fm["id"]),
		Slug:          asString(fm["slug"]),
		Severity:      Severity(asString(fm["severity"])),
		Category:      Category(asString(fm["category"])),
		Source:        Source(asString(fm["source"])),
		FoundDuring:   asString(fm["found_during"]),
		FoundAt:       asString(fm["found_at"]),
		LapsedAt:      asString(fm["lapsed_at"]),
		Grounds:       groundsEntries(body),
		Resolution:    asString(fm["resolution"]),
		WontfixReason: asString(fm["wontfix_reason"]),
		Status:        status,
		Path:          path,
		Body:          body,
	}
	iss.RelatedIntents = asStrList(fm["related_intents"])
	iss.RelatedSpecs = asStrList(fm["related_specs"])
	iss.RelatedIssues = asStrList(fm["related_issues"])
	iss.BlockedBy = asStrList(fm["blocked_by"])
	iss.Duplicates = asStrList(fm["duplicates"])
	iss.Refines = asStrList(fm["refines"])
	if rb, ok := fm["resolved_by"].(map[string]any); ok {
		iss.ResolvedBy = &ResolvedBy{
			Intent: asString(rb["intent"]),
			Spec:   asString(rb["spec"]),
			Commit: asString(rb["commit"]),
		}
	}
	return iss
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asStrList(v any) []string {
	l, _ := v.([]string)
	if len(l) == 0 {
		return nil
	}
	return l
}

// groundsEntries reads the record's `## Grounds` section through core/grounds's
// own reader, so what an ENTRY is has one definition and a bullet one writer
// appends is a bullet the other finds.
//
// It asks ParseSection where the intent half asks ParseSectionAboveFloor: the
// two families read the same section by the same rules and part company on the
// FLOOR alone. A wontfix stamps its grounds from a reason whose own contract is
// merely non-empty, so applying the floor here would drop entries the ledger has
// always accepted and leave a surface reporting no recorded grounds about a
// record that visibly carries one.
func groundsEntries(body string) []string {
	entries := grounds.ParseSection(body)
	if len(entries) == 0 {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, g := range entries {
		out = append(out, g.String())
	}
	return out
}

// ReadRefusal reports the error the ledger's read path would skip a committed
// record with, or nil when it reads the record: the same parse, the same strict
// schema validation and the same folder/filename invariants List runs, in that
// order. status is the status directory the record sits in and path its file
// path (only the basename is read).
//
// It exists so record-lint's record_schema gate refuses exactly the records this
// reader refuses, by asking the reader rather than re-deriving its grammar: a
// record the reader skips is invisible to every capture surface while it still
// sits in the ledger, and each private copy of the grammar the gate carried drifted
// from this one somewhere (iss-2608300205044566, iss-2608300244483405). The
// judgement itself is core/issuerecord's, the one the ledger reader runs.
func ReadRefusal(content, status, path string) error {
	fm, _, err := issuerecord.Parse(content)
	if err != nil {
		return err
	}
	if err := issuerecord.ValidateStrict(fm); err != nil {
		return err
	}
	return issuerecord.ValidateInvariants(fm, status, path)
}
