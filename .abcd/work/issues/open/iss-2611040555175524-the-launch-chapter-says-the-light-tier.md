---
schema_version: 1
id: "iss-2611040555175524"
slug: "the-launch-chapter-says-the-light-tier"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md"
remedy: "Confine the sentence to explicitly declared paths and name payload_completeness_test.go as the guard for the convention directories."
---

The launch chapter says the light tier fails a file present in the tree but excluded from the payload (.abcd/development/brief/04-surfaces/04-launch.md:438), and it counts the convention register, the command, agent, skill and hook paths a plugin is expected to carry, in the declared surface (lines 423-424). conventionEntries lists only what the payload tree already holds (internal/core/launch/installsurface.go:349), so a page reached by convention but outside the includes yields no entry and no finding. Only a path declared explicitly, by a plugin.json key or a hook command, gives the missing-path finding (internal/core/launch/smoke.go:123-127, internal/core/launch/smoke_test.go:47,53).
