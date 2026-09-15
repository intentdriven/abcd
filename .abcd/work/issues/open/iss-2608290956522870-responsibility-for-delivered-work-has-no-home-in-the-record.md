---
schema_version: 1
id: "iss-2608290956522870"
slug: "responsibility-for-delivered-work-has-no-home-in-the-record"
severity: "major"
category: "process"
source: "user-observation"
found_during: "role-clarification-run"
found_at: "scripts/check-attribution.sh"
---

Responsibility for delivered work has no home in the record, because the attribution trailer discloses who helped rather than who is answerable. The convention names an assisting tool, which is disclosure and deliberately not authorship, and nothing anywhere records the act that actually carries responsibility: a named person accepting a delivered promise as matching what they asked for. Adding a co-authorship trailer for the tool is the wrong repair and this repository already refuses it, since it asserts an authorship a tool cannot hold and inflates the contributor graph, and the largest project to deliberate the question chose the assisting form over the co-developed one for exactly that reason. The right shape is a separate acceptance trailer naming the human who accepted delivery, which only a person can sign, so an automated facilitator structurally cannot. It belongs on the intent rather than on a commit, because what is accepted is a delivered promise and not a diff, with a mirror on the merge commit if a commit-level trace is wanted. It should carry the verification rung alongside the name, because an acceptance is worth exactly as much as the checking behind it and an acceptance on the cheapest automatic check should not read identically to one that followed an outside audit.

Refined 2026-08-29. Making it a commit trailer reintroduces the ambiguity the attribution convention already rejects, since an absent trailer and a forgotten one are the same bytes, and requiring a negative form on every commit would stamp 'nobody accepted this' on the great majority of commits, which is intermediate work no product thinker will ever accept. The resolution is that acceptance is a field on the intent rather than a trailer on a diff: the field is always present so it cannot be forgotten, and its value is null until someone accepts, which makes the absence of acceptance a statement rather than a silence. The populated form carries who accepted, when, the verification rung the acceptance rested on, and the verdict it was taken against, so an acceptance on the cheapest automatic check does not read identically to one that followed an outside audit. A commit trailer may still mirror it on the merge that ships the intent, where its presence is tied to one event rather than expected everywhere.

