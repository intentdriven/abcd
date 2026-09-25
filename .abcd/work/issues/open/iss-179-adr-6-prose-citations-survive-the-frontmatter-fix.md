---
schema_version: 1
id: "iss-179"
slug: "adr-6-prose-citations-survive-the-frontmatter-fix"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "iss-39 record-schema validation"
found_at: ".abcd/development/decisions/adrs"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Restore adr-6 as a deprecated stub, mark unmigrated predecessors another way, or accept the prose as historical narration?"
---

adr-29 and adr-25 still narrate adr-6 in prose, but adr-6 is in no store and is absent from all git history (the ADR set landed in one import commit; 0004/0006/0008/0014-0018 were never migrated, and every one but 0006 is accounted for by a successor's supersedes). adr-29 states in its own body that adr-6's decision stands and that adr-29 does not supersede it, so the supersession vocabulary cannot record it and neither ADR body should be rewritten to paper over it. iss-39 closed the machine-readable half (the two related_adrs entries, and the two brief pages that cited it). The surviving prose sites are adr-29 lines 24, 39, 43, 56, 60, 63, 71 and adr-25 lines 51, 80; adr-29's line 43 in particular still says it "links to ADR-6" now that the frontmatter link is gone, which is the residual made legible rather than hidden.

What adr-6 WAS is recoverable from the commissioned review, so a future reader does not start from zero: `.abcd/work/reviews/2026-07-06-plan-consistency/05-link-structure.md` lines 47-49 name the original file (`adrs/adr-6-rp-review-storage-and-architecture.md`) and quote its outbound links, and `03-cross-corpus-consistency.md` line 42 records its disposition — voided by `.abcd/work/DECISIONS.md` lines 8-9 ("no external tools … RepoPrompt … codex"), together with adr-8 and adr-17, the only other two ADRs that decision voids. Both of those have since been pruned by a successor that names them (adr-25 supersedes adr-8, adr-22 supersedes adr-17), which is what leaves adr-6 alone: same voiding decision, no successor record. That review also observes that `adrs/README.md`'s `deprecated` status ("the decision no longer applies but no successor replaces it") is the state this fits and is used zero times, which is a candidate disposition: adr-6 is not superseded, it is voided.

Decide whether the record restores adr-6 as a `deprecated` stub carrying that disposition, marks the unmigrated predecessors some other way, or accepts the prose as historical narration; `record_schema` is frontmatter-scoped and does not read prose, so nothing gates this either way.