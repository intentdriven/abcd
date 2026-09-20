---
schema_version: 1
id: "iss-2609120447480903"
slug: "ahoy-install-writes-githooks-pre-commit-but-leaves-git-confi"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "ahoy-install-onboarding-2026-09-12"
origin: researcher-authored
production_mode: hand-written
---

ahoy install writes .githooks/pre-commit but leaves 'git config core.hooksPath .githooks' to the user, so the hook is committed but not running on the clone that just installed it. Detection knows this (hooks_path_armed=false) yet emits no gap for it. Proposal: a user-state gap that arms the clone after checking .git/hooks has no non-sample hooks the redirect would bypass, refusing with a note when it does.

---

_Relocated from another repository's ledger on 2026-09-15. It was captured by an
`ahoy install` onboarding session whose working directory was a teaching-materials
repository, so the finding landed where nothing could resolve or detect it: that
tree has no installer, no plugin root and no `~/.local/bin` surface. The id,
the `found_during` stamp and the body are unchanged; only the ledger it sits in
has moved. The store resolved correctly — it wrote to the repository it was
standing in — and the reason nothing refused the write is recorded as
iss-2609120511058115._
