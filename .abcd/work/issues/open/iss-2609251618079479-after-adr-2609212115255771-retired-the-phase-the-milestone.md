---
schema_version: 1
id: "iss-2609251618079479"
slug: "after-adr-2609212115255771-retired-the-phase-the-milestone"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/04-scope.md"
deferred_after: "v0.11.1"
deferral_reason: "no ruling owed; carried past v0.11.1 by lane drainDrift3 (run A 2026-09-29) on its size: adr-2609212115255771 settles the model (sequencing is builds_on and blocked_by plus the drafts, planned and shipped shelves; the checkpoint is the derived release), but at 8322cdf65 the brief still names the phase at about 250 sites in 60 files, and each needs a reading to tell the retired sequencing unit from a legitimate use (the retired-term glossary entry, the phase-audit receipt, a process phase). It wants a docs lane of its own, taking 01-product/04-scope.md and the release, version, loop, plan and spec glossary entries first."
---

After adr-2609212115255771 retired the phase, the milestone and the word roadmap, the brief outside the mental-model chapter still describes the phase as the live sequencing unit: 59 files under .abcd/development/brief mention it, among them 01-product/04-scope.md (bounded by the planned phases), and the glossary entries release, version, loop, plan and spec, whose prose says a phase sequences the work and a release falls out of completing one. itd-2609211913453478 rewrote only the mental-model chapter and repointed each entry's not_to_be_confused_with; no record carries the rest of the sweep.
