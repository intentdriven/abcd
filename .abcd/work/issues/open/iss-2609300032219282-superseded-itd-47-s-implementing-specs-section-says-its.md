---
schema_version: 1
id: "iss-2609300032219282"
slug: "superseded-itd-47-s-implementing-specs-section-says-its"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/superseded/itd-47-oracle-gates-autonomous-mode.md"
remedy: "Rewrite the section to say the ids are the predecessor store's, preserved as history, that the native store reuses both numbers, and that the frontmatter spec_id is null because the intent was superseded by adr-22 without a native spec; qualify each id '(predecessor store)'. Grounds: the frontmatter and superseded_by are the primary record, and iss-2609300025372991's fix is the shape."
refines: [iss-2609300025372991]
---

Superseded itd-47's Implementing specs section says its frontmatter spec_id records spc-27 as the primary delivering spec, while the frontmatter holds spec_id: null, and spc-27 and spc-32 are predecessor-store ids that the live store reuses for the surface-coverage registry and abcd update. The same defect iss-2609300025372991 fixed in itd-36 and itd-4, found in a third intent while qualifying predecessor-store citations for iss-2609290448510918.
