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
resolution: "Answered by 177a407d (iss-2609161805447092): a PATH pin into a sibling plugin-cache vintage whose binary still exists classifies as abcd's own (owned-superseded) rather than foreign, detection raises the required, resolvable symlink.superseded gap, and ahoy install replaces the pin — with the owned copy when an attested cache is present (installOwnedEntry's heal), else by repointing it at the current root. The stranded shape (old root deleted) was already owned before this record. The record's exact scenario is pinned by TestInstallUpgradesSupersededVintagePinToOwnedCopy (0e71f05a), which fails on the tree before 177a407d and passes after. The record's alternative ask, never writing the plugin-root fallback on a cold cache, is held by iss-2609100506263330."
impact: internal
resolved_by:
  commit: "177a407d"
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

## Grounds

- pursued: the prescribed re-run upgrades abcd's own superseded pin to the owned copy with no manual removal; shown wrong if a pin into an older vintage of abcd's cache is refused as foreign or left in place by ahoy install
