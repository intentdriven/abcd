---
schema_version: 1
id: "iss-2609181121305984"
slug: "the-fidelity-review-request-carries-no"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "Gropius managed-repo session gropiusllm-56, seven fidelity audits at v0.9.0, relayed to abcd-17 on 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
resolution: "The request carries a Verdict shape section rendered by reflection from the verdict struct the ingest decodes with DisallowUnknownFields, so it is the schema rather than a copy, and the ingest's --verdict-json help names the section. The dry-run validator the addendum offers is the alternative remedy the record names as sufficient either way, and is not added."
impact: fix
resolved_by:
  commit: "d3a4bd667760f980f44e549c1a2dbafe122021fd"
---

The fidelity review request carries no verdict-JSON schema and names no place to read one. auditPromptBody ends with "Run the intent-auditor agent over the criteria and the delivered diff, then ingest its verdict JSON", and abcd intent audit ingest --help documents a single flag; the concrete verdict shape (verdict enum, scope_conditions with condition_id and disposition, the two sha256 provenance fields) lives only in the bundled intent-auditor agent definition. A host without that agent, or a reviewer working from the request alone, has nothing to write against and learns the shape from ingest refusals. Relayed from the Gropius managed-repo session gropiusllm-56 on 2026-09-18 after seven fidelity audits at v0.9.0. Wanted, either: the request quotes the verdict shape beside the rubric, or a read-only abcd intent audit schema verb prints it, and the ingest help points at whichever exists. Filed apart from the scope-condition identities finding because the remedies differ: that one is about which items the verdict must cover, this one about the shape it must take.

**Addendum (2026-09-19, Gropius session gropiusllm-66, relayed to abcd-17).**
The missing schema has a second cost: with no `--verdict-json` dry-run
validator, every lane in a forty-lane sweep rewrote the same pre-ingest
checker. A validate-only mode on the ingest (parse and shape-check, write
nothing, exit as the ingest would) is the same remedy family as printing the
shape, and either one retires the hand-written checkers.

## Grounds

- pursued: a reviewer working from the request alone writes a verdict the ingest decodes; a field on the verdict struct absent from the stated shape, or a stated field the ingest refuses as unknown, would show it wrong
