---
schema_version: 1
id: "iss-255"
slug: "ahoy-install-gitignore-block-hides"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "v0.5.0 cycle ahoy install on abcd-cli"
found_at: "internal/core/ahoy"
resolution: "The public visibility fence narrows its .abcd/ entry — and only that entry — to the local-ephemeral tier whenever .abcd/ holds tracked files; the memory/ snapshot fence is kept. effectiveVisibilityEntries resolves the entry set both apply and the drift check use (no perpetual re-prompt); narrowing needs positive evidence, so a repo whose git cannot be asked and a non-repo directory both keep the declared set; the install receipt says out loud that the committed record tiers remain published. The brief's visibility table (05-internals/03-configuration.md footnote 2) records the boundary. On abcd-cli a fresh install now leaves new record files visible to git. Pinned by TestPublicVisibilityBlockNarrowsWhenRecordTiersAreTracked and TestPublicVisibilityBlockUnchangedWhenNothingTracked."
impact: fix
---

ahoy install writes a .gitignore managed block that ignores /.abcd/ (and /memory/) wholesale — on any repo that commits the durable tiers, including abcd-cli itself, this hides every new record file from git status and makes git add refuse them, silently breaking the record workflow for every session in the clone. Observed live: install on abcd-cli appended the block; new intents/issues stopped appearing as untracked; the write was reverted by hand. Adjacent to iss-176 (config in an ignored path reported NOT ENFORCEABLE) but broader: the block contradicts the three-tier layout the installer itself scaffolds. Detector: ahoy install on a repo whose .abcd/development is tracked must not write an ignore rule covering it; acceptance: install on abcd-cli leaves git status of a new .abcd/work/issues file visible.