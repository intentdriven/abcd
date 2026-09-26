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
---

After adr-2609212115255771 retired the phase, the milestone and the word roadmap, the brief outside the mental-model chapter still describes the phase as the live sequencing unit: 59 files under .abcd/development/brief mention it, among them 01-product/04-scope.md (bounded by the planned phases), and the glossary entries release, version, loop, plan and spec, whose prose says a phase sequences the work and a release falls out of completing one. itd-2609211913453478 rewrote only the mental-model chapter and repointed each entry's not_to_be_confused_with; no record carries the rest of the sweep.
