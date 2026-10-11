---
schema_version: 1
id: "iss-2608261312157721"
slug: "unreleased-hand-entry-vs-derived-cut"
severity: "major"
category: "process"
source: "user-observation"
found_during: "peer-session-flag"
found_at: "CHANGELOG.md"
resolution: "The hand-written Unreleased entry is removed; its content is carried by the two resolved records it cited (iss-2608261040378346, iss-2608261041020040), which the derived cut reads directly. The merge-time gate half of the finding stays open as future work in the record body."
impact: internal
---

main's Unreleased section carries a hand-written entry the derived cut refuses. Commit 3dc7aba3 (round-7 resolutions, PR #511) added a '### Fixed' entry under '## [Unreleased]' on a branch parallel to 5897a85d, the commit that adopted the derived-changelog rule (Unreleased must be EMPTY or the launch ship ingest refuses); neither author could see the other, and the collision only exists on merged main. As written, the next 'launch ship' refuses the cut. The entry's content is already carried by the two resolved records it cites (iss-2608261040378346, iss-2608261041020040), so the remedy is removing the hand-written entry, not relocating it. Structural half: nothing gates Unreleased emptiness at merge time, so any two parallel branches can recreate this class silently until release; a compose-time or CI check on '## [Unreleased]' being empty is the durable fix (same family as iss-220's compose-time-prevention finding). Rounds 8a/8b (#512/#513) were checked and touch no CHANGELOG.