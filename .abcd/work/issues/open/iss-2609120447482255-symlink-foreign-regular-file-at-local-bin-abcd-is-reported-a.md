---
schema_version: 1
id: "iss-2609120447482255"
slug: "symlink-foreign-regular-file-at-local-bin-abcd-is-reported-a"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "ahoy-install-onboarding-2026-09-12"
origin: researcher-authored
production_mode: hand-written
---

symlink.foreign (regular file at ~/.local/bin/abcd) is reported as unresolvable with 'resolve manually' but says nothing about what the file is. Reporting size, mtime, and whether it identifies as an abcd binary (and which version) would let the user decide safely instead of inspecting by hand. The removal then needs a second full install run; a re-detect after the user clears it in the same run would save the round-trip.

---

_Relocated from another repository's ledger on 2026-09-15. It was captured by an
`ahoy install` onboarding session whose working directory was a teaching-materials
repository, so the finding landed where nothing could resolve or detect it: that
tree has no installer, no plugin root and no `~/.local/bin` surface. The id,
the `found_during` stamp and the body are unchanged; only the ledger it sits in
has moved. The store resolved correctly — it wrote to the repository it was
standing in — and the reason nothing refused the write is recorded as
iss-2609120511058115._
