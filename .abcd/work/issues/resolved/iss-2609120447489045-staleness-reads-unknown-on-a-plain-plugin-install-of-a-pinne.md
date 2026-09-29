---
schema_version: 1
id: "iss-2609120447489045"
slug: "staleness-reads-unknown-on-a-plain-plugin-install-of-a-pinne"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "ahoy-install-onboarding-2026-09-12"
origin: researcher-authored
production_mode: hand-written
resolution: "Does not reproduce at the base or on a pinned release: a v0.9.0 plugin install, run from a terminal with no plugin environment, reports staleness 'up to date' in both abcd version and abcd ahoy --json. What the record asks for is what vintageFrom does (internal/core/ahoy/vintage.go:84-89): the stamped version is compared with the release tag the bootstrap recorded, and a match is Fresh. The terminal reaches that record through the plugin root's .data-dir stamp (6e05c4494, in every release since v0.8.0); pinned by TestVintageFromPinnedInstall and TestReadPinnedTagPrefersRootThenCache. Unknown remains only where no record answers, which is the honest reading."
impact: internal
---

staleness reads 'unknown' on a plain plugin install of a pinned release, every run. For a plugin user that is noise, not signal: the skill text tells the agent to report staleness so a stale binary is never silent, so 'unknown' gets relayed on every healthy render. Either resolve it to 'current' when the pinned version matches the plugin manifest, or suppress it for pinned installs.

---

_Relocated from another repository's ledger on 2026-09-15. It was captured by an
`ahoy install` onboarding session whose working directory was a teaching-materials
repository, so the finding landed where nothing could resolve or detect it: that
tree has no installer, no plugin root and no `~/.local/bin` surface. The id,
the `found_during` stamp and the body are unchanged; only the ledger it sits in
has moved. The store resolved correctly — it wrote to the repository it was
standing in — and the reason nothing refused the write is recorded as
iss-2609120511058115._

## Grounds

- pursued: a pinned install whose binary matches its recorded release tag reports up to date; a healthy pinned install reporting unknown from a terminal or a session would show it wrong
