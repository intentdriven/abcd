---
schema_version: 1
id: "iss-2609261536147903"
slug: "planned-intents-and-disciplines-still-cite-specs-of-the"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned"
resolution: "Each named site was read against the live spec its id collides with, and every one is the predecessor store's: spc-12 (live: disembark grounding; cited as the fidelity reviewer), spc-28 (live: the audit rename; cited as the lifecycle hook), spc-29 (live: lint rename; cited as the Role-2/3 finding owner), spc-31 (live: source ledger; cited as HOLD), spc-33 (live: collision-proof ids; cited as the cleanup sweep), spc-43 (live: worktree name guard; cited as the drainer), spc-52 (live: dangling supersedes; cited as audit_loop_policy tasks), spc-66 (live: ledger assistance; cited as the phase-audit receipt), spc-6 (live: issue capture; cited as the lifecycle-owning spec), spc-8 (live: epic-to-spec terminology; cited as delivering IL002, which no live spec or code carries), spc-17 (live: citations; cited as disembark stubs), and itd-37's illustrative spc-1, spc-3, spc-7. itd-1, itd-24, itd-37, itd-48, itd-50 and itd-53 now carry the '(predecessor store)' qualifier on every such prose citation. The routed_from and bundle frontmatter values are data and are left as they are. No gate reads the qualifier, so none would have caught this. The same pattern reaches about forty further intents (192 sites), captured as iss-2609290448510918 for its own reading."
impact: internal
resolved_by:
  commit: "95fc0b106"
---

Planned intents and disciplines still cite specs of the retired predecessor store unqualified, and such an id at or below spc-70 resolves to a live spec on another subject (spc-12, for one, is live as the disembark grounding spec): itd-48, itd-50 and itd-53 name spc-12, spc-28, spc-29, spc-31, spc-33, spc-43 and spc-52 for the fidelity reviewer, lifecycle hook and cleanup work (which of these are predecessor ids is itself the reading this record owes); itd-24 names spc-66 as the phase-audit receipt; itd-1 and itd-37 name spc-12 as the manual reviewer. The specs charter (Two spc-N Namespaces) rules that every predecessor citation carries the '(predecessor store)' qualifier; iss-239 applied it to the draft corpus only. Each site needs a reading to tell a predecessor citation from a live one, which is why the sweep was not folded into iss-239.

## Grounds

- pursued: no prose citation in the six intents names a colliding spc-N without the qualifier; a census over them printing an unqualified id at or below spc-70 that is not the intent's own would show it wrong
