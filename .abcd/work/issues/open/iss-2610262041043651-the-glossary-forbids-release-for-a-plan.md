---
schema_version: 1
id: "iss-2610262041043651"
slug: "the-glossary-forbids-release-for-a-plan"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 975 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/distribution/release.md:40"
remedy: "none (filed automatically)"
related_intents: [itd-2610170638184486]
finding_ref: cfk-8adbb75dc36e1e39
---

The glossary forbids 'release' for a plan of what ships together; the checklist draft plans its bundle as 'the next big release'.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/distribution/release.md:40` — "Do not use "release" for a stretch of development work or for a plan of what ships"
- End B: `.abcd/development/intents/drafts/itd-2610170638184486-every-abcd-session-keeps-a-checklist-on.md:302` — "built as the next big release once current work"

release.md says a release is a derived cut, never something work is planned into, and the delivery grouping is a bundle. itd-2610170638184486 says the autonomous-agents bundle is 'built as the next big release', planning work into a release in the sense the glossary forbids.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 975.
