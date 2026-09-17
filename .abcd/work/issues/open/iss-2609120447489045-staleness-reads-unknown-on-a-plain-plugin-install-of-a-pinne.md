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
