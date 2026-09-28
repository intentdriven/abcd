---
schema_version: 1
id: "iss-2608241347321759"
slug: "isnull-misses-trailing-comment-null-form"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "pr-294"
found_at: "internal/core/frontmatter/frontmatter.go"
resolution: "Every field reader now honours a trailing comment and the bare-null spellings the way frontmatter.Fields does: Fields already stripped the comment (iss-2608301744268001) and TestFieldsReadsACommentedNullAsNull pins the null half for every spelling; the sweep fixed issueschema.ParseDisposition (comment and null, TestParseDispositionHonoursCommentAndNull) and lint provenanceValue (null asked of the raw scalar, TestRecordProvenanceQuotedNullIsAValue)."
impact: fix
resolved_by:
  commit: "66398414"
---

frontmatter.Fields trims the captured value but strips nothing after a comment marker, and IsNull compares exact strings, so superseded_by: NULL # no successor is read as the literal string 'NULL # no successor' rather than the null token -- a trailing-comment form valid YAML never resolves as null; follow-up to pr-294 per reviewer (frontmatter self-describes as a line scanner, so decide: trim unquoted trailing comments or document as unsupported) takeover 2026-08-24

## Grounds

- pursued: every reader of a frontmatter value strips a trailing comment and reads a bare null as absent; a record read one way by the board and another by the verb would show it wrong
