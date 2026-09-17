---
schema_version: 1
id: "iss-2609120447494818"
slug: "capture-source-and-category-are-closed-sets-but-neither-help"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "ahoy-install-onboarding-2026-09-12"
origin: researcher-authored
production_mode: hand-written
---

capture --source and --category are closed sets but neither --help nor the refusal names the valid values: 'invalid source "session-observation"' with no list. An agent relaying a capture has to guess or fall back to the default. Every closed-set refusal should print the set, as --production-mode's help text already does.

---

_Relocated from another repository's ledger on 2026-09-15. It was captured by an
`ahoy install` onboarding session whose working directory was a teaching-materials
repository, so the finding landed where nothing could resolve or detect it: that
tree has no installer, no plugin root and no `~/.local/bin` surface. The id,
the `found_during` stamp and the body are unchanged; only the ledger it sits in
has moved. The store resolved correctly — it wrote to the repository it was
standing in — and the reason nothing refused the write is recorded as
iss-2609120511058115._
