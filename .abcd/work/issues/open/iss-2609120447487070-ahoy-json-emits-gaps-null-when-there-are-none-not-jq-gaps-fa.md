---
schema_version: 1
id: "iss-2609120447487070"
slug: "ahoy-json-emits-gaps-null-when-there-are-none-not-jq-gaps-fa"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "ahoy-install-onboarding-2026-09-12"
origin: researcher-authored
production_mode: hand-written
---

ahoy --json emits "gaps": null when there are none, not []. jq '.gaps[]' fails on the healthy case, which is the case most scripts will hit. Same for any other list field that can be empty.

---

_Relocated from another repository's ledger on 2026-09-15. It was captured by an
`ahoy install` onboarding session whose working directory was a teaching-materials
repository, so the finding landed where nothing could resolve or detect it: that
tree has no installer, no plugin root and no `~/.local/bin` surface. The id,
the `found_during` stamp and the body are unchanged; only the ledger it sits in
has moved. The store resolved correctly — it wrote to the repository it was
standing in — and the reason nothing refused the write is recorded as
iss-2609120511058115._
