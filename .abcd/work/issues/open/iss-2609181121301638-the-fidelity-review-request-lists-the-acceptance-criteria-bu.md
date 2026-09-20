---
schema_version: 1
id: "iss-2609181121301638"
slug: "the-fidelity-review-request-lists-the-acceptance-criteria-bu"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "Gropius managed-repo session gropiusllm-56, seven fidelity audits at v0.9.0, relayed to abcd-17 on 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
---

The fidelity review request lists the acceptance criteria but never the scope-condition identities the verdict must dispose. auditPromptBody in internal/core/intent/audit.go writes an Acceptance Criteria block numbered ac-1..ac-K and the rubric, while the verdict schema (verdictCondition, keyed on condition_id) and the intent-auditor contract require scope_conditions to cover the intent's conditions exactly, one disposition per cond-… identity; the identities reach the auditor only as HTML comments in the intent record that the request does not quote. Relayed from the Gropius managed-repo session gropiusllm-56 on 2026-09-18 after seven fidelity audits at v0.9.0: the reviewer scraped the cond comments by hand, and a miscount quarantined the verdict at ingest. Wanted: the request carries a Scope Conditions block listing each cond-… identity with its text, as it carries the criteria, so the auditor disposes exactly the set the ingest will check. The prompt body is hashed for provenance, so the block is part of the pure composition and the prompt_hash policy version moves with it.

**Addendum (2026-09-19, Gropius session gropiusllm-66, relayed to abcd-17).**
The same request also says "numbered ac-1..ac-K in order" and never prints K.
A lane briefed its auditor with the wrong count; the auditor counted the
bullets itself and was right. Printing K beside the block, and the cond-…
identities this record asks for, are one change to the same composer.
