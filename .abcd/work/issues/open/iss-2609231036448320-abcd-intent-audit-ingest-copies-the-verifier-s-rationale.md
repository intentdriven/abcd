---
schema_version: 1
id: "iss-2609231036448320"
slug: "abcd-intent-audit-ingest-copies-the-verifier-s-rationale"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
---

abcd intent audit ingest copies the verifier's rationale prose verbatim into the shipped intent's Audit Notes, and that record is lint-bound: a rationale that names a test's illustrative id (an auditor citing TestCheckRefusesADanglingSpecTarget wrote the spec id the fixture dangles to) makes record-lint refuse the whole tree with a prose_citation_resolves BLOCKER on the ingested line, and the verdict schema offers no way to carry the '<!-- record-lint: illustrative -->' marker the rule asks for. The ingest validates the verdict against the rubric but not against the record gates the record it writes must pass, so a valid verdict can produce an uncommittable record; the auditor's only remedy is to re-word and re-ingest. Either the ingest should run the prose-citation check over the rendered block and refuse the verdict with the offending id named, or the verdict shape should let a rationale mark an id as illustrative
