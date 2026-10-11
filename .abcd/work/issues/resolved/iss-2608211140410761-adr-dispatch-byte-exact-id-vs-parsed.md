---
schema_version: 1
id: "iss-2608211140410761"
slug: "adr-dispatch-byte-exact-id-vs-parsed"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-b-round-4"
found_at: "internal/core/record/record.go"
resolution: "describeADR now compares parsed handles (unquote, ^([A-Za-z]+)-0*([0-9]+)$, EqualFold prefix, numeric compare against the routed ordinal) and renders the canonical id in both the Description and the not-found error, mirroring record-lint and recordid. The hostile-leaf guard is preserved (an empty/garbage head matches no handle). Watched-fail: TestDescribeADRIDSpellingsAreOneHandle (quoted/padded/cased/padded-ask)."
impact: fix
---

abcd <id> ADR dispatch (record.go describeADR) confirms the frontmatter id with a byte-exact 'fields["id"].Value != id' compare (frontmatter.Field.Value is TrimSpace-only, quotes not stripped), while record-lint schema.go compares parsed handles (a tested tolerance, TestRecordSchemaFilenameIDComparesNumerically) and the citation resolver rebuilds ids from the parsed integer. So an ADR written id: "adr-12" (quoted), adr-0012 (padded), or ADR-12 (cased) is lint-green and citation-resolvable but abcd adr-12 reports 'not found'. A live facet against the shipped corpus: abcd adr-0003 (padded, admitted by IDRe) routes to the file then refuses. Second site under iss-285's quote-normalisation-split principle.