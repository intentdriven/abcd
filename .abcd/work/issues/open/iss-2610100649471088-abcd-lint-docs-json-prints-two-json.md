---
schema_version: 1
id: "iss-2610100649471088"
slug: "abcd-lint-docs-json-prints-two-json"
severity: "minor"
category: "inconsistency"
source: "managed-repo"
found_during: "abcd inbox report rpt-2610071636212229 from a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0)"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd lint docs --json, abcd lint --json"
remedy: "none (filed automatically)"
---

abcd lint docs --json prints two JSON documents on a blocker, and names its finding fields differently from abcd lint

A lab prototype that runs abcd's checks on every save parsed both outputs and hit two shape differences:

1. **Two documents.** With a blocker finding, abcd lint docs --json writes the result object (findings, blockers, checks, documents) and then a second object (abcd: error, lint docs: 1 blocker finding(s), exit_code 1) to stdout. A standard JSON parse of stdout fails with "Extra data"; a caller has to decode only the first document.
2. **Field casing.** abcd lint --json findings use file, ruleId, severity, message, fix; abcd lint docs --json findings use File, Line, RuleID, Severity, Message. A caller filtering by file on one shape silently matches nothing on the other.

Also: with no findings, abcd lint docs --json writes "findings": null rather than an empty list.

Remedy the reporter proposes: Print one JSON document per run (the refusal folded into the result object, or the result folded into the error object), and use one field spelling for findings across lint targets (file, line, ruleId, severity, message).

Reported by a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0) through the abcd inbox as rpt-2610071636212229, a defect against abcd v0.13.1, surface abcd lint docs --json, abcd lint --json.

Evidence:

- rpt-2610071636212229 (the report, kept in the inbox)
- rpt-2610071632391737
