---
schema_version: 1
id: "iss-2609120447482506"
slug: "path-entry-written-as-a-symlink-to-the-plugin-root-binary-no"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "ahoy-install-onboarding-2026-09-12"
origin: researcher-authored
production_mode: hand-written
wontfix_reason: "duplicate of iss-2609100506263330: the same PATH entry written as a symlink to the plugin-root binary on a cold cache, which a plugin update breaks; its evidence and its two asks are appended to the survivor, and its superseded-root detection half is answered on main by 177a407d (iss-2609161805447092)"
---

PATH entry written as a symlink to the plugin-root binary (no verified artefact in the cache) is fragile by abcd's own note, yet bare 'ahoy' reports zero gaps afterwards. Immediately after a plugin update the symlink still pointed at the superseded root e3696dc while the live root was eda67ccc, and install_mode read 'pinned'. Detection should surface this as a gap (entry points at a non-current plugin root / not an owned copy) rather than only in a one-time install note; better still, provision the cache during install so the owned copy is written first time.

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

- declined: duplicate of iss-2609100506263330: the same PATH entry written as a symlink to the plugin-root binary on a cold cache, which a plugin update breaks; its evidence and its two asks are appended to the survivor, and its superseded-root detection half is answered on main by 177a407d (iss-2609161805447092)
