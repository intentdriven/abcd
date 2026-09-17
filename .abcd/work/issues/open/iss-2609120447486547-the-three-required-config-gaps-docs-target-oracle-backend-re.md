---
schema_version: 1
id: "iss-2609120447486547"
slug: "the-three-required-config-gaps-docs-target-oracle-backend-re"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "ahoy-install-onboarding-2026-09-12"
origin: researcher-authored
production_mode: hand-written
---

The three required config gaps (docs.target, oracle.backend, repo.visibility) say 'ahoy install prompts for the value', but the flags --docs-target/--oracle-backend/--visibility exist and are the only reliable way to answer in a piped run. The fix_hint should name the flag; and a --yes run that still has to prompt for a free-text config value should say so up front rather than blocking on stdin.

---

_Relocated from another repository's ledger on 2026-09-15. It was captured by an
`ahoy install` onboarding session whose working directory was a teaching-materials
repository, so the finding landed where nothing could resolve or detect it: that
tree has no installer, no plugin root and no `~/.local/bin` surface. The id,
the `found_during` stamp and the body are unchanged; only the ledger it sits in
has moved. The store resolved correctly — it wrote to the repository it was
standing in — and the reason nothing refused the write is recorded as
iss-2609120511058115._
