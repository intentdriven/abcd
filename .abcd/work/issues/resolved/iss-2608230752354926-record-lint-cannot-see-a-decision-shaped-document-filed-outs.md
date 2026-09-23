---
schema_version: 1
id: "iss-2608230752354926"
slug: "record-lint-cannot-see-a-decision-shaped-document-filed-outs"
severity: "major"
category: "drift"
source: "user-observation"
found_during: "record-review"
found_at: "internal/core/lint/schema.go"
details: "A markdown file anywhere under .abcd/development/ can title itself '# ADR-NN:', declare 'Status: Accepted', carry no frontmatter, and reuse an id that already belongs to a real ADR, and record-lint reports nothing. record_schema enumerates the configured record stores (decisions/adrs, intents, specs); a file outside them is not a record it can see. Confirmed by probe on 2026-08-23: a note claiming the taken id ADR-23 with status 'Accepted (locked)' passed record-lint clean, exit 0, zero findings."
suggested_fix: "Add a rule that flags a record-id claim made outside that id's store: a heading or status block asserting adr-/itd-/spc-/iss-N in a file the corresponding record store does not contain. The id-collision half is the sharp edge and the cheap win; the looser 'reads as a decision but is not one' half can follow. Weigh against the grandfathered undated Phase 0 notes so the rule does not fire on filenames alone."
related_issues: ["iss-2608221457227162"]
related_intents: [itd-161]
resolution: "record-lint's cross_store_id_claim rule flags a decision-shaped document filed outside the record stores that claims an already-taken id, weighing the claim against the record graph rather than against a filename (itd-161)"
impact: internal
resolved_by:
  intent: "itd-161"
  spec: "spc-53"
---

record-lint cannot see a decision-shaped document filed outside the record stores

`record_schema` is scoped to the record stores it is configured with
(`decisions/adrs`, `intents`, `specs`). Anything outside them is not a
malformed record to the engine; it is not a record at all. So the checks that
would catch a decision-shaped document never run on one filed in the wrong
place.

Probe, run 2026-08-23 in a clean worktree. A file
`research/notes/zz-recurrence-probe.md` containing:

```
# ADR-23: Transport Agnostic Core (probe)

## Status

Accepted (locked)
```

reuses an id that a real ADR already holds
([adr-23](../../../development/decisions/adrs/0023-transport-agnostic-core.md)),
declares itself accepted, and carries no frontmatter. `make record-lint` exits
0 with zero findings.

This is the general form of iss-2608221457227162, which is one instance that
reached the record and survived a full architecture change. That instance was
plausibly excused by abcd's own genesis, since the conventions did not exist
when the document was written. The probe removes that excuse: the same defect
authored today, in this repo or in any abcd-managed repo, is equally invisible.

Why the id-collision half matters most: a duplicate id is objectively wrong and
cheap to detect, whereas "this reads like a decision" is a judgement call. A
rule that catches only the collision would have caught the known instance.
