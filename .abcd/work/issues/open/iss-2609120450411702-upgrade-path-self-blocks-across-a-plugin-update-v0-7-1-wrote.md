---
schema_version: 1
id: "iss-2609120450411702"
slug: "upgrade-path-self-blocks-across-a-plugin-update-v0-7-1-wrote"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "ahoy-install-onboarding-2026-09-12"
origin: researcher-authored
production_mode: hand-written
---

Upgrade path self-blocks across a plugin update: v0.7.1 wrote ~/.local/bin/abcd as a symlink into its own plugin root (e3696dc) with the note 're-run install from a session whose hooks have provisioned the cache to upgrade it to an owned copy'. After the plugin updated to v0.8.0 (eda67ccc), that re-run classified the symlink as symlink.foreign ('expected .../eda67ccc02eb/abcd'), unresolvable, 'ahoy refuses to clobber' — so the prescribed remedy is refused by its own predecessor's write, abcd on PATH stays at v0.7.1, and install_mode reads empty. Only a manual rm and a third install produced the owned copy. Fix: a symlink whose target is inside abcd's own plugin-cache directory (any root hash) is abcd's, not foreign — adopt and replace it; or have the first install never write the plugin-root symlink fallback at all, since it is guaranteed to go stale on the next update. Related: iss-2609120447482506.

---

_Relocated from another repository's ledger on 2026-09-15. It was captured by an
`ahoy install` onboarding session whose working directory was a teaching-materials
repository, so the finding landed where nothing could resolve or detect it: that
tree has no installer, no plugin root and no `~/.local/bin` surface. The id,
the `found_during` stamp and the body are unchanged; only the ledger it sits in
has moved. The store resolved correctly — it wrote to the repository it was
standing in — and the reason nothing refused the write is recorded as
iss-2609120511058115._
