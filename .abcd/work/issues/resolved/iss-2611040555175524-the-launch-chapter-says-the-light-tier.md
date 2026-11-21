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
resolution: "The launch chapter's light-tier sentence now covers only paths declared explicitly, by a plugin.json key or a hook command. It says a page reached by convention but left outside the includes yields no finding, and it names payload_completeness_test.go as the guard for the convention directories. A doc test holds the chapter to this wording."
impact: internal
resolved_by:
  commit: "82bfdcf1430593b64259d0c63ab44d4dd9ecd7fb"
---

The launch chapter says the light tier fails a file present in the tree but excluded from the payload (.abcd/development/brief/04-surfaces/04-launch.md:438), and it counts the convention register, the command, agent, skill and hook paths a plugin is expected to carry, in the declared surface (lines 423-424). conventionEntries lists only what the payload tree already holds (internal/core/launch/installsurface.go:349), so a page reached by convention but outside the includes yields no entry and no finding. Only a path declared explicitly, by a plugin.json key or a hook command, gives the missing-path finding (internal/core/launch/smoke.go:123-127, internal/core/launch/smoke_test.go:47,53).

## Grounds

- pursued: the chapter matches smoke.go and conventionEntries: only explicitly declared paths give the missing-path finding. It would be shown wrong if the light tier gave a finding for a convention-directory file outside the includes, or if payload_completeness_test.go stopped guarding those directories.
