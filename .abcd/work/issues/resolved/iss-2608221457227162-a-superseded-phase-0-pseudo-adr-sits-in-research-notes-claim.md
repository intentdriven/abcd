---
schema_version: 1
id: "iss-2608221457227162"
slug: "a-superseded-phase-0-pseudo-adr-sits-in-research-notes-claim"
severity: "major"
category: "drift"
source: "user-observation"
found_during: "record-review"
found_at: ".abcd/development/research/notes/01-harness-interface.md"
details: "research/notes/01-harness-interface.md is titled 'ADR-01: Harness Interface Design' and declares 'Status: Accepted (Phase 0 lock)', but it lives in research/notes/, carries no record frontmatter, collides with the real adr-1 (three-layer-mental-model), and describes a Python architecture (harness.py, pluggy, abc.ABC, anthropic.types.Message) that adr-21 superseded on 2026-07-06. Two inbound references point at it and at a sibling that does not exist."
suggested_fix: "Decide the document's status, then make its location and shape say so: either promote it to a real ADR with frontmatter and an explicit superseded_by pointing at adr-21/adr-23, or retitle it as the dated research note it actually is and drop the ADR-01 claim. Repair the two dangling references either way."
related_issues: ["iss-2608230752354926", "iss-2608230752354927"]
resolution: "Already fixed on main by f1644d47: the note's status block says it is Phase 0 evidence stating no current architecture, the ADR-01 title and Accepted status are gone, and both dangling references (the brief's 0001-harness-interface.md example and itd-6's 02-mcpbridge-implementation-contract.md citation) were repaired. cross_store_id_claim (itd-161) runs clean over the note at e076c8e3."
impact: internal
shipped_in: v0.6.2
resolved_by:
  commit: "f1644d47"
---

a superseded Phase-0 pseudo-ADR sits in research/notes/ claiming ADR-01, outside every record gate

`.abcd/development/research/notes/01-harness-interface.md` is shaped like an
accepted decision but filed as evidence, and the mismatch has let a superseded
architecture sit in the record unchallenged. Four defects, one root cause.

1. **It presents as a decision in the evidence folder.** Line 1 is
   `# ADR-01: Harness Interface Design` and it declares
   `Status: Accepted (Phase 0 lock)`. The `research/notes/README.md` "What does
   NOT belong here" list opens with decisions, which belong in
   `.abcd/development/decisions/adrs/`.
2. **The id collides.** The real adr-1 is
   `decisions/adrs/0001-three-layer-mental-model.md`, with `id: adr-1`
   frontmatter. Two documents claim ADR-01.
3. **It carries no frontmatter,** so it has no `id`, no `status`, and no
   `superseded_by`. record-lint's ADR machinery never sees it as a record, which
   is the mechanism behind the staleness: an "accepted lock" describing
   `harness.py`, `pluggy`, `abc.ABC`, and `anthropic.types.Message` passed
   through [adr-21](../../../development/decisions/adrs/0021-rebuild-in-go.md)
   (`rebuild-in-go`, 2026-07-06) untouched, because no gate was watching it.
4. **Two inbound references dangle.** The brief, at
   `05-internals/03-configuration.md:403`, gives the ADR naming convention as
   `NNNN-<slug>.md (e.g. 0001-harness-interface.md)`, citing a filename that
   does not exist; `itd-6` line 73 cites
   "ADR-02 (`02-mcpbridge-implementation-contract.md`)", which exists nowhere in
   the repo.

Why it matters: the document is not held to currency by
[adr-5](../../../development/decisions/adrs/0005-brief-is-current-state.md), because
it is not the brief. The defect is that it reads as normative while sitting
outside every gate that governs normative records, so a reader reasonably takes a
superseded Python design for current architecture. That misreading has already
happened once in review.

Note on discoverability: the harness-name normalisation (PR #442) rewrote 22
occurrences in this file, so its git mtime now suggests active maintenance. The
content is unchanged by that sweep and remains superseded.

## Grounds

- pursued: the note no longer presents as a decision; a record-lint run at e076c8e3 with cross_store_id_claim armed reports nothing on it, and a reintroduced ADR-01 title would show it wrong by tripping that rule
