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
remedy: "Make abcd lint docs --json write one document whose findings read like abcd lint's: after rendering, return the blocker exit as an empty-message exit error with code 1 (internal/surface/cli/cli.go:955), as lint site and bare lint do (site.go:139, lint.go:76), so no error envelope follows the result; encode findings with the keys file, line, ruleId, severity and message that repolint.Finding uses (internal/core/repolint/repolint.go:68-74), keeping finding_ref and lint's blocker, warn and info values; write an empty list rather than null when there are no findings; and update the key list at commands/lint.md:120-121, resolving with impact breaking since documented keys change; proven by a CLI test, like TestUpdateJSONRefusalIsOneDocument, that over a one-blocker fixture stdout decodes to exactly one JSON value with lowercase finding keys and the process exits 1, and that a clean fixture carries an empty findings list. (All three halves of the report reproduce in the code as it stands; the key rename breaks readers of the documented capitalised keys, which is why the impact is breaking.)"
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
