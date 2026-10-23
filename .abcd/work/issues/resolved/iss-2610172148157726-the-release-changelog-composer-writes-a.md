---
schema_version: 1
id: "iss-2610172148157726"
slug: "the-release-changelog-composer-writes-a"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "triaging the v0.15.1 release review's changelog records"
origin: researcher-authored
production_mode: hand-written
found_at: "agents/release-changelog-composer.md"
remedy: "Have the composer's brief, and the ship path's check, treat a fix record whose surface is absent at the anchor tag as present behaviour: word it as what the surface does, or fold it into that surface's Added line, never as a fix"
resolution: "The cut marks a record whose found_at is a path the anchor tag did not hold (found_at_new) and lists the commands the anchor's surface lacked (guard.new_commands); the ingest refuses a Fixed line citing a found_at_new record, and the composer folds such a fix into the surface's Added line"
impact: fix
---

The release changelog composer writes a fix to a surface that did not exist at the anchor tag as a Fixed line saying the behaviour 'no longer' happens, so a release's notes describe a fix to something no earlier release had. v0.15.1's section has such lines for abcd status and the cloud scripts, all new since v0.15.0, because each record carries impact: fix and nothing ties an entry's section to whether its surface existed at the anchor.

## Grounds

- pursued: a release's Fixed section no longer reports a fix to a surface its anchor lacked where found_at names the surface's path; a Fixed line about a found_at_new record reaching a release would show it wrong, while one whose found_at is words or an older directory stays the composer's judgement
